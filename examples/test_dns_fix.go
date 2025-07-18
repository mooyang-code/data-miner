// Package main 测试DNS解析修复
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mooyang-code/data-miner/internal/exchanges/binance"
	"github.com/mooyang-code/data-miner/internal/ipmanager"
	"github.com/mooyang-code/data-miner/internal/types"
	"github.com/mooyang-code/data-miner/pkg/cryptotrader/currency"
)

func main() {
	fmt.Println("=== 测试DNS解析修复 ===")

	// 首先测试IP管理器的DNS解析
	fmt.Println("\n=== 测试IP管理器DNS解析 ===")
	config := ipmanager.DefaultConfig("api.binance.com")
	manager := ipmanager.New(config)

	ctx := context.Background()
	err := manager.Start(ctx)
	if err != nil {
		log.Fatalf("启动IP管理器失败: %v", err)
	}
	defer manager.Stop()

	// 等待DNS解析完成
	fmt.Println("等待DNS解析完成...")
	time.Sleep(5 * time.Second)

	// 显示解析结果
	status := manager.GetStatus()
	fmt.Printf("IP管理器状态:\n")
	fmt.Printf("  域名: %s\n", status["hostname"])
	fmt.Printf("  运行状态: %v\n", status["running"])
	fmt.Printf("  当前IP: %s\n", status["current_ip"])
	fmt.Printf("  所有IP: %v\n", status["all_ips"])
	fmt.Printf("  IP数量: %v\n", status["ip_count"])

	// 验证IP地址
	allIPs := manager.GetAllIPs()
	fmt.Printf("\n=== IP地址验证 ===\n")
	for i, ip := range allIPs {
		fmt.Printf("%d. IP: %s\n", i+1, ip)
		
		// 检查是否是已知的问题IP
		if ip == "199.59.148.246" {
			fmt.Printf("   ⚠️  警告: 这是Twitter的IP地址，不应该用于Binance!\n")
		} else if isCloudFrontIP(ip) {
			fmt.Printf("   ✅ 这是CloudFront IP，可能是正确的\n")
		} else {
			fmt.Printf("   ❓ 未知IP，需要验证\n")
		}
	}

	// 测试实际的API请求
	fmt.Println("\n=== 测试API请求 ===")
	api := binance.NewRestAPI()
	api.Verbose = true

	// 初始化
	binanceConfig := types.BinanceConfig{}
	err = api.Initialize(binanceConfig)
	if err != nil {
		log.Fatalf("初始化Binance API失败: %v", err)
	}
	defer api.Close()

	// 等待初始化完成
	time.Sleep(3 * time.Second)

	// 测试获取价格
	fmt.Println("测试获取BTC/USDT价格...")
	pair := currency.NewPair(currency.BTC, currency.USDT)
	price, err := api.GetLatestSpotPrice(ctx, pair)
	if err != nil {
		fmt.Printf("❌ 获取价格失败: %v\n", err)
		
		// 检查是否是TLS证书错误
		if containsString(err.Error(), "certificate") {
			fmt.Println("   这是TLS证书错误，说明连接到了错误的服务器")
		}
		if containsString(err.Error(), "facebook.com") {
			fmt.Println("   错误信息包含facebook.com，说明DNS被污染了")
		}
	} else {
		fmt.Printf("✅ 成功获取价格: $%.2f\n", price.Price)
	}

	// 测试获取交易数据
	fmt.Println("\n测试获取交易数据...")
	trades, err := api.GetTrades(ctx, "BTCUSDT", 5)
	if err != nil {
		fmt.Printf("❌ 获取交易数据失败: %v\n", err)
	} else {
		fmt.Printf("✅ 成功获取 %d 条交易数据\n", len(trades))
	}

	// 显示最终的IP管理器状态
	fmt.Println("\n=== 最终IP管理器状态 ===")
	finalStatus := api.GetIPManagerStatus()
	fmt.Printf("最终状态: %+v\n", finalStatus)

	fmt.Println("\n=== 测试完成 ===")
}

// isCloudFrontIP 检查是否是CloudFront IP
func isCloudFrontIP(ip string) bool {
	// 已知的一些CloudFront IP段
	cloudFrontRanges := []string{
		"13.32.", "13.226.", "54.230.", "54.239.",
		"52.84.", "204.246.", "205.251.", "216.137.",
	}
	
	for _, prefix := range cloudFrontRanges {
		if len(ip) >= len(prefix) && ip[:len(prefix)] == prefix {
			return true
		}
	}
	return false
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
