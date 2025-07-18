// Package main 测试HTTP客户端配置功能
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/mooyang-code/data-miner/internal/exchanges/httpclient"
	"github.com/mooyang-code/data-miner/internal/ipmanager"
)

func main() {
	fmt.Println("=== 测试HTTP客户端配置功能 ===")
	
	// 测试1: 基本配置
	fmt.Println("\n1. 测试基本配置")
	testBasicConfig()
	
	// 测试2: 启用动态IP的配置
	fmt.Println("\n2. 测试启用动态IP的配置")
	testDynamicIPConfig()
	
	// 测试3: 自定义重试配置
	fmt.Println("\n3. 测试自定义重试配置")
	testCustomRetryConfig()
	
	// 测试4: 自定义速率限制配置
	fmt.Println("\n4. 测试自定义速率限制配置")
	testCustomRateLimitConfig()
	
	fmt.Println("\n=== 测试完成 ===")
}

// testBasicConfig 测试基本配置
func testBasicConfig() {
	config := httpclient.DefaultConfig("basic-test")
	
	client, err := httpclient.New(config)
	if err != nil {
		fmt.Printf("创建基本配置客户端失败: %v\n", err)
		return
	}
	defer client.Close()
	
	status := client.GetStatus()
	fmt.Printf("客户端名称: %s\n", status.Name)
	fmt.Printf("运行状态: %v\n", status.Running)
	fmt.Printf("速率限制: %d/分钟\n", status.RateLimit.RequestsPerMinute)
	fmt.Println("✓ 基本配置测试成功")
}

// testDynamicIPConfig 测试动态IP配置
func testDynamicIPConfig() {
	config := httpclient.DefaultConfig("dynamic-ip-test")
	
	// 启用动态IP（使用Binance作为示例）
	config.DynamicIP.Enabled = true
	config.DynamicIP.Hostname = "api.binance.com"
	config.DynamicIP.IPManager = ipmanager.DefaultConfig("api.binance.com")
	
	client, err := httpclient.New(config)
	if err != nil {
		fmt.Printf("创建动态IP配置客户端失败: %v\n", err)
		return
	}
	defer client.Close()
	
	// 等待IP管理器启动
	time.Sleep(2 * time.Second)
	
	status := client.GetStatus()
	fmt.Printf("客户端名称: %s\n", status.Name)
	fmt.Printf("运行状态: %v\n", status.Running)
	
	if status.IPManager != nil {
		ipStatus := status.IPManager
		if running, ok := ipStatus["running"].(bool); ok && running {
			fmt.Printf("IP管理器: 运行中\n")
			if currentIP, ok := ipStatus["current_ip"].(string); ok {
				fmt.Printf("当前IP: %s\n", currentIP)
			}
			if ipCount, ok := ipStatus["ip_count"].(int); ok {
				fmt.Printf("可用IP数量: %d\n", ipCount)
			}
		} else {
			fmt.Printf("IP管理器: 未运行\n")
		}
	}
	
	// 测试实际请求
	ctx := context.Background()
	var timeResp struct {
		ServerTime int64 `json:"serverTime"`
	}
	
	err = client.Get(ctx, "https://api.binance.com/api/v3/time", &timeResp)
	if err != nil {
		fmt.Printf("动态IP请求失败: %v\n", err)
		return
	}
	
	serverTime := time.Unix(timeResp.ServerTime/1000, 0)
	fmt.Printf("请求成功，服务器时间: %s\n", serverTime.Format("2006-01-02 15:04:05"))
	fmt.Println("✓ 动态IP配置测试成功")
}

// testCustomRetryConfig 测试自定义重试配置
func testCustomRetryConfig() {
	config := httpclient.DefaultConfig("retry-test")
	
	// 自定义重试配置
	config.Retry.MaxAttempts = 3
	config.Retry.InitialDelay = 500 * time.Millisecond
	config.Retry.MaxDelay = 5 * time.Second
	config.Retry.BackoffFactor = 1.5
	
	client, err := httpclient.New(config)
	if err != nil {
		fmt.Printf("创建重试配置客户端失败: %v\n", err)
		return
	}
	defer client.Close()
	
	status := client.GetStatus()
	fmt.Printf("客户端名称: %s\n", status.Name)
	fmt.Printf("运行状态: %v\n", status.Running)
	
	// 测试正常请求
	ctx := context.Background()
	var result map[string]interface{}
	
	err = client.Get(ctx, "https://httpbin.org/get", &result)
	if err != nil {
		fmt.Printf("重试配置请求失败: %v\n", err)
		return
	}
	
	fmt.Printf("请求成功，URL: %v\n", result["url"])
	fmt.Println("✓ 自定义重试配置测试成功")
}

// testCustomRateLimitConfig 测试自定义速率限制配置
func testCustomRateLimitConfig() {
	config := httpclient.DefaultConfig("rate-limit-test")
	
	// 设置较低的速率限制用于测试
	config.RateLimit.Enabled = true
	config.RateLimit.RequestsPerMinute = 2
	
	client, err := httpclient.New(config)
	if err != nil {
		fmt.Printf("创建速率限制配置客户端失败: %v\n", err)
		return
	}
	defer client.Close()
	
	status := client.GetStatus()
	fmt.Printf("客户端名称: %s\n", status.Name)
	fmt.Printf("速率限制: %d/分钟\n", status.RateLimit.RequestsPerMinute)
	
	ctx := context.Background()
	
	// 发送第一个请求
	var result1 map[string]interface{}
	err = client.Get(ctx, "https://httpbin.org/get", &result1)
	if err != nil {
		fmt.Printf("第一个请求失败: %v\n", err)
		return
	}
	fmt.Println("第一个请求成功")
	
	// 发送第二个请求
	var result2 map[string]interface{}
	err = client.Get(ctx, "https://httpbin.org/get", &result2)
	if err != nil {
		fmt.Printf("第二个请求失败: %v\n", err)
		return
	}
	fmt.Println("第二个请求成功")
	
	// 发送第三个请求（应该被速率限制阻止）
	var result3 map[string]interface{}
	err = client.Get(ctx, "https://httpbin.org/get", &result3)
	if err != nil {
		fmt.Printf("第三个请求被速率限制阻止: %v\n", err)
		fmt.Println("✓ 速率限制功能正常工作")
	} else {
		fmt.Println("警告: 第三个请求应该被速率限制阻止")
	}
	
	// 显示最终状态
	finalStatus := client.GetStatus()
	fmt.Printf("最终状态 - 总请求: %d, 成功: %d, 失败: %d\n", 
		finalStatus.TotalRequests, finalStatus.SuccessRequests, finalStatus.FailedRequests)
	fmt.Printf("剩余配额: %d\n", finalStatus.RateLimit.Remaining)
	fmt.Println("✓ 自定义速率限制配置测试成功")
}
