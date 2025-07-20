// Package main 测试配置加载功能
package main

import (
	"fmt"
	"log"

	"github.com/mooyang-code/data-miner/pkg/utils"
)

func main() {
	// 测试加载配置文件
	fmt.Println("=== 测试配置文件加载 ===")

	// 加载配置
	config, err := utils.LoadConfig("../config/config.yaml")
	if err != nil {
		log.Fatal("加载配置失败:", err)
	}

	// 打印关键配置信息
	fmt.Printf("应用名称: %s\n", config.App.Name)
	fmt.Printf("应用版本: %s\n", config.App.Version)
	fmt.Printf("日志级别: %s\n", config.App.LogLevel)

	// 打印Binance配置
	binanceConfig := config.Exchanges.Binance
	fmt.Printf("\n=== Binance配置 ===\n")
	fmt.Printf("启用状态: %t\n", binanceConfig.Enabled)
	fmt.Printf("API URL: %s\n", binanceConfig.APIURL)
	fmt.Printf("WebSocket URL: %s\n", binanceConfig.WebsocketURL)
	fmt.Printf("使用WebSocket: %t\n", binanceConfig.UseWebsocket)

	// 打印交易对缓存配置
	tradablePairs := binanceConfig.TradablePairs
	fmt.Printf("\n=== 交易对缓存配置 ===\n")
	fmt.Printf("从API获取: %t\n", tradablePairs.FetchFromAPI)
	fmt.Printf("更新间隔: %v\n", tradablePairs.UpdateInterval)
	fmt.Printf("缓存启用: %t\n", tradablePairs.CacheEnabled)
	fmt.Printf("缓存TTL: %v\n", tradablePairs.CacheTTL)
	fmt.Printf("支持的资产类型: %v\n", tradablePairs.SupportedAssets)
	fmt.Printf("自动更新: %t\n", tradablePairs.AutoUpdate)

	// 打印数据类型配置
	dataTypes := binanceConfig.DataTypes
	fmt.Printf("\n=== 数据类型配置 ===\n")

	// Ticker配置
	fmt.Printf("Ticker - 启用: %t, 交易对: %v, 间隔: %s\n",
		dataTypes.Ticker.Enabled,
		dataTypes.Ticker.Symbols,
		dataTypes.Ticker.Interval)

	// Orderbook配置
	fmt.Printf("Orderbook - 启用: %t, 交易对: %v, 深度: %d, 间隔: %s\n",
		dataTypes.Orderbook.Enabled,
		dataTypes.Orderbook.Symbols,
		dataTypes.Orderbook.Depth,
		dataTypes.Orderbook.Interval)

	// Trades配置
	fmt.Printf("Trades - 启用: %t, 交易对: %v, 间隔: %s\n",
		dataTypes.Trades.Enabled,
		dataTypes.Trades.Symbols,
		dataTypes.Trades.Interval)

	// Klines配置
	fmt.Printf("Klines - 启用: %t, 交易对: %v, 时间间隔: %v, 间隔: %s\n",
		dataTypes.Klines.Enabled,
		dataTypes.Klines.Symbols,
		dataTypes.Klines.Intervals,
		dataTypes.Klines.Interval)

	// 打印调度器配置
	scheduler := config.Scheduler
	fmt.Printf("\n=== 调度器配置 ===\n")
	fmt.Printf("启用状态: %t\n", scheduler.Enabled)
	fmt.Printf("最大并发任务数: %d\n", scheduler.MaxConcurrentJobs)
	fmt.Printf("任务数量: %d\n", len(scheduler.Jobs))

	for i, job := range scheduler.Jobs {
		fmt.Printf("任务%d - 名称: %s, 交易所: %s, 数据类型: %s, Cron: %s\n",
			i+1, job.Name, job.Exchange, job.DataType, job.Cron)
	}

	// 验证关键配置
	fmt.Printf("\n=== 配置验证 ===\n")
	if tradablePairs.FetchFromAPI {
		fmt.Println("✅ fetch_from_api 已启用 - 将从缓存获取交易对列表")
	} else {
		fmt.Println("❌ fetch_from_api 未启用 - 将使用配置文件中的固定交易对列表")
	}

	// 检查是否有使用通配符的配置
	hasWildcard := false
	if len(dataTypes.Ticker.Symbols) == 1 && dataTypes.Ticker.Symbols[0] == "*" {
		fmt.Println("✅ Ticker 配置使用通配符 '*' - 将从缓存获取所有交易对")
		hasWildcard = true
	}
	if len(dataTypes.Klines.Symbols) == 1 && dataTypes.Klines.Symbols[0] == "*" {
		fmt.Println("✅ Klines 配置使用通配符 '*' - 将从缓存获取所有交易对")
		hasWildcard = true
	}

	if hasWildcard && !tradablePairs.FetchFromAPI {
		fmt.Println("⚠️  警告: 配置中使用了通配符但 fetch_from_api 未启用，可能无法获取交易对")
	}

	fmt.Println("\n=== 配置加载测试完成 ===")
}
