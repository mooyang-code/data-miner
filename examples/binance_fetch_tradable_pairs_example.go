package main

import (
	"context"
	"fmt"
	"log"

	"github.com/mooyang-code/data-miner/internal/exchanges/binance"
	"github.com/mooyang-code/data-miner/pkg/cryptotrader/exchanges/asset"
)

func main() {
	fmt.Println("=== Binance FetchTradablePairs 示例 ===")

	// 创建Binance交易所实例
	b := binance.New()

	// 创建上下文
	ctx := context.Background()

	// 1. 获取现货交易对
	fmt.Println("\n1. 获取现货交易对...")
	spotPairs, err := b.FetchTradablePairs(ctx, asset.Spot)
	if err != nil {
		log.Printf("获取现货交易对失败: %v", err)
		return
	}

	fmt.Printf("找到 %d 个现货交易对\n", len(spotPairs))
	fmt.Println("前10个现货交易对:")
	for i, pair := range spotPairs {
		if i >= 10 {
			break
		}
		fmt.Printf("  %d. %s (Base: %s, Quote: %s)\n", 
			i+1, pair.String(), pair.Base.String(), pair.Quote.String())
	}

	// 2. 获取保证金交易对
	fmt.Println("\n2. 获取保证金交易对...")
	marginPairs, err := b.FetchTradablePairs(ctx, asset.Margin)
	if err != nil {
		log.Printf("获取保证金交易对失败: %v", err)
		return
	}

	fmt.Printf("找到 %d 个保证金交易对\n", len(marginPairs))
	fmt.Println("前5个保证金交易对:")
	for i, pair := range marginPairs {
		if i >= 5 {
			break
		}
		fmt.Printf("  %d. %s\n", i+1, pair.String())
	}

	// 3. 统计不同计价货币的交易对数量
	fmt.Println("\n3. 统计不同计价货币的交易对数量...")
	quoteStats := make(map[string]int)
	for _, pair := range spotPairs {
		quoteStats[pair.Quote.String()]++
	}

	fmt.Println("主要计价货币统计:")
	for quote, count := range quoteStats {
		if count >= 50 { // 只显示交易对数量>=50的计价货币
			fmt.Printf("  %s: %d 个交易对\n", quote, count)
		}
	}

	// 4. 查找特定的交易对
	fmt.Println("\n4. 查找特定交易对...")
	targetSymbols := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "DOTUSDT"}
	
	for _, target := range targetSymbols {
		found := false
		for _, pair := range spotPairs {
			if pair.String() == target {
				fmt.Printf("  ✓ 找到交易对: %s\n", target)
				found = true
				break
			}
		}
		if !found {
			fmt.Printf("  ✗ 未找到交易对: %s\n", target)
		}
	}

	// 5. 测试不支持的资产类型
	fmt.Println("\n5. 测试不支持的资产类型...")
	_, err = b.FetchTradablePairs(ctx, asset.Futures)
	if err != nil {
		fmt.Printf("  预期错误: %v\n", err)
	}

	// 6. 获取交易所基本信息
	fmt.Println("\n6. 获取交易所基本信息...")
	exchangeInfo, err := b.RestAPI.GetExchangeInfo(ctx)
	if err != nil {
		log.Printf("获取交易所信息失败: %v", err)
		return
	}

	fmt.Printf("  交易所时区: %s\n", exchangeInfo.Timezone)
	fmt.Printf("  服务器时间: %s\n", exchangeInfo.ServerTime.Time().Format("2006-01-02 15:04:05"))
	fmt.Printf("  总交易对数量: %d\n", len(exchangeInfo.Symbols))

	// 统计不同状态的交易对
	statusStats := make(map[string]int)
	for _, symbol := range exchangeInfo.Symbols {
		statusStats[symbol.Status]++
	}

	fmt.Println("  交易对状态统计:")
	for status, count := range statusStats {
		fmt.Printf("    %s: %d\n", status, count)
	}

	fmt.Println("\n=== 示例完成 ===")
}
