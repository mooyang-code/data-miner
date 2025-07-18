// Package main 测试高频请求的稳定性
package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mooyang-code/data-miner/internal/exchanges/binance"
	"github.com/mooyang-code/data-miner/internal/types"
	"github.com/mooyang-code/data-miner/pkg/cryptotrader/currency"
)

func main() {
	fmt.Println("=== 测试高频请求稳定性 ===")

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

	// 测试1: 连续高频请求
	fmt.Println("\n=== 测试1: 连续高频请求 ===")
	testContinuousRequests(ctx, api, 20, time.Second)

	// 测试2: 并发请求
	fmt.Println("\n=== 测试2: 并发请求 ===")
	testConcurrentRequests(ctx, api, 10)

	// 测试3: 模拟调度器场景
	fmt.Println("\n=== 测试3: 模拟调度器场景 ===")
	testSchedulerScenario(ctx, api)

	fmt.Println("\n=== 测试完成 ===")
}

// testContinuousRequests 测试连续高频请求
func testContinuousRequests(ctx context.Context, api *binance.BinanceRestAPI, count int, interval time.Duration) {
	successCount := 0
	errorCount := 0
	
	fmt.Printf("发送 %d 个连续请求，间隔 %v\n", count, interval)
	
	for i := 0; i < count; i++ {
		fmt.Printf("请求 %d/%d: ", i+1, count)
		
		// 测试获取交易数据
		trades, err := api.GetTrades(ctx, "BTCUSDT", 10)
		if err != nil {
			fmt.Printf("失败 - %v\n", err)
			errorCount++
		} else {
			fmt.Printf("成功 - 获取到 %d 条交易数据\n", len(trades))
			successCount++
		}
		
		// 等待间隔
		if i < count-1 {
			time.Sleep(interval)
		}
	}
	
	fmt.Printf("连续请求结果: 成功 %d 次, 失败 %d 次, 成功率 %.1f%%\n", 
		successCount, errorCount, float64(successCount)/float64(count)*100)
}

// testConcurrentRequests 测试并发请求
func testConcurrentRequests(ctx context.Context, api *binance.BinanceRestAPI, concurrency int) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0
	errorCount := 0
	
	fmt.Printf("发送 %d 个并发请求\n", concurrency)
	
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			// 测试获取价格
			pair := currency.NewPair(currency.BTC, currency.USDT)
			price, err := api.GetLatestSpotPrice(ctx, pair)
			
			mu.Lock()
			if err != nil {
				fmt.Printf("并发请求 %d: 失败 - %v\n", id+1, err)
				errorCount++
			} else {
				fmt.Printf("并发请求 %d: 成功 - 价格 $%.2f\n", id+1, price.Price)
				successCount++
			}
			mu.Unlock()
		}(i)
	}
	
	wg.Wait()
	
	fmt.Printf("并发请求结果: 成功 %d 次, 失败 %d 次, 成功率 %.1f%%\n", 
		successCount, errorCount, float64(successCount)/float64(concurrency)*100)
}

// testSchedulerScenario 模拟调度器场景
func testSchedulerScenario(ctx context.Context, api *binance.BinanceRestAPI) {
	fmt.Println("模拟调度器高频场景（30秒）...")
	
	var wg sync.WaitGroup
	var mu sync.Mutex
	totalRequests := 0
	totalErrors := 0
	
	// 模拟trades任务（每10秒）
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		
		timeout := time.After(30 * time.Second)
		
		for {
			select {
			case <-timeout:
				return
			case <-ticker.C:
				trades, err := api.GetTrades(ctx, "BTCUSDT", 100)
				mu.Lock()
				totalRequests++
				if err != nil {
					fmt.Printf("Trades任务失败: %v\n", err)
					totalErrors++
				} else {
					fmt.Printf("Trades任务成功: 获取 %d 条数据\n", len(trades))
				}
				mu.Unlock()
			}
		}
	}()
	
	// 模拟orderbook任务（每5秒）
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		
		timeout := time.After(30 * time.Second)
		
		for {
			select {
			case <-timeout:
				return
			case <-ticker.C:
				orderBookParams := binance.OrderBookDataRequestParams{
					Symbol: currency.NewPair(currency.BTC, currency.USDT),
					Limit:  10,
				}
				orderBook, err := api.GetOrderBook(ctx, orderBookParams)
				mu.Lock()
				totalRequests++
				if err != nil {
					fmt.Printf("OrderBook任务失败: %v\n", err)
					totalErrors++
				} else {
					fmt.Printf("OrderBook任务成功: 买单 %d, 卖单 %d\n", 
						len(orderBook.Bids), len(orderBook.Asks))
				}
				mu.Unlock()
			}
		}
	}()
	
	// 模拟ticker任务（每分钟，但我们只运行一次）
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(5 * time.Second) // 延迟5秒开始
		
		pair := currency.NewPair(currency.BTC, currency.USDT)
		price, err := api.GetLatestSpotPrice(ctx, pair)
		mu.Lock()
		totalRequests++
		if err != nil {
			fmt.Printf("Ticker任务失败: %v\n", err)
			totalErrors++
		} else {
			fmt.Printf("Ticker任务成功: 价格 $%.2f\n", price.Price)
		}
		mu.Unlock()
	}()
	
	wg.Wait()
	
	fmt.Printf("调度器模拟结果: 总请求 %d 次, 失败 %d 次, 成功率 %.1f%%\n", 
		totalRequests, totalErrors, float64(totalRequests-totalErrors)/float64(totalRequests)*100)
}
