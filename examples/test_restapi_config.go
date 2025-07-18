// Package main 直接测试restapi.go中的HTTP客户端配置
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/mooyang-code/data-miner/internal/exchanges/httpclient"
	"github.com/mooyang-code/data-miner/internal/ipmanager"
)

func main() {
	fmt.Println("=== 测试合并后的HTTP客户端配置（直接测试）===")
	
	// 测试1: 手动创建Binance配置（模拟restapi.go中的配置）
	fmt.Println("\n1. 测试Binance专用配置")
	testBinanceConfig()
	
	fmt.Println("\n=== 测试完成 ===")
}

// createBinanceHTTPConfig 创建Binance专用的HTTP客户端配置（从restapi.go复制）
func createBinanceHTTPConfig() *httpclient.Config {
	config := httpclient.DefaultConfig("binance")

	// 启用动态IP
	config.DynamicIP.Enabled = true
	config.DynamicIP.Hostname = "api.binance.com"
	config.DynamicIP.IPManager = ipmanager.DefaultConfig("api.binance.com")

	// 调整重试配置
	config.Retry.MaxAttempts = 5
	config.Retry.InitialDelay = time.Second
	config.Retry.MaxDelay = 8 * time.Second

	// 调整速率限制（Binance限制）
	config.RateLimit.RequestsPerMinute = 1200

	// 启用调试日志
	config.Debug = false

	return config
}

// testBinanceConfig 测试Binance配置
func testBinanceConfig() {
	// 使用合并后的配置逻辑
	config := createBinanceHTTPConfig()
	client, err := httpclient.New(config)
	if err != nil {
		fmt.Printf("创建Binance配置客户端失败: %v\n", err)
		return
	}
	defer client.Close()
	
	// 等待IP管理器启动
	fmt.Println("等待IP管理器启动...")
	time.Sleep(3 * time.Second)
	
	// 测试请求
	ctx := context.Background()
	
	// 测试1: 获取服务器时间
	var timeResp struct {
		ServerTime int64 `json:"serverTime"`
	}
	
	err = client.Get(ctx, "https://api.binance.com/api/v3/time", &timeResp)
	if err != nil {
		fmt.Printf("获取服务器时间失败: %v\n", err)
		return
	}
	
	serverTime := time.Unix(timeResp.ServerTime/1000, 0)
	fmt.Printf("服务器时间: %s\n", serverTime.Format("2006-01-02 15:04:05"))
	
	// 测试2: 获取BTC价格
	var priceResp struct {
		Symbol string `json:"symbol"`
		Price  string `json:"price"`
	}
	
	err = client.Get(ctx, "https://api.binance.com/api/v3/ticker/price?symbol=BTCUSDT", &priceResp)
	if err != nil {
		fmt.Printf("获取BTC价格失败: %v\n", err)
		return
	}
	
	fmt.Printf("BTC价格: %s USDT\n", priceResp.Price)
	
	// 显示客户端状态
	status := client.GetStatus()
	fmt.Printf("\n客户端状态:\n")
	fmt.Printf("  名称: %s\n", status.Name)
	fmt.Printf("  运行状态: %v\n", status.Running)
	fmt.Printf("  总请求数: %d\n", status.TotalRequests)
	fmt.Printf("  成功请求数: %d\n", status.SuccessRequests)
	fmt.Printf("  失败请求数: %d\n", status.FailedRequests)
	fmt.Printf("  重试次数: %d\n", status.RetryCount)
	
	if status.RateLimit != nil {
		fmt.Printf("  速率限制: %d/分钟，剩余: %d\n", 
			status.RateLimit.RequestsPerMinute, 
			status.RateLimit.Remaining)
	}
	
	if status.IPManager != nil {
		ipStatus := status.IPManager
		if running, ok := ipStatus["running"].(bool); ok && running {
			fmt.Printf("  IP管理器: 运行中\n")
			if currentIP, ok := ipStatus["current_ip"].(string); ok {
				fmt.Printf("  当前IP: %s\n", currentIP)
			}
			if ipCount, ok := ipStatus["ip_count"].(int); ok {
				fmt.Printf("  可用IP数量: %d\n", ipCount)
			}
		} else {
			fmt.Printf("  IP管理器: 未运行\n")
		}
	}
	
	fmt.Println("\n✓ Binance专用配置测试成功")
	fmt.Println("✓ HTTP客户端配置合并成功")
}
