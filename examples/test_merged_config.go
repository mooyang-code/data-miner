// Package main 测试合并后的HTTP客户端配置
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mooyang-code/data-miner/internal/exchanges/binance"
)

func main() {
	fmt.Println("=== 测试合并后的HTTP客户端配置 ===")
	
	// 测试1: 使用默认配置创建HTTP客户端
	fmt.Println("\n1. 测试默认配置")
	testDefaultConfig()
	
	// 测试2: 使用自定义配置创建HTTP客户端
	fmt.Println("\n2. 测试自定义配置")
	testCustomConfig()
	
	fmt.Println("\n=== 测试完成 ===")
}

// testDefaultConfig 测试默认配置
func testDefaultConfig() {
	// 使用合并后的配置函数
	client, err := binance.NewHTTPClient()
	if err != nil {
		fmt.Printf("创建默认配置客户端失败: %v\n", err)
		return
	}
	defer client.Close()
	
	// 等待IP管理器启动
	time.Sleep(2 * time.Second)
	
	// 测试请求
	ctx := context.Background()
	var timeResp struct {
		ServerTime int64 `json:"serverTime"`
	}
	
	err = client.Get(ctx, "https://api.binance.com/api/v3/time", &timeResp)
	if err != nil {
		fmt.Printf("默认配置请求失败: %v\n", err)
		return
	}
	
	serverTime := time.Unix(timeResp.ServerTime/1000, 0)
	fmt.Printf("请求成功，服务器时间: %s\n", serverTime.Format("2006-01-02 15:04:05"))
	
	// 显示客户端状态
	status := client.GetStatus()
	fmt.Printf("客户端名称: %s\n", status.Name)
	fmt.Printf("总请求数: %d\n", status.TotalRequests)
	fmt.Printf("成功请求数: %d\n", status.SuccessRequests)
	
	if status.IPManager != nil {
		ipStatus := status.IPManager
		if running, ok := ipStatus["running"].(bool); ok && running {
			fmt.Printf("IP管理器: 运行中\n")
			if currentIP, ok := ipStatus["current_ip"].(string); ok {
				fmt.Printf("当前IP: %s\n", currentIP)
			}
		}
	}
	
	fmt.Println("✓ 默认配置测试成功")
}

// testCustomConfig 测试自定义配置
func testCustomConfig() {
	// 使用自定义配置（禁用动态IP，启用调试）
	client, err := binance.NewHTTPClientWithCustomConfig(false, true)
	if err != nil {
		fmt.Printf("创建自定义配置客户端失败: %v\n", err)
		return
	}
	defer client.Close()
	
	// 测试请求
	ctx := context.Background()
	var priceResp struct {
		Symbol string `json:"symbol"`
		Price  string `json:"price"`
	}
	
	err = client.Get(ctx, "https://api.binance.com/api/v3/ticker/price?symbol=BTCUSDT", &priceResp)
	if err != nil {
		fmt.Printf("自定义配置请求失败: %v\n", err)
		return
	}
	
	fmt.Printf("交易对: %s\n", priceResp.Symbol)
	fmt.Printf("价格: %s USDT\n", priceResp.Price)
	
	// 显示客户端状态
	status := client.GetStatus()
	fmt.Printf("客户端名称: %s\n", status.Name)
	fmt.Printf("总请求数: %d\n", status.TotalRequests)
	fmt.Printf("成功请求数: %d\n", status.SuccessRequests)
	
	// 检查IP管理器状态（应该是未运行，因为我们禁用了动态IP）
	if status.IPManager != nil {
		ipStatus := status.IPManager
		if running, ok := ipStatus["running"].(bool); ok {
			if running {
				fmt.Printf("IP管理器: 运行中\n")
			} else {
				fmt.Printf("IP管理器: 未运行（符合预期，因为禁用了动态IP）\n")
			}
		}
	} else {
		fmt.Printf("IP管理器: 未初始化（符合预期，因为禁用了动态IP）\n")
	}
	
	fmt.Println("✓ 自定义配置测试成功")
}
