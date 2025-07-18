// Package main 演示Binance REST API动态IP管理功能
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
	fmt.Println("=== Binance REST API 动态IP管理示例 ===")
	
	// 创建Binance REST API客户端
	api := binance.NewRestAPI()
	api.Verbose = true // 启用详细日志以查看IP切换过程
	
	// 初始化
	config := types.BinanceConfig{}
	err := api.Initialize(config)
	if err != nil {
		log.Fatalf("初始化失败: %v", err)
	}
	defer api.Close()
	
	// 等待IP管理器启动
	fmt.Println("等待IP管理器启动...")
	time.Sleep(time.Second * 3)
	
	// 显示IP管理器状态
	status := api.GetIPManagerStatus()
	fmt.Printf("IP管理器状态: %+v\n", status)
	
	// 创建交易对
	btcusdt := currency.NewPair(currency.BTC, currency.USDT)
	ethusdt := currency.NewPair(currency.ETH, currency.USDT)
	
	ctx := context.Background()
	
	// 示例1: 获取多个交易对的价格
	fmt.Println("\n=== 示例1: 获取多个交易对价格 ===")
	pairs := []currency.Pair{btcusdt, ethusdt}
	
	for i, pair := range pairs {
		fmt.Printf("请求 %d: 获取 %s 价格\n", i+1, pair.String())
		
		price, err := api.GetLatestSpotPrice(ctx, pair)
		if err != nil {
			fmt.Printf("获取 %s 价格失败: %v\n", pair.String(), err)
			continue
		}
		
		fmt.Printf("%s 当前价格: $%.2f\n", price.Symbol, price.Price)
		time.Sleep(time.Second) // 短暂延迟
	}
	
	// 示例2: 获取订单簿数据
	fmt.Println("\n=== 示例2: 获取订单簿数据 ===")
	orderBookParams := binance.OrderBookDataRequestParams{
		Symbol: btcusdt,
		Limit:  10,
	}
	
	orderBook, err := api.GetOrderBook(ctx, orderBookParams)
	if err != nil {
		fmt.Printf("获取订单簿失败: %v\n", err)
	} else {
		fmt.Printf("订单簿数据 - 买单数量: %d, 卖单数量: %d\n", 
			len(orderBook.Bids), len(orderBook.Asks))
		
		if len(orderBook.Bids) > 0 {
			fmt.Printf("最高买价: $%.2f (数量: %.6f)\n", 
				orderBook.Bids[0].Price, orderBook.Bids[0].Quantity)
		}
		
		if len(orderBook.Asks) > 0 {
			fmt.Printf("最低卖价: $%.2f (数量: %.6f)\n", 
				orderBook.Asks[0].Price, orderBook.Asks[0].Quantity)
		}
	}
	
	// 示例3: 获取K线数据
	fmt.Println("\n=== 示例3: 获取K线数据 ===")
	candleParams := &binance.KlinesRequestParams{
		Symbol:   btcusdt,
		Interval: "1m",
		Limit:    5,
	}

	candles, err := api.GetSpotKline(ctx, candleParams)
	if err != nil {
		fmt.Printf("获取K线数据失败: %v\n", err)
	} else {
		fmt.Printf("获取到 %d 根K线数据\n", len(candles))
		for i, candle := range candles {
			fmt.Printf("K线 %d: 开盘: $%.2f, 最高: $%.2f, 最低: $%.2f, 收盘: $%.2f\n",
				i+1, float64(candle.Open), float64(candle.High), float64(candle.Low), float64(candle.Close))
		}
	}
	
	// 示例4: 压力测试 - 连续请求以测试IP切换
	fmt.Println("\n=== 示例4: 压力测试 (连续请求) ===")
	successCount := 0
	failCount := 0
	
	for i := 0; i < 10; i++ {
		fmt.Printf("压力测试请求 %d/10\n", i+1)
		
		price, err := api.GetLatestSpotPrice(ctx, btcusdt)
		if err != nil {
			fmt.Printf("请求失败: %v\n", err)
			failCount++
		} else {
			fmt.Printf("成功获取价格: $%.2f\n", price.Price)
			successCount++
		}
		
		time.Sleep(time.Millisecond * 500) // 短暂延迟
	}
	
	fmt.Printf("\n压力测试结果: 成功 %d 次, 失败 %d 次\n", successCount, failCount)
	
	// 最终状态
	fmt.Println("\n=== 最终IP管理器状态 ===")
	finalStatus := api.GetIPManagerStatus()
	fmt.Printf("最终状态: %+v\n", finalStatus)
	
	fmt.Println("\n=== 示例完成 ===")
}
