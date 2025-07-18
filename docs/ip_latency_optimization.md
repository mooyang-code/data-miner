# IP管理器网络延迟优化功能

## 概述

IP管理器现在支持网络延迟检测功能，能够自动测量所有解析到的IP地址的网络延迟，并优先选择延迟最低的IP地址进行连接，从而显著提升网络请求的响应速度。

## 功能特性

### 1. 自动延迟检测
- 定期测量所有IP地址的网络延迟
- 支持并发检测，提高检测效率
- 自动标记不可用的IP地址

### 2. 智能IP排序
- 按网络延迟从低到高自动排序IP列表
- 优先选择可用且延迟最低的IP
- 不可用的IP自动排在列表末尾

### 3. 灵活配置
- 可启用/禁用延迟检测功能
- 可配置检测间隔、超时时间和检测端口
- 向后兼容，不影响现有代码

### 4. 实时监控
- 提供详细的延迟信息和可用性状态
- 支持强制执行延迟检测
- 完整的状态报告功能

## 配置选项

```go
type Config struct {
    // 基础配置
    Hostname       string        // 要解析的域名
    UpdateInterval time.Duration // 更新间隔，默认5分钟
    DNSServers     []string      // DNS服务器列表
    DNSTimeout     time.Duration // DNS查询超时时间，默认5秒
    
    // 延迟检测配置
    EnableLatencyCheck   bool          // 是否启用延迟检测，默认true
    LatencyCheckInterval time.Duration // 延迟检测间隔，默认30秒
    LatencyTimeout       time.Duration // 延迟检测超时，默认3秒
    LatencyPort          string        // 用于延迟检测的端口，默认443
}
```

## 使用方法

### 1. 基本使用（启用延迟检测）

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/mooyang-code/data-miner/internal/ipmanager"
)

func main() {
    // 使用默认配置（延迟检测已启用）
    config := ipmanager.DefaultConfig("api.binance.com")
    
    // 创建IP管理器
    manager := ipmanager.New(config)
    
    // 启动管理器
    ctx := context.Background()
    err := manager.Start(ctx)
    if err != nil {
        panic(err)
    }
    defer manager.Stop()
    
    // 等待初始化和延迟检测完成
    time.Sleep(5 * time.Second)
    
    // 获取最佳IP（延迟最低）
    bestIP, latency, err := manager.GetBestIP()
    if err != nil {
        fmt.Printf("获取最佳IP失败: %v\n", err)
    } else {
        fmt.Printf("最佳IP: %s (延迟: %v)\n", bestIP, latency)
    }
    
    // GetCurrentIP() 现在会自动返回延迟最低的IP
    currentIP, err := manager.GetCurrentIP()
    if err != nil {
        fmt.Printf("获取当前IP失败: %v\n", err)
    } else {
        fmt.Printf("当前IP: %s\n", currentIP)
    }
}
```

### 2. 自定义延迟检测配置

```go
config := &ipmanager.Config{
    Hostname:             "api.binance.com",
    UpdateInterval:       5 * time.Minute,
    DNSServers:           []string{"8.8.8.8:53", "1.1.1.1:53"},
    DNSTimeout:           5 * time.Second,
    
    // 自定义延迟检测配置
    EnableLatencyCheck:   true,
    LatencyCheckInterval: 15 * time.Second, // 每15秒检测一次
    LatencyTimeout:       2 * time.Second,  // 2秒超时
    LatencyPort:          "443",            // 使用HTTPS端口
}

manager := ipmanager.New(config)
```

### 3. 禁用延迟检测

```go
config := ipmanager.DefaultConfig("api.binance.com")
config.EnableLatencyCheck = false // 禁用延迟检测

manager := ipmanager.New(config)
// 此时行为与原来完全一致
```

## 新增API方法

### 1. GetBestIP() - 获取最佳IP
```go
bestIP, latency, err := manager.GetBestIP()
```
返回延迟最低的可用IP地址及其延迟时间。

### 2. GetAllIPsWithLatency() - 获取所有IP延迟信息
```go
ipInfos := manager.GetAllIPsWithLatency()
for _, info := range ipInfos {
    fmt.Printf("IP: %s, 延迟: %v, 可用: %v\n", 
        info.IP, info.Latency, info.Available)
}
```

### 3. ForceLatencyCheck() - 强制延迟检测
```go
manager.ForceLatencyCheck()
```
立即执行一次延迟检测，不等待定时器。

## 状态信息

GetStatus() 方法现在包含延迟相关信息：

```go
status := manager.GetStatus()
fmt.Printf("延迟检测启用: %v\n", status["latency_check_enabled"])

if latencyInfo, ok := status["latency_info"]; ok {
    // 详细的延迟信息
    for _, info := range latencyInfo.([]map[string]interface{}) {
        fmt.Printf("IP: %s, 延迟: %s, 可用: %v\n",
            info["ip"], info["latency"], info["available"])
    }
}
```

## 工作原理

### 1. 延迟检测机制
- 使用TCP连接测试到目标IP的延迟
- 默认连接HTTPS端口(443)，确保测试的是实际服务延迟
- 并发测试所有IP，提高检测效率

### 2. 智能排序算法
- 可用的IP优先排在前面
- 可用IP按延迟从低到高排序
- 不可用IP排在列表末尾

### 3. 自动故障转移
- 当最佳IP不可用时，自动切换到下一个最佳IP
- 定期重新检测，自动恢复可用的IP

## 性能优化建议

### 1. 检测间隔配置
- 对于稳定的服务，可以设置较长的检测间隔（如60秒）
- 对于不稳定的网络环境，可以设置较短的检测间隔（如15秒）

### 2. 超时时间配置
- 根据网络环境调整超时时间
- 国内服务建议1-2秒，国外服务建议3-5秒

### 3. 检测端口选择
- 使用实际服务端口进行检测更准确
- HTTPS服务使用443端口，HTTP服务使用80端口

## 兼容性说明

- 完全向后兼容，现有代码无需修改
- GetCurrentIP() 方法行为增强，但接口不变
- 默认启用延迟检测，可通过配置禁用

## 示例程序

运行示例程序查看延迟检测效果：

```bash
go run examples/ip_latency_example.go
```

## 测试

运行延迟检测相关测试：

```bash
go test ./internal/ipmanager -run TestLatency -v
```
