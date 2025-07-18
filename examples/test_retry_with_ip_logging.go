package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mooyang-code/data-miner/internal/exchanges/binance"
)

func main() {
	fmt.Println("Testing Binance REST API retry logic with IP logging...")

	// 创建Binance REST API客户端
	api := binance.NewRestAPI()

	// 初始化
	if err := api.Initialize(nil); err != nil {
		log.Fatalf("Failed to initialize Binance API: %v", err)
	}
	defer api.Close()

	// 注意：详细日志由内部控制

	// 创建上下文，设置较短的超时来触发重试
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 测试一个可能会失败的API调用（使用无效的路径来触发错误）
	fmt.Println("\n=== Testing with invalid endpoint to trigger retries ===")
	var result interface{}
	err := api.SendHTTPRequest(ctx, "/api/v3/invalid_endpoint", &result)
	if err != nil {
		fmt.Printf("Expected failure occurred: %v\n", err)
	} else {
		fmt.Println("Unexpected success!")
	}

	// 测试正常的API调用
	fmt.Println("\n=== Testing normal API call ===")
	var serverTime interface{}
	err = api.SendHTTPRequest(ctx, "/api/v3/time", &serverTime)
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

	fmt.Println("\n=== Test completed ===")
}
