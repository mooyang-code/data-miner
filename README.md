# 加密货币数据挖掘器 (Crypto Data Miner)

一个用于从各大加密货币交易所拉取实时市场数据的高性能数据挖掘工具。

## 功能特性

- 🔄 **多交易所支持**: 当前支持Binance，易于扩展其他交易所
- 📊 **多数据类型**: 支持行情(Ticker)、订单簿(Orderbook)、交易(Trades)、K线(Klines)数据
- ⏰ **定时任务**: 基于Cron表达式的灵活调度系统
- 🛡️ **速率限制**: 内置API调用频率控制，避免触及交易所限制
- 📝 **结构化日志**: 使用zap提供高性能结构化日志
- 🔧 **配置驱动**: YAML配置文件，支持热配置更新
- 🏥 **健康检查**: 内置健康检查服务，便于监控

## 项目结构

```
data-miner/
├── config/                 # 配置文件目录
│   └── config.yaml        # 主配置文件
├── internal/              # 内部包
│   ├── exchanges/         # 交易所实现
│   │   └── binance/       # Binance交易所
│   ├── scheduler/         # 任务调度器
│   └── types/             # 类型定义
├── pkg/                   # 公共包
│   └── utils/             # 工具函数
├── main.go                # 程序入口
├── go.mod                 # Go模块文件
└── README.md              # 项目文档
```

## 快速开始

### 1. 编译程序

```bash
go build -o data-miner .
```

### 2. 查看帮助信息

```bash
./data-miner -help
```

### 3. 启动程序

```bash
./data-miner
```

或者指定配置文件：

```bash
./data-miner -config path/to/config.yaml
```

### 4. 查看版本

```bash
./data-miner -version
```

## 配置说明

配置文件使用YAML格式，主要包含以下部分：

### 应用配置
```yaml
app:
  name: "crypto-data-miner"
  version: "1.0.0"
  log_level: "info"  # debug, info, warn, error
```

### 交易所配置
```yaml
exchanges:
  binance:
    enabled: true
    api_url: "https://api.binance.com"
    websocket_url: "wss://stream.binance.com:9443"
    api_key: ""      # 可选，用于需要认证的接口
    api_secret: ""   # 可选
    
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
      
      trades:
        enabled: true
        symbols: ["BTCUSDT", "ETHUSDT"]
        interval: "10s"
      
      klines:
        enabled: true
        symbols: ["BTCUSDT", "ETHUSDT", "BNBUSDT"]
        intervals: ["1m", "5m", "1h", "1d"]
        interval: "1m"
```

### 调度器配置
```yaml
scheduler:
  enabled: true
  max_concurrent_jobs: 10
  
  jobs:
    - name: "binance_ticker"
      exchange: "binance"
      data_type: "ticker"
      cron: "0 * * * * *"  # 每分钟执行
    
    - name: "binance_orderbook"
      exchange: "binance"
      data_type: "orderbook"
      cron: "*/5 * * * * *"  # 每5秒执行
```

### 存储配置
```yaml
storage:
  file:
    enabled: true
    base_path: "./data"
    format: "json"  # json, csv
  
  cache:
    enabled: true
    max_size: 1000
    ttl: "1h"
```

## 支持的数据类型

### 1. Ticker (行情数据)
包含24小时价格变动、成交量等信息。

### 2. Orderbook (订单簿)
包含买卖盘深度数据。

### 3. Trades (交易数据)
包含最新的成交记录。

### 4. Klines (K线数据)
包含指定时间间隔的OHLCV数据。

## Cron表达式说明

支持6位格式的Cron表达式：

```
秒 分 时 日 月 周
```

示例：
- `0 * * * * *` - 每分钟执行
- `*/5 * * * * *` - 每5秒执行
- `0 0 * * * *` - 每小时执行
- `0 0 0 * * *` - 每天执行

## 监控和健康检查

程序内置健康检查服务，默认端口8081：

```bash
curl http://localhost:8081/health
```

## 日志格式

程序使用结构化JSON日志，便于日志分析：

```json
{
  "level": "info",
  "time": "2025-07-16T23:40:39.646+0800",
  "caller": "main.go:55",
  "msg": "启动加密货币数据挖掘器",
  "name": "crypto-data-miner",
  "version": "1.0.0"
}
```

## API限制说明

### Binance API限制
- REST API: 1200 requests/minute
- WebSocket连接: 自动重连机制

程序内置了速率限制功能，会自动控制API调用频率。

## 扩展开发

### 添加新的交易所

1. 在`internal/exchanges/`下创建新的交易所目录
2. 实现`types.ExchangeInterface`接口
3. 在`main.go`中注册新的交易所

### 添加新的数据类型

1. 在`internal/types/data.go`中定义新的数据结构
2. 在交易所实现中添加对应的获取方法
3. 在调度器中添加处理逻辑

## 故障排除

### 常见问题

1. **配置文件不存在**
   ```
   配置文件不存在: config/config.yaml
   ```
   解决方案：确保配置文件存在或使用`-config`参数指定正确路径。

2. **API调用失败**
   ```
   failed to get ticker: Get "https://api.binance.com/api/v3/ticker/24hr": dial tcp: lookup api.binance.com: no such host
   ```
   解决方案：检查网络连接和DNS配置。

3. **速率限制**
   ```
   rate limit exceeded
   ```
   解决方案：调整配置文件中的任务频率或增加API密钥配额。

## 贡献指南

欢迎提交Issue和Pull Request来改进项目！

## 许可证

MIT License 