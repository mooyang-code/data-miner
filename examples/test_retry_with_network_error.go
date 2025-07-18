package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/mooyang-code/data-miner/internal/exchanges/binance"
)

func main() {
	fmt.Println("Testing Binance REST API retry logic with network errors...")

	// 创建Binance REST API客户端
	api := binance.NewRestAPI()

	// 初始化
	if err := api.Initialize(nil); err != nil {
		log.Fatalf("Failed to initialize Binance API: %v", err)
	}
	defer api.Close()

	// 创建上下文，设置较短的超时来可能触发超时错误
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	fmt.Println("\n=== Testing with very short timeout to potentially trigger timeout errors ===")
	var result interface{}
	err := api.SendHTTPRequest(ctx, "/api/v3/exchangeInfo", &result)
	if err != nil {
		fmt.Printf("Expected timeout or error occurred: %v\n", err)
	} else {
		fmt.Println("Request succeeded despite short timeout!")
	}

	// 测试正常的API调用
	fmt.Println("\n=== Testing normal API call with longer timeout ===")
	ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel2()
	
	var serverTime interface{}
	err = api.SendHTTPRequest(ctx2, "/api/v3/time", &serverTime)
	if err != nil {
		fmt.Printf("Server Time API failed: %v\n", err)
	} else {
		fmt.Println("Server Time API succeeded!")
		fmt.Printf("Server Time Response: %+v\n", serverTime)
	}

	// 获取IP管理器状态
	fmt.Println("\n=== IP Manager Status ===")
	status := api.GetIPManagerStatus()
	fmt.Printf("IP Manager Status: %+v\n", status)

	// 测试模拟网络错误的情况
	fmt.Println("\n=== Testing with simulated network error ===")
	testNetworkError()

	fmt.Println("\n=== Test completed ===")
}

func testNetworkError() {
	// 这个函数演示了如何模拟网络错误
	// 在实际应用中，网络错误会自动触发重试逻辑
	
	// 模拟连接超时
	conn, err := net.DialTimeout("tcp", "192.0.2.1:80", 1*time.Second) // 使用测试用的IP地址
	if err != nil {
		fmt.Printf("Simulated network error (this would trigger retry): %v\n", err)
	} else {
		conn.Close()
		fmt.Println("Unexpected connection success")
	}
	
	// 模拟HTTP客户端超时
	client := &http.Client{
		Timeout: 100 * time.Millisecond, // 非常短的超时
	}
	
	_, err = client.Get("https://httpbin.org/delay/1") // 这个URL会延迟1秒响应
	if err != nil {
		fmt.Printf("Simulated HTTP timeout error (this would trigger retry): %v\n", err)
	} else {
		fmt.Println("Unexpected HTTP success")
	}
}
