# HTTP客户端模块

这是一个通用的HTTP请求客户端模块，专为加密货币交易所API设计，支持动态IP管理、智能重试机制和速率限制。

## 功能特性

### 🌐 动态IP管理
- **自动IP切换**: 集成IP管理器，支持自动IP轮换
- **故障转移**: 请求失败时自动切换到下一个可用IP
- **延迟优化**: 支持基于延迟的IP选择
- **透明代理**: 外部无需了解复杂的IP管理策略

### 🔄 智能重试机制
- **指数退避**: 使用指数退避算法控制重试间隔
- **错误分类**: 智能识别可重试和不可重试的错误
- **可配置策略**: 支持自定义重试次数、延迟和退避因子
- **上下文感知**: 支持context取消和超时

### ⚡ 速率限制
- **内置限流**: 支持每分钟请求数限制
- **动态重置**: 自动重置计数器
- **状态监控**: 实时监控速率限制状态

### 📊 状态监控
- **详细统计**: 请求成功率、失败率、重试次数等
- **实时状态**: IP管理器状态、速率限制状态
- **错误追踪**: 记录最后的错误信息

## 快速开始

### 基本使用

```go
package main

import (
    "context"
    "fmt"
    "github.com/mooyang-code/data-miner/internal/exchanges/httpclient"
)

func main() {
    // 创建基本HTTP客户端
    config := httpclient.DefaultConfig("my-client")
    client, err := httpclient.New(config)
    if err != nil {
        panic(err)
    }
    defer client.Close()

    // 发送GET请求
    var result map[string]interface{}
    err = client.Get(context.Background(), "https://httpbin.org/get", &result)
    if err != nil {
        fmt.Printf("请求失败: %v\n", err)
        return
    }

    fmt.Printf("请求成功: %v\n", result["url"])
}
```

### 启用动态IP的配置

```go
// 创建启用动态IP的配置
config := httpclient.DefaultConfig("my-client")

// 启用动态IP
config.DynamicIP.Enabled = true
config.DynamicIP.Hostname = "api.binance.com"
config.DynamicIP.IPManager.Hostname = "api.binance.com"

// 配置重试策略
config.Retry.MaxAttempts = 5
config.Retry.InitialDelay = time.Second
config.Retry.MaxDelay = 10 * time.Second

// 配置速率限制
config.RateLimit.RequestsPerMinute = 1200

// 创建客户端
client, err := httpclient.New(config)
```

### 高级用法

```go
// 自定义请求
req := &httpclient.Request{
    Method: "POST",
    URL:    "https://api.example.com/data",
    Headers: map[string]string{
        "Authorization": "Bearer token",
    },
    Body: map[string]interface{}{
        "symbol": "BTCUSDT",
        "limit":  100,
    },
    Options: &httpclient.RequestOptions{
        MaxRetries:      3,
        EnableDynamicIP: true,
        Verbose:         true,
    },
}

response, err := client.DoRequest(context.Background(), req)
if err != nil {
    fmt.Printf("请求失败: %v\n", err)
    return
}

fmt.Printf("状态码: %d\n", response.StatusCode)
fmt.Printf("使用IP: %s\n", response.IP)
fmt.Printf("响应时间: %v\n", response.Duration)
```

## 配置选项

### 基本配置
- `Name`: 客户端名称
- `UserAgent`: 用户代理字符串
- `Timeout`: 请求超时时间
- `Debug`: 是否启用调试日志

### 动态IP配置
- `DynamicIP.Enabled`: 是否启用动态IP
- `DynamicIP.Hostname`: 目标主机名
- `DynamicIP.IPManager`: IP管理器配置

### 重试配置
- `Retry.Enabled`: 是否启用重试
- `Retry.MaxAttempts`: 最大重试次数
- `Retry.InitialDelay`: 初始延迟时间
- `Retry.MaxDelay`: 最大延迟时间
- `Retry.BackoffFactor`: 退避因子

### 速率限制配置
- `RateLimit.Enabled`: 是否启用速率限制
- `RateLimit.RequestsPerMinute`: 每分钟最大请求数

### 传输配置
- `Transport.MaxIdleConns`: 最大空闲连接数
- `Transport.MaxIdleConnsPerHost`: 每个主机最大空闲连接数
- `Transport.MaxConnsPerHost`: 每个主机最大连接数
- `Transport.IdleConnTimeout`: 空闲连接超时
- `Transport.TLSHandshakeTimeout`: TLS握手超时
- `Transport.ResponseHeaderTimeout`: 响应头超时

## 错误处理

模块提供了智能的错误分类和处理：

### 错误类型
- `ErrorTypeNetwork`: 网络错误（可重试）
- `ErrorTypeTimeout`: 超时错误（可重试）
- `ErrorTypeTLS`: TLS错误（可重试）
- `ErrorTypeHTTP`: HTTP错误（部分可重试）
- `ErrorTypeRateLimit`: 速率限制错误（可重试）

### 可重试错误
- 网络连接错误
- 超时错误
- TLS握手错误
- HTTP 5xx错误
- HTTP 429错误（速率限制）

### 不可重试错误
- HTTP 4xx错误（除429外）
- 请求格式错误
- 认证错误

## 状态监控

```go
status := client.GetStatus()

fmt.Printf("客户端状态:\n")
fmt.Printf("  名称: %s\n", status.Name)
fmt.Printf("  运行状态: %v\n", status.Running)
fmt.Printf("  总请求数: %d\n", status.TotalRequests)
fmt.Printf("  成功请求数: %d\n", status.SuccessRequests)
fmt.Printf("  失败请求数: %d\n", status.FailedRequests)
fmt.Printf("  重试次数: %d\n", status.RetryCount)

// 速率限制状态
if status.RateLimit != nil {
    fmt.Printf("  速率限制: %d/分钟\n", status.RateLimit.RequestsPerMinute)
    fmt.Printf("  剩余配额: %d\n", status.RateLimit.Remaining)
}

// IP管理器状态
if status.IPManager != nil {
    fmt.Printf("  IP管理器: 运行中\n")
    fmt.Printf("  当前IP: %s\n", status.IPManager["current_ip"])
    fmt.Printf("  可用IP数量: %d\n", status.IPManager["ip_count"])
}
```

## 最佳实践

### 1. 使用专用客户端
对于特定的交易所，建议使用专用的客户端创建函数：
```go
// 推荐：使用专用客户端
client, err := httpclient.NewBinanceClient()

// 而不是手动配置
config := httpclient.DefaultConfig("binance")
config.DynamicIP.Enabled = true
config.DynamicIP.Hostname = "api.binance.com"
client, err := httpclient.New(config)
```

### 2. 合理设置重试策略
```go
config.Retry.MaxAttempts = 5        // 适中的重试次数
config.Retry.InitialDelay = time.Second  // 1秒初始延迟
config.Retry.MaxDelay = 8 * time.Second  // 8秒最大延迟
```

### 3. 监控客户端状态
定期检查客户端状态，特别是在生产环境中：
```go
go func() {
    ticker := time.NewTicker(time.Minute)
    defer ticker.Stop()
    
    for range ticker.C {
        status := client.GetStatus()
        if status.FailedRequests > 0 {
            log.Printf("HTTP客户端错误率: %.2f%%", 
                float64(status.FailedRequests)/float64(status.TotalRequests)*100)
        }
    }
}()
```

### 4. 优雅关闭
确保在程序退出时正确关闭客户端：
```go
defer func() {
    if err := client.Close(); err != nil {
        log.Printf("关闭HTTP客户端失败: %v", err)
    }
}()
```

## 与原有代码的对比

### 原有方式
```go
// 复杂的IP管理和重试逻辑分散在各处
func (b *BinanceRestAPI) sendHTTPRequestWithRetry(...) error {
    for attempt := 0; attempt < maxRetries; attempt++ {
        err := b.doHTTPRequest(...)
        if err == nil {
            return nil
        }
        
        // 手动IP切换逻辑
        nextIP, switchErr := b.switchToNextIP()
        // 手动重试逻辑
        time.Sleep(time.Second * 2)
    }
}
```

### 新的方式
```go
// 简洁的接口，复杂逻辑被封装
func (b *BinanceRestAPI) SendHTTPRequest(ctx context.Context, path string, result interface{}) error {
    fullURL := spotAPIURL + path
    return b.httpClient.Get(ctx, fullURL, result)
}
```

## 扩展性

模块设计为高度可扩展：

1. **新的交易所**: 只需创建专用配置函数
2. **新的重试策略**: 实现RetryHandler接口
3. **新的错误类型**: 扩展ErrorType枚举
4. **新的传输层**: 自定义Transport配置

这个模块将HTTP请求的复杂性完全封装，为其他交易所模块提供了统一、简洁的接口。

## 架构设计原则

### 单一职责原则
HTTP客户端模块只负责通用的HTTP请求功能：
- 动态IP管理
- 重试机制
- 速率限制
- 错误处理
- 状态监控

**不包含**任何特定交易所的业务逻辑。

### 交易所专用配置
每个交易所在自己的模块中创建专用的HTTP客户端配置：

```go
// internal/exchanges/binance/restapi.go
func NewHTTPClient() (httpclient.Client, error) {
    config := createBinanceHTTPConfig()
    return httpclient.New(config)
}

func createBinanceHTTPConfig() *httpclient.Config {
    config := httpclient.DefaultConfig("binance")
    config.DynamicIP.Enabled = true
    config.DynamicIP.Hostname = "api.binance.com"
    config.RateLimit.RequestsPerMinute = 1200
    return config
}

// internal/exchanges/okx/restapi.go
func NewHTTPClient() (httpclient.Client, error) {
    config := createOKXHTTPConfig()
    return httpclient.New(config)
}

func createOKXHTTPConfig() *httpclient.Config {
    config := httpclient.DefaultConfig("okx")
    config.DynamicIP.Enabled = true
    config.DynamicIP.Hostname = "www.okx.com"
    config.RateLimit.RequestsPerMinute = 600
    return config
}
```

### 清晰的模块边界
- **httpclient模块**: 通用HTTP功能
- **交易所模块**: 交易所特定的配置和业务逻辑
- **应用层**: 使用交易所模块提供的接口

这种设计确保了：
1. 代码的可维护性
2. 模块的可复用性
3. 清晰的职责分离
4. 易于测试和扩展
