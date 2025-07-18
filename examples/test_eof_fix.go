// Package main 测试EOF错误修复
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
	fmt.Println("=== 测试EOF错误修复 ===")

	// 创建Binance REST API客户端
	api := binance.NewRestAPI()
	api.Verbose = true // 启用详细日志

	// 初始化
	config := types.BinanceConfig{}
	err := api.Initialize(config)
	if err != nil {
		log.Fatalf("初始化失败: %v", err)
	}
	defer api.Close()

	// 等待IP管理器启动
	fmt.Println("等待IP管理器启动...")
	time.Sleep(3 * time.Second)

	// 显示IP管理器状态
	status := api.GetIPManagerStatus()
	fmt.Printf("IP管理器状态: %+v\n", status)

	ctx := context.Background()

	// 测试多次请求，检查是否还有EOF错误
	fmt.Println("\n=== 连续请求测试 ===")
	successCount := 0
	errorCount := 0

	for i := 0; i < 10; i++ {
		fmt.Printf("请求 %d/10: ", i+1)

		// 测试获取交易数据
		trades, err := api.GetTrades(ctx, "BTCUSDT", 10)
		if err != nil {
			fmt.Printf("失败 - %v\n", err)
			errorCount++
		} else {
			fmt.Printf("成功 - 获取到 %d 条交易数据\n", len(trades))
			successCount++
		}

		// 短暂延迟
		time.Sleep(time.Second)
	}

	fmt.Printf("\n=== 测试结果 ===\n")
	fmt.Printf("成功: %d 次\n", successCount)
	fmt.Printf("失败: %d 次\n", errorCount)
	fmt.Printf("成功率: %.1f%%\n", float64(successCount)/10*100)

	if errorCount == 0 {
		fmt.Println("✅ 所有请求都成功，EOF错误已修复！")
	} else {
		fmt.Printf("⚠️  仍有 %d 次失败，需要进一步调试\n", errorCount)
	}

	// 测试其他API
	fmt.Println("\n=== 其他API测试 ===")

	// 测试获取价格
	pair := currency.NewPair(currency.BTC, currency.USDT)
	price, err := api.GetLatestSpotPrice(ctx, pair)
	if err != nil {
		fmt.Printf("获取价格失败: %v\n", err)
	} else {
		fmt.Printf("BTC/USDT 价格: $%.2f\n", price.Price)
	}

	// 测试获取订单簿
	orderBookParams := binance.OrderBookDataRequestParams{
		Symbol: pair,
		Limit:  10,
	}
	orderBook, err := api.GetOrderBook(ctx, orderBookParams)
	if err != nil {
		fmt.Printf("获取订单簿失败: %v\n", err)
	} else {
		fmt.Printf("订单簿 - 买单: %d, 卖单: %d\n", len(orderBook.Bids), len(orderBook.Asks))
	}

	fmt.Println("\n=== 测试完成 ===")
}
