// Package main 演示如何使用通用HTTP客户端模块
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mooyang-code/data-miner/internal/exchanges/httpclient"
)

func main() {
	fmt.Println("=== HTTP客户端模块使用示例 ===")
	
	// 示例1: 创建通用HTTP客户端（启用动态IP）
	fmt.Println("\n1. 创建通用HTTP客户端")
	config := httpclient.DefaultConfig("example")
	config.DynamicIP.Enabled = true
	config.DynamicIP.Hostname = "api.binance.com"
	config.DynamicIP.IPManager.Hostname = "api.binance.com"

	binanceClient, err := httpclient.New(config)
	if err != nil {
		log.Fatalf("创建HTTP客户端失败: %v", err)
	}
	defer binanceClient.Close()
	
	// 测试Binance API
	testBinanceAPI(binanceClient)
	
	// 示例2: 创建自定义客户端（不启用动态IP）
	fmt.Println("\n2. 创建自定义HTTP客户端")
	customClient, err := httpclient.NewCustomClient("custom", "", false)
	if err != nil {
		log.Fatalf("创建自定义客户端失败: %v", err)
	}
	defer customClient.Close()
	
	// 示例3: 使用完全自定义配置（不启用动态IP以避免DNS问题）
	fmt.Println("\n3. 使用完全自定义配置")
	config := httpclient.DefaultConfig("test")
	config.DynamicIP.Enabled = false  // 禁用动态IP以避免DNS解析问题
	config.Retry.MaxAttempts = 3
	config.RateLimit.RequestsPerMinute = 60
	config.Debug = false
	
	testClient, err := httpclient.New(config)
	if err != nil {
		log.Fatalf("创建测试客户端失败: %v", err)
	}
	defer testClient.Close()
	
	// 测试HTTP请求
	testHTTPRequests(testClient)
	
	// 示例4: 查看客户端状态
	fmt.Println("\n4. 查看客户端状态")
	showClientStatus(binanceClient)
}

// testBinanceAPI 测试Binance API
func testBinanceAPI(client httpclient.Client) {
	ctx := context.Background()
	
	// 测试获取服务器时间
	fmt.Println("测试Binance服务器时间API...")
	var timeResp struct {
		ServerTime int64 `json:"serverTime"`
	}
	
	err := client.Get(ctx, "https://api.binance.com/api/v3/time", &timeResp)
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return
	}
	
	serverTime := time.Unix(timeResp.ServerTime/1000, 0)
	fmt.Printf("Binance服务器时间: %s\n", serverTime.Format("2006-01-02 15:04:05"))
	
	// 测试获取交易对信息
	fmt.Println("测试Binance交易对信息API...")
	var exchangeInfo struct {
		Timezone   string `json:"timezone"`
		ServerTime int64  `json:"serverTime"`
		Symbols    []struct {
			Symbol string `json:"symbol"`
			Status string `json:"status"`
		} `json:"symbols"`
	}
	
	err = client.Get(ctx, "https://api.binance.com/api/v3/exchangeInfo", &exchangeInfo)
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return
	}
	
	fmt.Printf("交易所时区: %s\n", exchangeInfo.Timezone)
	fmt.Printf("活跃交易对数量: %d\n", len(exchangeInfo.Symbols))
}

// testHTTPRequests 测试各种HTTP请求
func testHTTPRequests(client httpclient.Client) {
	ctx := context.Background()
	
	// 测试GET请求
	fmt.Println("测试GET请求...")
	var getResp map[string]interface{}
	err := client.Get(ctx, "https://httpbin.org/get", &getResp)
	if err != nil {
		fmt.Printf("GET请求失败: %v\n", err)
	} else {
		fmt.Printf("GET请求成功，响应URL: %v\n", getResp["url"])
	}
	
	// 测试POST请求
	fmt.Println("测试POST请求...")
	postData := map[string]interface{}{
		"message": "Hello from HTTP client",
		"time":    time.Now().Unix(),
	}
	
	var postResp map[string]interface{}
	err = client.Post(ctx, "https://httpbin.org/post", postData, &postResp)
	if err != nil {
		fmt.Printf("POST请求失败: %v\n", err)
	} else {
		fmt.Printf("POST请求成功，响应URL: %v\n", postResp["url"])
	}
	
	// 测试自定义请求
	fmt.Println("测试自定义请求...")
	req := &httpclient.Request{
		Method: "PUT",
		URL:    "https://httpbin.org/put",
		Headers: map[string]string{
			"X-Custom-Header": "test-value",
		},
		Body: map[string]string{
			"action": "update",
		},
		Options: &httpclient.RequestOptions{
			MaxRetries:      2,
			EnableDynamicIP: false,
			Verbose:         true,
		},
	}
	
	resp, err := client.DoRequest(ctx, req)
	if err != nil {
		fmt.Printf("自定义请求失败: %v\n", err)
	} else {
		fmt.Printf("自定义请求成功，状态码: %d，使用IP: %s\n", resp.StatusCode, resp.IP)
	}
}

// showClientStatus 显示客户端状态
func showClientStatus(client httpclient.Client) {
	status := client.GetStatus()
	
	fmt.Printf("客户端名称: %s\n", status.Name)
	fmt.Printf("运行状态: %v\n", status.Running)
	fmt.Printf("总请求数: %d\n", status.TotalRequests)
	fmt.Printf("成功请求数: %d\n", status.SuccessRequests)
	fmt.Printf("失败请求数: %d\n", status.FailedRequests)
	fmt.Printf("重试次数: %d\n", status.RetryCount)
	
	if status.RateLimit != nil {
		fmt.Printf("速率限制: %d/分钟，剩余: %d\n", 
			status.RateLimit.RequestsPerMinute, 
			status.RateLimit.Remaining)
	}
	
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
	
	if status.LastError != "" {
		fmt.Printf("最后错误: %s\n", status.LastError)
	}
}
