# 频控管理使用指南

## 快速开始

### 1. 配置调整

修改 `config/config.yaml` 中的任务执行频率：

```yaml
scheduler:
  jobs:
    - name: "binance_klines"
      exchange: "binance"
      data_type: "klines"
      cron: "30 */2 * * * *"  # 每2分钟执行一次，避免频控
```

### 2. 交易对数量控制

如果使用大量交易对，建议：

```yaml
exchanges:
  binance:
    data_types:
      klines:
        symbols: ["BTCUSDT", "ETHUSDT", "BNBUSDT"]  # 指定具体交易对
        # 或者使用 ["*"] 但调整执行频率
```

### 3. 运行程序

```bash
go run main.go
```

## 监控频控状态

### 查看日志输出

程序运行时会输出频控相关日志：

```
INFO    开始分批处理    {"total_symbols": 3, "batch_size": 80, "estimated_batches": 1}
DEBUG   处理批次       {"batch_num": 1, "total_batches": 1, "batch_size": 3}
DEBUG   批次处理完成    {"estimated_weight_used": 6, "total_estimated_weight": 6}
```

### 权重使用情况

- `estimated_weight_used`: 本批次估算使用的权重
- `total_estimated_weight`: 累计估算权重
- `usage_percent`: 权重使用百分比

## 常见问题处理

### 1. 网络连接错误

**现象**：
```
WARN    获取权重信息失败，使用本地估算    {"error": "Get \"https://api.binance.com/api/v3/time\": EOF"}
```

**处理**：
- 这是正常的网络波动，系统会自动使用本地估算
- 如果频繁出现，检查网络连接
- 系统会继续正常运行，不影响数据获取

### 2. 权重使用过高

**现象**：
```
INFO    权重使用接近限制，等待下一分钟    {"current_weight": 1100, "max_weight": 1200}
```

**处理**：
- 系统会自动等待到下一分钟
- 考虑减少交易对数量
- 增加任务执行间隔

### 3. 批次处理缓慢

**现象**：批次处理时间过长

**处理**：
- 检查网络连接质量
- 减少单批次的交易对数量
- 检查API响应时间

## 性能优化建议

### 1. 合理设置执行频率

```yaml
# 推荐配置
scheduler:
  jobs:
    - name: "binance_ticker"
      cron: "0 */3 * * * *"    # 每3分钟
    - name: "binance_klines"  
      cron: "30 */2 * * * *"   # 每2分钟，错开30秒
```

### 2. 交易对分组

对于大量交易对，可以创建多个任务：

```yaml
jobs:
  - name: "binance_klines_group1"
    symbols: ["BTCUSDT", "ETHUSDT", "BNBUSDT"]
    cron: "0 */2 * * * *"
  - name: "binance_klines_group2"  
    symbols: ["ADAUSDT", "XRPUSDT", "SOLUSDT"]
    cron: "60 */2 * * * *"  # 错开1分钟
```

### 3. 监控权重使用

定期检查权重使用情况：

```go
// 获取频控状态
status := scheduler.GetRateLimitStatus()
fmt.Printf("权重使用率: %.2f%%\n", status["usage_percent"])
```

## 测试验证

### 运行测试程序

```bash
cd examples
go run test_simple_rate_limit.go
```

### 预期输出

```
INFO    频控管理器状态    {"batch_size":80,"current_weight":0,"usage_percent":0}
INFO    开始分批处理     {"total_symbols": 5, "batch_size": 80}
INFO    批量处理完成     {"total_duration": "5.5s", "symbols_per_second": 0.9}
```

## 故障排除

### 1. 编译错误

确保所有依赖已安装：
```bash
go mod tidy
go mod download
```

### 2. 配置文件错误

检查YAML格式：
```bash
# 验证配置文件
go run examples/test_config_loading.go
```

### 3. 网络连接问题

测试网络连接：
```bash
curl -I https://api.binance.com/api/v3/time
```

## 最佳实践

1. **渐进式部署**：先用少量交易对测试，确认无误后再扩展
2. **监控告警**：设置权重使用率告警（建议80%）
3. **日志分析**：定期分析频控日志，优化配置
4. **备用方案**：准备多个API密钥轮换使用
5. **错峰执行**：不同数据类型错开执行时间

## 技术支持

如遇到问题，请检查：
1. 日志文件中的错误信息
2. 网络连接状态
3. 配置文件格式
4. API密钥有效性

参考文档：
- [频控优化实现文档](rate_limit_improvement.md)
- [Binance API文档](https://binance-docs.github.io/apidocs/)
