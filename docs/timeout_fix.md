# 上下文超时问题修复

## 问题描述

在实施频控优化后，出现了新的错误：

```
error   scheduler/scheduler.go:265      批量处理K线数据失败     {"interval": "1m", "error": "batch 2 processing failed: context deadline exceeded"}
error   scheduler/scheduler.go:125      任务执行失败    {"job": "binance_klines", "error": "batch 2 processing failed: context deadline exceeded"}
```

## 问题分析

### 根本原因

1. **调度器超时时间过短**：原来设置为30秒，但批量处理大量交易对需要更长时间
2. **批量处理时间累积**：多个批次 + 多个K线间隔 + 网络延迟 = 总时间超过30秒
3. **权重等待时间**：当权重接近限制时，等待时间可能很长

### 具体场景

当处理大量交易对时：
- 第1批：80个交易对 × 4个间隔 = 320个API调用
- 第2批：剩余交易对 × 4个间隔 = 更多API调用
- 每个API调用：网络延迟 + 处理时间
- 权重检查：可能需要等待到下一分钟

总时间很容易超过30秒的超时限制。

## 修复方案

### 1. 动态超时时间设置

根据数据类型设置不同的超时时间：

```go
func (s *Scheduler) getTimeoutForDataType(dataType string) time.Duration {
    switch types.DataType(dataType) {
    case types.DataTypeKlines:
        return 5 * time.Minute  // K线数据需要更长时间
    case types.DataTypeTicker:
        return 2 * time.Minute  // Ticker数据相对简单
    case types.DataTypeOrderbook:
        return 3 * time.Minute  // Orderbook数据中等复杂度
    case types.DataTypeTrades:
        return 3 * time.Minute  // Trades数据中等复杂度
    default:
        return 2 * time.Minute  // 默认超时时间
    }
}
```

### 2. 限制权重等待时间

避免长时间阻塞：

```go
// 限制最大等待时间，避免长时间阻塞
maxWaitTime := 90 * time.Second
if waitTime > maxWaitTime {
    waitTime = maxWaitTime
}
```

### 3. 分层超时控制

- **任务级别**：5分钟超时（整个任务）
- **API级别**：30秒超时（单个API调用）

```go
// 为单个API调用设置较短的超时时间
apiCtx, apiCancel := context.WithTimeout(ctx, 30*time.Second)
klines, err := exchange.GetKlines(apiCtx, symbol, interval, 100)
apiCancel()
```

### 4. 增强错误处理和监控

```go
// 记录批次处理时间
batchStartTime := time.Now()
if err := processor(batch); err != nil {
    batchDuration := time.Since(batchStartTime)
    r.logger.Error("批次处理失败",
        zap.Int("batch_num", batchNum),
        zap.Duration("batch_duration", batchDuration),
        zap.Error(err))
    return fmt.Errorf("batch %d processing failed: %w", batchNum, err)
}
```

### 5. 上下文检查优化

在关键点检查上下文状态：

```go
// 检查上下文是否已取消
select {
case <-ctx.Done():
    s.logger.Warn("批次处理被取消",
        zap.String("interval", interval),
        zap.Int("processed", i),
        zap.Int("total", len(symbols)))
    return ctx.Err()
default:
}
```

## 修复效果

### 1. 超时时间优化

| 数据类型 | 原超时时间 | 新超时时间 | 提升倍数 |
|---------|-----------|-----------|---------|
| Klines  | 30秒      | 5分钟     | 10倍    |
| Ticker  | 30秒      | 2分钟     | 4倍     |
| Orderbook | 30秒    | 3分钟     | 6倍     |
| Trades  | 30秒      | 3分钟     | 6倍     |

### 2. 处理能力提升

- **支持更多交易对**：可以处理数百个交易对而不超时
- **更好的容错性**：单个API失败不影响整批处理
- **详细的监控**：记录每批次的处理时间和成功率

### 3. 稳定性改善

- **避免级联失败**：超时不会导致整个调度器停止
- **渐进式处理**：即使部分失败，已处理的数据仍然有效
- **智能重试**：下次调度时会重新尝试

## 测试验证

### 测试场景

创建了 `test_timeout_fix.go` 来验证修复效果：

```go
// 测试大量交易对（100个）
testSymbols := []string{
    "BTCUSDT", "ETHUSDT", "BNBUSDT", // ... 100个交易对
}

// 测试多个K线间隔
intervals := ["1m", "5m", "1h", "1d"]
```

### 预期结果

- ✅ 任务不再因超时而失败
- ✅ 批次处理时间被正确记录
- ✅ 部分失败不影响整体进度
- ✅ 权重管理正常工作

## 最佳实践建议

### 1. 合理配置交易对数量

```yaml
# 建议配置
exchanges:
  binance:
    data_types:
      klines:
        symbols: ["BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "XRPUSDT"]
        # 或者分组处理大量交易对
```

### 2. 调整任务执行频率

```yaml
scheduler:
  jobs:
    - name: "binance_klines"
      cron: "0 */3 * * * *"  # 每3分钟执行，给足够的处理时间
```

### 3. 监控关键指标

- 批次处理时间
- 成功率统计
- 超时发生频率
- 权重使用情况

### 4. 分时段处理

对于大量交易对，可以分时段处理：

```yaml
jobs:
  - name: "klines_group_1"
    symbols: ["BTCUSDT", "ETHUSDT", ...]
    cron: "0 */5 * * * *"
  - name: "klines_group_2"
    symbols: ["ADAUSDT", "XRPUSDT", ...]
    cron: "150 */5 * * * *"  # 错开2.5分钟
```

## 总结

通过这次修复，解决了频控优化后出现的超时问题：

1. **根本解决**：将超时时间从30秒提升到5分钟
2. **分层控制**：任务级和API级分别设置超时
3. **智能等待**：限制权重等待的最大时间
4. **完善监控**：详细记录处理时间和成功率
5. **容错处理**：部分失败不影响整体进度

这些改进确保了系统在处理大量交易对时的稳定性和可靠性，同时保持了频控管理的有效性。
