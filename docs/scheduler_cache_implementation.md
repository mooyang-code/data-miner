# Scheduler Cache 配置实现

## 概述

本文档描述了如何在调度器(Scheduler)中实现从cache配置中读取交易对的功能。

## 实现的功能

### 1. 配置驱动的交易对获取

调度器现在可以从配置文件中读取交易对信息，支持以下配置方式：

```yaml
exchanges:
  binance:
    data_types:
      ticker:
        symbols: ["BTCUSDT", "ETHUSDT", "BNBUSDT"]  # 指定具体交易对
      klines:
        symbols: ["*"]  # 使用通配符从cache获取所有交易对
```

### 2. Cache集成

当配置中使用`["*"]`通配符时，调度器会：
- 从Binance的TradablePairsCache中获取所有可用交易对
- 支持实时更新的交易对列表
- 自动处理缓存过期和刷新

### 3. 配置参数支持

调度器现在从配置中读取以下参数：
- **交易对列表**: 支持具体指定或通配符
- **订单簿深度**: 从配置中读取depth参数
- **K线时间间隔**: 从配置中读取intervals数组

## 代码变更

### 1. Scheduler结构体更新

```go
type Scheduler struct {
    cron      *cron.Cron
    logger    *zap.Logger
    exchanges map[string]types.ExchangeInterface
    callback  types.DataCallback
    jobs      map[string]*JobInfo
    mutex     sync.RWMutex
    config    *types.Config // 新增配置字段
}
```

### 2. 构造函数更新

```go
func New(logger *zap.Logger, exchanges map[string]types.ExchangeInterface, 
         callback types.DataCallback, config *types.Config) *Scheduler
```

### 3. 核心方法实现

#### getSymbolsForExchange
- 从配置中读取对应交易所和数据类型的symbols
- 支持"*"通配符，从cache中获取所有可用交易对
- 包含错误处理和日志记录

#### getBinanceSymbols
- 根据数据类型获取对应的配置symbols
- 处理通配符逻辑

#### getTradablePairsFromCache
- 从Binance的TradablePairsCache中获取交易对
- 支持类型断言和错误处理
- 自动转换currency.Pairs到types.Symbol

#### getDepthForExchange
- 从配置中读取订单簿深度参数

#### getIntervalsForExchange
- 从配置中读取K线时间间隔数组

## 配置示例

### 基本配置
```yaml
exchanges:
  binance:
    data_types:
      ticker:
        enabled: true
        symbols: ["BTCUSDT", "ETHUSDT", "BNBUSDT"]
        interval: "1m"
      
      orderbook:
        enabled: true
        symbols: ["BTCUSDT", "ETHUSDT"]
        depth: 20
        interval: "5s"
      
      klines:
        enabled: true
        symbols: ["*"]  # 从cache获取所有交易对
        intervals: ["1m", "5m", "1h", "1d"]
        interval: "1m"
```

### Cache配置
```yaml
exchanges:
  binance:
    tradable_pairs:
      fetch_from_api: true
      update_interval: "1h"
      cache_enabled: true
      cache_ttl: "2h"
      supported_assets: ["spot", "margin"]
      auto_update: true
```

## 使用方法

### 1. 创建调度器
```go
// 加载配置
config, err := system.LoadConfig("config.yaml")
if err != nil {
    log.Fatal("加载配置失败:", err)
}

// 创建调度器
sched := scheduler.New(logger, exchanges, dataCallback, config)
```

### 2. 启动交易对缓存（如果使用通配符）
```go
// 启动Binance交易对缓存
ctx := context.Background()
if err := binanceExchange.StartTradablePairsCache(ctx); err != nil {
    logger.Error("启动交易对缓存失败", zap.Error(err))
}
```

### 3. 添加和启动任务
```go
// 添加任务
for _, job := range config.Scheduler.Jobs {
    if err := sched.AddJob(job); err != nil {
        logger.Error("添加任务失败", zap.Error(err))
    }
}

// 启动调度器
if err := sched.Start(); err != nil {
    logger.Fatal("启动调度器失败", zap.Error(err))
}
```

## 测试

运行测试示例：
```bash
cd examples
go run scheduler_cache_test.go
```

测试将验证：
- 配置文件加载
- 交易对从配置和cache中获取
- 调度器任务执行
- 错误处理和日志记录

## 注意事项

1. **Cache依赖**: 使用"*"通配符时，需要确保TradablePairsCache已启动
2. **配置验证**: 建议在启动前验证配置文件的正确性
3. **错误处理**: 所有方法都包含适当的错误处理和日志记录
4. **性能考虑**: Cache获取是异步的，首次获取可能需要一些时间

## 扩展性

该实现支持：
- 添加新的交易所支持
- 扩展配置参数
- 自定义交易对过滤逻辑
- 集成其他数据源
