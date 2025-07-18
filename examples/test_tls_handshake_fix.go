// Package main 测试TLS握手超时修复
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
	fmt.Println("=== 测试TLS握手超时修复 ===")

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

	// 等待IP管理器启动和连接预热
	fmt.Println("等待IP管理器启动和连接预热...")
	time.Sleep(5 * time.Second)

	// 显示IP管理器状态
	status := api.GetIPManagerStatus()
	fmt.Printf("IP管理器状态: %+v\n", status)

	ctx := context.Background()

	// 测试1: 基本连接测试
	fmt.Println("\n=== 测试1: 基本连接测试 ===")
	testBasicConnection(ctx, api)

	// 测试2: 连续请求测试（模拟调度器场景）
	fmt.Println("\n=== 测试2: 连续请求测试 ===")
	testContinuousRequests(ctx, api, 10)

	// 测试3: 长时间运行测试
	fmt.Println("\n=== 测试3: 长时间运行测试 ===")
	testLongRunning(ctx, api, 60*time.Second)

	fmt.Println("\n=== 测试完成 ===")
}

// testBasicConnection 测试基本连接
func testBasicConnection(ctx context.Context, api *binance.BinanceRestAPI) {
	fmt.Println("测试基本API连接...")

	// 测试获取服务器时间
	start := time.Now()
	// 这里我们直接测试trades接口，因为这是出现问题的接口
	trades, err := api.GetTrades(ctx, "BTCUSDT", 10)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("❌ 基本连接测试失败: %v (耗时: %v)\n", err, duration)
		
		// 分析错误类型
		if containsString(err.Error(), "TLS handshake timeout") {
			fmt.Println("   这是TLS握手超时错误")
		} else if containsString(err.Error(), "timeout") {
			fmt.Println("   这是其他类型的超时错误")
		} else if containsString(err.Error(), "EOF") {
			fmt.Println("   这是EOF错误")
		}
	} else {
		fmt.Printf("✅ 基本连接测试成功: 获取 %d 条交易数据 (耗时: %v)\n", len(trades), duration)
	}
}

// testContinuousRequests 测试连续请求
func testContinuousRequests(ctx context.Context, api *binance.BinanceRestAPI, count int) {
	fmt.Printf("发送 %d 个连续请求，测试TLS握手稳定性...\n", count)
	
	successCount := 0
	errorCount := 0
	tlsTimeoutCount := 0
	totalDuration := time.Duration(0)
	
	for i := 0; i < count; i++ {
		fmt.Printf("请求 %d/%d: ", i+1, count)
		
		start := time.Now()
		trades, err := api.GetTrades(ctx, "BTCUSDT", 10)
		duration := time.Since(start)
		totalDuration += duration
		
		if err != nil {
			fmt.Printf("失败 - %v (耗时: %v)\n", err, duration)
			errorCount++
			
			if containsString(err.Error(), "TLS handshake timeout") {
				tlsTimeoutCount++
			}
		} else {
			fmt.Printf("成功 - 获取到 %d 条交易数据 (耗时: %v)\n", len(trades), duration)
			successCount++
		}
		
		// 短暂延迟，模拟调度器间隔
		time.Sleep(2 * time.Second)
	}
	
	avgDuration := totalDuration / time.Duration(count)
	fmt.Printf("\n连续请求结果:\n")
	fmt.Printf("  成功: %d 次\n", successCount)
	fmt.Printf("  失败: %d 次\n", errorCount)
	fmt.Printf("  TLS握手超时: %d 次\n", tlsTimeoutCount)
	fmt.Printf("  成功率: %.1f%%\n", float64(successCount)/float64(count)*100)
	fmt.Printf("  平均耗时: %v\n", avgDuration)
}

// testLongRunning 测试长时间运行
func testLongRunning(ctx context.Context, api *binance.BinanceRestAPI, duration time.Duration) {
	fmt.Printf("长时间运行测试 (%v)...\n", duration)
	
	startTime := time.Now()
	endTime := startTime.Add(duration)
	
	requestCount := 0
	successCount := 0
	errorCount := 0
	tlsTimeoutCount := 0
	
	for time.Now().Before(endTime) {
		requestCount++
		
		// 测试不同的API接口
		var err error
		switch requestCount % 3 {
		case 0:
			// 测试trades
			_, err = api.GetTrades(ctx, "BTCUSDT", 10)
		case 1:
			// 测试价格
			pair := currency.NewPair(currency.BTC, currency.USDT)
			_, err = api.GetLatestSpotPrice(ctx, pair)
		case 2:
			// 测试订单簿
			orderBookParams := binance.OrderBookDataRequestParams{
				Symbol: currency.NewPair(currency.BTC, currency.USDT),
				Limit:  10,
			}
			_, err = api.GetOrderBook(ctx, orderBookParams)
		}
		
		if err != nil {
			errorCount++
			if containsString(err.Error(), "TLS handshake timeout") {
				tlsTimeoutCount++
			}
			fmt.Printf("请求 %d 失败: %v\n", requestCount, err)
		} else {
			successCount++
			if requestCount%10 == 0 {
				fmt.Printf("请求 %d 成功\n", requestCount)
			}
		}
		
		// 等待间隔
		time.Sleep(5 * time.Second)
	}
	
	actualDuration := time.Since(startTime)
	fmt.Printf("\n长时间运行测试结果:\n")
	fmt.Printf("  运行时间: %v\n", actualDuration)
	fmt.Printf("  总请求: %d 次\n", requestCount)
	fmt.Printf("  成功: %d 次\n", successCount)
	fmt.Printf("  失败: %d 次\n", errorCount)
	fmt.Printf("  TLS握手超时: %d 次\n", tlsTimeoutCount)
	fmt.Printf("  成功率: %.1f%%\n", float64(successCount)/float64(requestCount)*100)
	fmt.Printf("  请求频率: %.2f 请求/分钟\n", float64(requestCount)/actualDuration.Minutes())
}

// containsString 检查字符串是否包含子字符串
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && 
		   func() bool {
			   for i := 0; i <= len(s)-len(substr); i++ {
				   if s[i:i+len(substr)] == substr {
					   return true
				   }
			   }
			   return false
		   }()
}
