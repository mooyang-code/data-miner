package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mooyang-code/data-miner/internal/exchanges/binance"
)

func main() {
	fmt.Println("Testing Binance REST API with retry-go library...")

	// 创建Binance REST API客户端
	api := binance.NewRestAPI()

	// 初始化
	if err := api.Initialize(nil); err != nil {
		log.Fatalf("Failed to initialize Binance API: %v", err)
	}
	defer api.Close()

	// 设置详细日志
	api.Verbose = true

	// 创建上下文，设置超时
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 测试获取交易所信息（这个API通常比较稳定）
	fmt.Println("\n=== Testing Exchange Info API ===")
	var exchangeInfo interface{}
	err := api.SendHTTPRequest(ctx, "/api/v3/exchangeInfo", &exchangeInfo)
	if err != nil {
		fmt.Printf("Exchange Info API failed: %v\n", err)
	} else {
		fmt.Println("Exchange Info API succeeded!")
	}

	// 测试获取服务器时间（轻量级API）
	fmt.Println("\n=== Testing Server Time API ===")
	var serverTime interface{}
	err = api.SendHTTPRequest(ctx, "/api/v3/time", &serverTime)
	if err != nil {
		fmt.Printf("Server Time API failed: %v\n", err)
	} else {
		fmt.Println("Server Time API succeeded!")
		fmt.Printf("Server Time Response: %+v\n", serverTime)
	}

	// 测试获取BTCUSDT价格
	fmt.Println("\n=== Testing Symbol Price API ===")
	var priceInfo interface{}
	err = api.SendHTTPRequest(ctx, "/api/v3/ticker/price?symbol=BTCUSDT", &priceInfo)
	if err != nil {
		fmt.Printf("Symbol Price API failed: %v\n", err)
	} else {
		fmt.Println("Symbol Price API succeeded!")
		fmt.Printf("Price Info: %+v\n", priceInfo)
	}

	fmt.Println("\n=== Test completed ===")
}
