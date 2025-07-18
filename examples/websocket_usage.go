// Package examples 展示如何使用 Binance WebSocket 订阅功能
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
	exchange := binance.New()

	// 连接 WebSocket
	fmt.Println("连接 WebSocket...")
	if err := exchange.WsConnect(); err != nil {
		log.Fatalf("WebSocket 连接失败: %v", err)
	}

	// 等待连接建立
	time.Sleep(2 * time.Second)

	// 检查连接状态
	if !exchange.IsWsConnected() {
		log.Fatal("WebSocket 未连接")
	}
	fmt.Println("WebSocket 连接成功")

	// 定义交易对
	symbols := []types.Symbol{"BTCUSDT", "ETHUSDT"}

	// 订阅行情数据
	fmt.Println("订阅行情数据...")
	err := exchange.SubscribeTicker(symbols, func(data types.MarketData) error {
		fmt.Printf("收到行情数据: %+v\n", data)
		return nil
	})
	if err != nil {
		log.Printf("订阅行情数据失败: %v", err)
	}

	// 订阅交易数据
	fmt.Println("订阅交易数据...")
	err = exchange.SubscribeTrades(symbols, func(data types.MarketData) error {
		fmt.Printf("收到交易数据: %+v\n", data)
		return nil
	})
	if err != nil {
		log.Printf("订阅交易数据失败: %v", err)
	}

	// 订阅订单簿数据
	fmt.Println("订阅订单簿数据...")
	err = exchange.SubscribeOrderbook(symbols, func(data types.MarketData) error {
		fmt.Printf("收到订单簿数据: %+v\n", data)
		return nil
	})
	if err != nil {
		log.Printf("订阅订单簿数据失败: %v", err)
	}

	// 订阅K线数据
	fmt.Println("订阅K线数据...")
	intervals := []string{"1m", "5m"}
	err = exchange.SubscribeKlines(symbols, intervals, func(data types.MarketData) error {
		fmt.Printf("收到K线数据: %+v\n", data)
		return nil
	})
	if err != nil {
		log.Printf("订阅K线数据失败: %v", err)
	}

	// 显示当前订阅状态
	fmt.Printf("当前订阅数量: %d\n", exchange.GetSubscriptionCount())
	fmt.Printf("活跃订阅: %v\n", exchange.GetActiveSubscriptions())

	// 运行一段时间来接收数据
	fmt.Println("接收数据中，30秒后退出...")
	time.Sleep(30 * time.Second)

	// 取消所有订阅
	fmt.Println("取消所有订阅...")
	if err := exchange.UnsubscribeAll(); err != nil {
		log.Printf("取消订阅失败: %v", err)
	}

	// 关闭连接
	fmt.Println("关闭 WebSocket 连接...")
	if err := exchange.WsClose(); err != nil {
		log.Printf("关闭连接失败: %v", err)
	}

	fmt.Println("示例完成")
}

// 高级用法示例
func advancedUsageExample() {
	exchange := binance.New()

	// 连接 WebSocket
	if err := exchange.WsConnect(); err != nil {
		log.Fatalf("WebSocket 连接失败: %v", err)
	}
	defer exchange.WsClose()

	symbols := []types.Symbol{"BTCUSDT"}

	// 使用自定义深度和更新频率订阅订单簿
	fmt.Println("订阅自定义深度订单簿...")
	err := exchange.SubscribeOrderbookWithDepth(symbols, 10, "100ms", func(data types.MarketData) error {
		fmt.Printf("收到10档订单簿数据: %+v\n", data)
		return nil
	})
	if err != nil {
		log.Printf("订阅自定义订单簿失败: %v", err)
	}

	// 监控订阅状态
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for {
		select {
		case <-ticker.C:
			fmt.Printf("当前订阅数量: %d\n", exchange.GetSubscriptionCount())
			fmt.Printf("IP管理器状态: %+v\n", exchange.GetIPManagerStatus())
		case <-ctx.Done():
			fmt.Println("高级示例完成")
			return
		}
	}
}

// 错误处理示例
func errorHandlingExample() {
	exchange := binance.New()

	// 尝试在未连接时订阅（应该失败）
	symbols := []types.Symbol{"BTCUSDT"}
	err := exchange.SubscribeTicker(symbols, nil)
	if err != nil {
		fmt.Printf("预期的错误: %v\n", err)
	}

	// 连接后重试
	if err := exchange.WsConnect(); err != nil {
		log.Fatalf("WebSocket 连接失败: %v", err)
	}
	defer exchange.WsClose()

	// 现在应该成功
	err = exchange.SubscribeTicker(symbols, func(data types.MarketData) error {
		fmt.Printf("成功接收数据: %+v\n", data)
		return nil
	})
	if err != nil {
		log.Printf("意外的错误: %v", err)
	} else {
		fmt.Println("订阅成功")
	}

	time.Sleep(5 * time.Second)
}
