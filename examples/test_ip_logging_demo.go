package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mooyang-code/data-miner/internal/exchanges/binance"
)

func main() {
	fmt.Println("Demo: Binance REST API IP logging functionality...")

	// 创建Binance REST API客户端
	api := binance.NewRestAPI()

	// 初始化
	if err := api.Initialize(nil); err != nil {
		log.Fatalf("Failed to initialize Binance API: %v", err)
	}
	defer api.Close()

	// 等待IP管理器发现更多IP
	fmt.Println("Waiting for IP manager to discover IPs...")
	time.Sleep(3 * time.Second)

	// 获取IP管理器状态
	fmt.Println("\n=== Current IP Manager Status ===")
	status := api.GetIPManagerStatus()
	fmt.Printf("IP Manager Status: %+v\n", status)

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 测试正常的API调用 - 这会显示当前使用的IP
	fmt.Println("\n=== Testing normal API calls to show current IP usage ===")
	
	// 第一次调用
	fmt.Println("Making first API call...")
	var serverTime1 interface{}
	err := api.SendHTTPRequest(ctx, "/api/v3/time", &serverTime1)
	if err != nil {
		fmt.Printf("First API call failed: %v\n", err)
	} else {
		fmt.Println("First API call succeeded!")
	}

	// 第二次调用
	fmt.Println("\nMaking second API call...")
	var serverTime2 interface{}
	err = api.SendHTTPRequest(ctx, "/api/v3/time", &serverTime2)
	if err != nil {
		fmt.Printf("Second API call failed: %v\n", err)
	} else {
		fmt.Println("Second API call succeeded!")
	}

	// 第三次调用
	fmt.Println("\nMaking third API call...")
	var exchangeInfo interface{}
	err = api.SendHTTPRequest(ctx, "/api/v3/exchangeInfo", &exchangeInfo)
	if err != nil {
		fmt.Printf("Third API call failed: %v\n", err)
	} else {
		fmt.Println("Third API call succeeded!")
	}

	fmt.Println("\n=== Demo completed ===")
	fmt.Println("Note: In case of actual network failures, you would see:")
	fmt.Println("- 'Binance REST API request failed with IP <current_ip>: <error_details>'")
	fmt.Println("- 'retry Binance REST API request #<attempt>, because got err: <error>'")
	fmt.Println("- 'Switching to next IP: <new_ip>'")
}
