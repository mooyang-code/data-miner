# Binance REST API 动态IP管理

本文档介绍如何在Binance REST API中使用动态IP管理功能，以提高连接稳定性和避免单点故障。

## 功能概述

动态IP管理功能为Binance REST API提供以下特性：

- **自动DNS解析**: 定期解析`api.binance.com`域名获取最新的IP地址列表
- **故障转移**: 当某个IP地址不可用时，自动切换到下一个可用IP
- **负载均衡**: 在多个IP地址之间分配请求负载
- **透明集成**: 对现有API调用完全透明，无需修改业务代码

## 工作原理

1. **DNS解析**: IP管理器使用多个DNS服务器（8.8.8.8、1.1.1.1、208.67.222.222）解析域名
2. **IP缓存**: 将解析到的IP地址缓存在内存中，定期更新（默认5分钟）
3. **自定义拨号**: 使用自定义的`DialContext`在TCP连接级别替换IP地址
4. **TLS验证**: 保持原始域名用于TLS证书验证，确保安全性

## 使用方法

### 基本使用

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/mooyang-code/data-miner/internal/exchanges/binance"
    "github.com/mooyang-code/data-miner/internal/types"
    "github.com/mooyang-code/data-miner/pkg/cryptotrader/currency"
)

func main() {
    // 创建Binance REST API客户端
    api := binance.NewRestAPI()
    
    // 初始化（会自动启动IP管理器）
    config := types.BinanceConfig{}
    err := api.Initialize(config)
    if err != nil {
        log.Fatalf("初始化失败: %v", err)
    }
    defer api.Close()
    
    // 等待IP管理器启动
    time.Sleep(time.Second * 3)
    
    // 正常使用API
    ctx := context.Background()
    pair := currency.NewPair(currency.BTC, currency.USDT)
    
    price, err := api.GetLatestSpotPrice(ctx, pair)
    if err != nil {
        log.Fatalf("获取价格失败: %v", err)
    }
    
    fmt.Printf("BTC/USDT 价格: $%.2f\n", price.Price)
}
```

### 启用详细日志

```go
api := binance.NewRestAPI()
api.Verbose = true // 启用详细日志，可以看到IP切换过程
```

### 检查IP管理器状态

```go
status := api.GetIPManagerStatus()
fmt.Printf("IP管理器状态: %+v\n", status)
```

状态信息包含：
- `running`: IP管理器是否正在运行
- `hostname`: 管理的域名
- `current_ip`: 当前使用的IP地址
- `all_ips`: 所有可用的IP地址列表
- `ip_count`: 可用IP数量
- `current_index`: 当前IP在列表中的索引
- `dns_servers`: 使用的DNS服务器列表
- `update_interval`: IP更新间隔

## 配置选项

### IP管理器配置

IP管理器使用默认配置，但可以通过修改`ipmanager.DefaultConfig()`来自定义：

```go
config := ipmanager.Config{
    Hostname:       "api.binance.com",
    UpdateInterval: 5 * time.Minute,  // IP更新间隔
    DNSServers:     []string{         // DNS服务器列表
        "8.8.8.8:53",
        "1.1.1.1:53", 
        "208.67.222.222:53",
    },
    DNSTimeout: 5 * time.Second,      // DNS查询超时
}
```

## 故障处理

### 自动重试机制

当API请求失败时，系统会自动：

1. 切换到下一个可用IP地址
2. 重试请求（默认最多3次）
3. 记录详细的错误日志

### 手动IP切换

```go
// 获取当前IP
currentIP, err := api.getCurrentIP()
if err != nil {
    log.Printf("获取当前IP失败: %v", err)
}

// 切换到下一个IP
nextIP, err := api.switchToNextIP()
if err != nil {
    log.Printf("切换IP失败: %v", err)
}
```

## 性能考虑

### 连接复用

HTTP客户端配置了连接池以提高性能：

```go
Transport: &http.Transport{
    MaxIdleConns:    100,
    IdleConnTimeout: 90 * time.Second,
}
```

### DNS缓存

- IP地址缓存在内存中，避免频繁DNS查询
- 定期更新确保IP地址的时效性
- 使用多个DNS服务器提高解析成功率

## 监控和调试

### 日志级别

- `Verbose = true`: 显示详细的请求和IP切换日志
- `Verbose = false`: 只显示错误和警告日志

### 状态监控

定期检查IP管理器状态以监控系统健康状况：

```go
ticker := time.NewTicker(time.Minute * 5)
defer ticker.Stop()

for range ticker.C {
    status := api.GetIPManagerStatus()
    if !status["running"].(bool) {
        log.Println("警告: IP管理器未运行")
    }
    
    ipCount := status["ip_count"].(int)
    if ipCount == 0 {
        log.Println("警告: 没有可用的IP地址")
    }
}
```

## 注意事项

1. **TLS证书验证**: 系统保持使用原始域名进行TLS验证，确保安全性
2. **DNS依赖**: 需要可靠的DNS服务器访问
3. **网络环境**: 在某些网络环境中可能需要调整DNS服务器配置
4. **资源清理**: 确保在程序退出时调用`api.Close()`以正确清理资源

## 示例程序

完整的示例程序位于 `examples/binance_dynamic_ip_example.go`，演示了：

- 基本API调用
- 订单簿数据获取
- K线数据获取
- 压力测试
- 状态监控

运行示例：

```bash
go run examples/binance_dynamic_ip_example.go
```

## 故障排除

### 常见问题

1. **403错误**: 可能是IP被限制，系统会自动切换到其他IP
2. **DNS解析失败**: 检查网络连接和DNS服务器配置
3. **连接超时**: 可能需要调整超时设置

### 调试步骤

1. 启用详细日志：`api.Verbose = true`
2. 检查IP管理器状态
3. 查看系统日志中的错误信息
4. 验证网络连接和DNS解析

通过动态IP管理，Binance REST API的稳定性和可靠性得到了显著提升，特别是在网络环境不稳定或面临IP限制的情况下。
