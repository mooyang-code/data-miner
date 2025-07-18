# WebSocket集成文档

## 概述

本项目现在支持两种数据获取模式：
1. **定时API拉取模式** - 通过定时任务调用REST API获取数据
2. **WebSocket实时模式** - 通过WebSocket连接实时接收数据

## 配置说明

在 `config/config.yaml` 文件中，通过 `use_websocket` 参数控制数据获取模式：

```yaml
exchanges:
  binance:
    enabled: true
    api_url: "https://api.binance.com"
    websocket_url: "wss://stream.binance.com:9443"
    # 数据获取模式: true=websocket实时模式, false=定时API拉取模式
    use_websocket: false  # 设置为true启用websocket模式
```

## 模式对比

### 定时API拉取模式 (use_websocket: false)

**特点：**
- 通过定时任务调用REST API获取数据
- 数据获取频率由cron表达式控制
- 适合对实时性要求不高的场景
- 网络开销相对较小

**配置示例：**
```yaml
use_websocket: false
scheduler:
  enabled: true
  jobs:
    - name: "binance_ticker"
      exchange: "binance"
      data_type: "ticker"
      cron: "0 * * * * *"  # 每分钟执行
```

### WebSocket实时模式 (use_websocket: true)

**特点：**
- 通过WebSocket连接实时接收数据
- 数据延迟极低，实时性强
- 适合对实时性要求高的场景
- 长连接，网络效率高

**配置示例：**
```yaml
use_websocket: true
data_types:
  ticker:
    enabled: true
    symbols: ["BTCUSDT", "ETHUSDT", "BNBUSDT"]
  trades:
    enabled: true
    symbols: ["BTCUSDT", "ETHUSDT"]
```

## 支持的数据类型

### WebSocket模式支持的订阅类型：

1. **Ticker数据** - 24小时价格统计
   - 订阅格式: `{symbol}@ticker`
   - 示例: `btcusdt@ticker`

2. **交易数据** - 实时成交记录
   - 订阅格式: `{symbol}@trade`
   - 示例: `btcusdt@trade`

3. **K线数据** - 蜡烛图数据
   - 订阅格式: `{symbol}@kline_{interval}`
   - 示例: `btcusdt@kline_1m`

4. **订单簿数据** - 买卖盘深度
   - 订阅格式: `{symbol}@depth{levels}`
   - 示例: `btcusdt@depth20`

## 使用方法

### 1. 启动程序

```bash
# 使用默认配置文件
./bin/data-miner

# 指定配置文件
./bin/data-miner -config config/config.yaml
```

### 2. 查看日志

程序会输出详细的日志信息，包括：
- 连接状态
- 订阅频道
- 接收到的数据
- 错误信息

### 3. 模式切换

只需修改配置文件中的 `use_websocket` 参数，重启程序即可切换模式。

## 故障排除

### WebSocket连接失败

如果WebSocket连接失败，程序会：
1. 记录错误日志
2. 继续运行其他功能
3. 不会启动调度器（避免重复数据获取）

常见问题：
- 网络连接问题
- 防火墙阻止WebSocket连接
- Binance服务器维护

### 解决方案

1. 检查网络连接
2. 尝试切换到API拉取模式
3. 查看Binance官方状态页面

## 开发说明

### 添加新的数据类型支持

1. 在 `websocket.go` 中添加新的处理函数
2. 在 `startBinanceWebsocket` 函数中添加订阅逻辑
3. 更新配置文件结构

### 扩展其他交易所

1. 实现 `ExchangeInterface` 接口
2. 添加WebSocket连接逻辑
3. 更新配置文件和主程序逻辑

## 注意事项

1. WebSocket模式下调度器不会启动，避免重复数据获取
2. 确保配置的交易对在交易所中存在
3. 注意WebSocket连接的稳定性，建议添加重连机制
4. 监控数据接收频率，避免超过处理能力

## 示例输出

### API拉取模式
```
2025-07-18T20:17:49.709+0800	info	使用定时API拉取模式
2025-07-18T20:17:49.709+0800	info	调度器启动成功
2025-07-18T20:17:50.003+0800	info	收到市场数据 {"exchange": "binance", "symbol": "BTCUSDT", "type": "orderbook"}
```

### WebSocket模式
```
2025-07-18T20:17:16.424+0800	info	启动Binance WebSocket模式
2025-07-18T20:17:16.424+0800	info	正在连接Binance WebSocket...
2025-07-18T20:17:16.500+0800	info	WebSocket连接成功
2025-07-18T20:17:16.501+0800	info	订阅WebSocket频道 {"channels": ["btcusdt@ticker", "ethusdt@ticker"]}
```
