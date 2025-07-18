// Package main 演示如何使用 Binance API 实现
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mooyang-code/data-miner/internal/exchanges/binance"
	"github.com/mooyang-code/data-miner/internal/types"
)

func main() {
	// 创建 Binance 交易所实例
	b := binance.New()
	
	// 初始化交易所
	err := b.Initialize(types.BinanceConfig{})
	if err != nil {
		log.Fatalf("初始化 Binance 交易所失败: %v", err)
	}

	fmt.Printf("成功初始化 %s 交易所\n", b.GetName())
	fmt.Printf("交易所状态: 启用=%v, 连接=%v\n", b.IsEnabled(), b.IsConnected())

	ctx := context.Background()

	// 示例1: 获取单个交易对行情
	fmt.Println("\n=== 获取 BTCUSDT 行情数据 ===")
	ticker, err := b.GetTicker(ctx, "BTCUSDT")
	if err != nil {
		fmt.Printf("获取行情失败: %v\n", err)
	} else {
		fmt.Printf("交易对: %s\n", ticker.Symbol)
		fmt.Printf("当前价格: %.2f\n", ticker.Price)
		fmt.Printf("24h成交量: %.2f\n", ticker.Volume)
		fmt.Printf("24h最高价: %.2f\n", ticker.High24h)
		fmt.Printf("24h最低价: %.2f\n", ticker.Low24h)
		fmt.Printf("24h涨跌幅: %.2f%%\n", ticker.Change24h)
		fmt.Printf("时间戳: %s\n", ticker.Timestamp.Format(time.RFC3339))
	}

	// 示例2: 获取订单簿数据
	fmt.Println("\n=== 获取 BTCUSDT 订单簿数据 ===")
	orderbook, err := b.GetOrderbook(ctx, "BTCUSDT", 5)
	if err != nil {
		fmt.Printf("获取订单簿失败: %v\n", err)
	} else {
		fmt.Printf("交易对: %s\n", orderbook.Symbol)
		fmt.Printf("买单数量: %d\n", len(orderbook.Bids))
		fmt.Printf("卖单数量: %d\n", len(orderbook.Asks))
		
		if len(orderbook.Bids) > 0 {
			fmt.Printf("最佳买价: %.2f (数量: %.6f)\n", 
				orderbook.Bids[0].Price, orderbook.Bids[0].Quantity)
		}
		if len(orderbook.Asks) > 0 {
			fmt.Printf("最佳卖价: %.2f (数量: %.6f)\n", 
				orderbook.Asks[0].Price, orderbook.Asks[0].Quantity)
		}
	}

	// 示例3: 获取最近交易数据
	fmt.Println("\n=== 获取 BTCUSDT 最近交易数据 ===")
	trades, err := b.GetTrades(ctx, "BTCUSDT", 5)
	if err != nil {
		fmt.Printf("获取交易数据失败: %v\n", err)
	} else {
		fmt.Printf("获取到 %d 条交易记录\n", len(trades))
		for i, trade := range trades {
			fmt.Printf("交易 %d: ID=%s, 价格=%.2f, 数量=%.6f, 方向=%s, 时间=%s\n",
				i+1, trade.ID, trade.Price, trade.Quantity, trade.Side,
				trade.Timestamp.Format("15:04:05"))
		}
	}

	// 示例4: 获取K线数据
	fmt.Println("\n=== 获取 BTCUSDT K线数据 (1分钟) ===")
	klines, err := b.GetKlines(ctx, "BTCUSDT", "1m", 5)
	if err != nil {
		fmt.Printf("获取K线数据失败: %v\n", err)
	} else {
		fmt.Printf("获取到 %d 条K线记录\n", len(klines))
		for i, kline := range klines {
			fmt.Printf("K线 %d: 开盘=%.2f, 最高=%.2f, 最低=%.2f, 收盘=%.2f, 成交量=%.2f, 时间=%s\n",
				i+1, kline.OpenPrice, kline.HighPrice, kline.LowPrice, 
				kline.ClosePrice, kline.Volume, kline.OpenTime.Format("15:04:05"))
		}
	}

	// 示例5: 批量获取多个交易对行情
	fmt.Println("\n=== 批量获取多个交易对行情 ===")
	symbols := []types.Symbol{"BTCUSDT", "ETHUSDT", "BNBUSDT"}
	tickers, err := b.GetMultipleTickers(ctx, symbols)
	if err != nil {
		fmt.Printf("批量获取行情失败: %v\n", err)
	} else {
		fmt.Printf("获取到 %d 个交易对的行情数据\n", len(tickers))
		for _, ticker := range tickers {
			fmt.Printf("%s: 价格=%.2f, 涨跌幅=%.2f%%\n", 
				ticker.Symbol, ticker.Price, ticker.Change24h)
		}
	}

	// 示例6: 批量获取多个交易对订单簿
	fmt.Println("\n=== 批量获取多个交易对订单簿 ===")
	orderbooks, err := b.GetMultipleOrderbooks(ctx, symbols, 3)
	if err != nil {
		fmt.Printf("批量获取订单簿失败: %v\n", err)
	} else {
		fmt.Printf("获取到 %d 个交易对的订单簿数据\n", len(orderbooks))
		for _, ob := range orderbooks {
			bestBid := 0.0
			bestAsk := 0.0
			if len(ob.Bids) > 0 {
				bestBid = ob.Bids[0].Price
			}
			if len(ob.Asks) > 0 {
				bestAsk = ob.Asks[0].Price
			}
			fmt.Printf("%s: 最佳买价=%.2f, 最佳卖价=%.2f\n", 
				ob.Symbol, bestBid, bestAsk)
		}
	}

	// 示例7: 显示速率限制信息
	fmt.Println("\n=== 速率限制信息 ===")
	rateLimit := b.GetRateLimit()
	fmt.Printf("每分钟请求限制: %d\n", rateLimit.RequestsPerMinute)
	fmt.Printf("最后请求时间: %s\n", rateLimit.LastRequest.Format(time.RFC3339))

	// 示例8: WebSocket 连接状态
	fmt.Println("\n=== WebSocket 连接信息 ===")
	fmt.Printf("WebSocket 连接状态: %v\n", b.IsConnected())
	fmt.Printf("最后 Ping 时间: %s\n", b.GetLastPing().Format(time.RFC3339))

	// 清理资源
	fmt.Println("\n=== 清理资源 ===")
	err = b.Close()
	if err != nil {
		fmt.Printf("关闭连接失败: %v\n", err)
	} else {
		fmt.Println("成功关闭连接")
	}

	fmt.Println("\n示例程序执行完成!")
}
