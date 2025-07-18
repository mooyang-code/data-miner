// Package main 测试新的Binance REST API实现
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
	fmt.Println("=== 测试新的Binance REST API实现 ===")
	
	// 创建新的Binance REST API客户端
	api := binance.NewRestAPI()
	if api == nil {
		log.Fatal("创建Binance REST API客户端失败")
	}
	
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
	
	ctx := context.Background()
	
	// 测试1: 获取最新现货价格
	fmt.Println("\n1. 测试获取BTC/USDT最新价格")
	testGetLatestPrice(api, ctx)
	
	// 测试2: 获取订单簿
	fmt.Println("\n2. 测试获取BTC/USDT订单簿")
	testGetOrderbook(api, ctx)
	
	// 测试3: 获取K线数据
	fmt.Println("\n3. 测试获取BTC/USDT K线数据")
	testGetKlines(api, ctx)
	
	// 测试4: 获取24小时价格变化统计
	fmt.Println("\n4. 测试获取24小时价格变化统计")
	testGetTickers(api, ctx)
	
	// 测试5: 查看客户端状态
	fmt.Println("\n5. 查看客户端状态")
	showAPIStatus(api)
	
	fmt.Println("\n=== 测试完成 ===")
}

// testGetLatestPrice 测试获取最新价格
func testGetLatestPrice(api *binance.BinanceRestAPI, ctx context.Context) {
	pair := currency.NewPair(currency.BTC, currency.USDT)
	
	price, err := api.GetLatestSpotPrice(ctx, pair)
	if err != nil {
		fmt.Printf("获取最新价格失败: %v\n", err)
		return
	}
	
	fmt.Printf("交易对: %s\n", price.Symbol)
	fmt.Printf("价格: %s\n", price.Price)
	
	if price.Symbol == "" || price.Price == "" {
		fmt.Println("警告: 价格数据不完整")
	} else {
		fmt.Println("✓ 获取最新价格成功")
	}
}

// testGetOrderbook 测试获取订单簿
func testGetOrderbook(api *binance.BinanceRestAPI, ctx context.Context) {
	pair := currency.NewPair(currency.BTC, currency.USDT)
	
	orderbook, err := api.GetOrderbook(ctx, pair, 10)
	if err != nil {
		fmt.Printf("获取订单簿失败: %v\n", err)
		return
	}
	
	fmt.Printf("交易对: %s\n", orderbook.Symbol)
	fmt.Printf("最后更新ID: %d\n", orderbook.LastUpdateID)
	fmt.Printf("买单数量: %d\n", len(orderbook.Bids))
	fmt.Printf("卖单数量: %d\n", len(orderbook.Asks))
	
	if len(orderbook.Bids) > 0 {
		fmt.Printf("最高买价: %.2f (数量: %.6f)\n", 
			orderbook.Bids[0].Price, orderbook.Bids[0].Quantity)
	}
	
	if len(orderbook.Asks) > 0 {
		fmt.Printf("最低卖价: %.2f (数量: %.6f)\n", 
			orderbook.Asks[0].Price, orderbook.Asks[0].Quantity)
	}
	
	if len(orderbook.Bids) > 0 && len(orderbook.Asks) > 0 {
		fmt.Println("✓ 获取订单簿成功")
	} else {
		fmt.Println("警告: 订单簿数据不完整")
	}
}

// testGetKlines 测试获取K线数据
func testGetKlines(api *binance.BinanceRestAPI, ctx context.Context) {
	pair := currency.NewPair(currency.BTC, currency.USDT)
	
	klines, err := api.GetKlines(ctx, pair, "1h", 5, 0, 0)
	if err != nil {
		fmt.Printf("获取K线数据失败: %v\n", err)
		return
	}
	
	fmt.Printf("K线数据数量: %d\n", len(klines))
	
	if len(klines) > 0 {
		latest := klines[len(klines)-1]
		fmt.Printf("最新K线:\n")
		fmt.Printf("  开盘时间: %d\n", latest.OpenTime)
		fmt.Printf("  开盘价: %s\n", latest.Open)
		fmt.Printf("  最高价: %s\n", latest.High)
		fmt.Printf("  最低价: %s\n", latest.Low)
		fmt.Printf("  收盘价: %s\n", latest.Close)
		fmt.Printf("  成交量: %s\n", latest.Volume)
		fmt.Println("✓ 获取K线数据成功")
	} else {
		fmt.Println("警告: 没有获取到K线数据")
	}
}

// testGetTickers 测试获取24小时价格变化统计
func testGetTickers(api *binance.BinanceRestAPI, ctx context.Context) {
	pair := currency.NewPair(currency.BTC, currency.USDT)
	
	tickers, err := api.GetTickers(ctx, pair)
	if err != nil {
		fmt.Printf("获取价格变化统计失败: %v\n", err)
		return
	}
	
	if len(tickers) > 0 {
		ticker := tickers[0]
		fmt.Printf("交易对: %s\n", ticker.Symbol)
		fmt.Printf("24h价格变化: %s\n", ticker.PriceChange)
		fmt.Printf("24h价格变化百分比: %s\n", ticker.PriceChangePercent)
		fmt.Printf("24h最高价: %s\n", ticker.HighPrice)
		fmt.Printf("24h最低价: %s\n", ticker.LowPrice)
		fmt.Printf("24h成交量: %s\n", ticker.Volume)
		fmt.Printf("24h成交额: %s\n", ticker.QuoteVolume)
		fmt.Println("✓ 获取价格变化统计成功")
	} else {
		fmt.Println("警告: 没有获取到价格变化统计数据")
	}
}

// showAPIStatus 显示API状态
func showAPIStatus(api *binance.BinanceRestAPI) {
	status := api.GetStatus()
	
	fmt.Printf("API名称: %v\n", status["name"])
	fmt.Printf("启用状态: %v\n", status["enabled"])
	
	if httpClient, ok := status["http_client"]; ok {
		if clientStatus, ok := httpClient.(map[string]interface{}); ok {
			fmt.Printf("HTTP客户端状态:\n")
			fmt.Printf("  名称: %v\n", clientStatus["name"])
			fmt.Printf("  运行状态: %v\n", clientStatus["running"])
			fmt.Printf("  总请求数: %v\n", clientStatus["total_requests"])
			fmt.Printf("  成功请求数: %v\n", clientStatus["success_requests"])
			fmt.Printf("  失败请求数: %v\n", clientStatus["failed_requests"])
			fmt.Printf("  重试次数: %v\n", clientStatus["retry_count"])
			
			if rateLimit, ok := clientStatus["rate_limit"]; ok {
				if rl, ok := rateLimit.(map[string]interface{}); ok {
					fmt.Printf("  速率限制: %v/分钟\n", rl["requests_per_minute"])
					fmt.Printf("  剩余配额: %v\n", rl["remaining"])
				}
			}
			
			if ipManager, ok := clientStatus["ip_manager"]; ok {
				if ipm, ok := ipManager.(map[string]interface{}); ok {
					fmt.Printf("  IP管理器运行状态: %v\n", ipm["running"])
					if currentIP, ok := ipm["current_ip"]; ok {
						fmt.Printf("  当前IP: %v\n", currentIP)
					}
					if ipCount, ok := ipm["ip_count"]; ok {
						fmt.Printf("  可用IP数量: %v\n", ipCount)
					}
				}
			}
		}
	}
}
