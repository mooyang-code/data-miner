// Package main 简单测试新的Binance REST API实现
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mooyang-code/data-miner/internal/exchanges/httpclient"
)

func main() {
	fmt.Println("=== 简单测试HTTP客户端模块 ===")
	
	// 创建通用HTTP客户端（配置为Binance使用）
	config := httpclient.DefaultConfig("binance-demo")
	config.DynamicIP.Enabled = true
	config.DynamicIP.Hostname = "api.binance.com"
	config.DynamicIP.IPManager.Hostname = "api.binance.com"
	config.RateLimit.RequestsPerMinute = 1200

	client, err := httpclient.New(config)
	if err != nil {
		log.Fatalf("创建HTTP客户端失败: %v", err)
	}
	defer client.Close()
	
	// 等待IP管理器启动
	fmt.Println("等待IP管理器启动...")
	time.Sleep(3 * time.Second)
	
	ctx := context.Background()
	
	// 测试1: 获取服务器时间
	fmt.Println("\n1. 测试获取服务器时间")
	testServerTime(client, ctx)
	
	// 测试2: 获取交易对信息
	fmt.Println("\n2. 测试获取交易对信息")
	testExchangeInfo(client, ctx)
	
	// 测试3: 获取BTC/USDT价格
	fmt.Println("\n3. 测试获取BTC/USDT价格")
	testSymbolPrice(client, ctx)
	
	// 测试4: 获取订单簿
	fmt.Println("\n4. 测试获取订单簿")
	testOrderbook(client, ctx)
	
	// 测试5: 查看客户端状态
	fmt.Println("\n5. 查看客户端状态")
	showClientStatus(client)
	
	fmt.Println("\n=== 测试完成 ===")
}

// testServerTime 测试获取服务器时间
func testServerTime(client httpclient.Client, ctx context.Context) {
	var timeResp struct {
		ServerTime int64 `json:"serverTime"`
	}
	
	err := client.Get(ctx, "https://api.binance.com/api/v3/time", &timeResp)
	if err != nil {
		fmt.Printf("获取服务器时间失败: %v\n", err)
		return
	}
	
	serverTime := time.Unix(timeResp.ServerTime/1000, 0)
	fmt.Printf("Binance服务器时间: %s\n", serverTime.Format("2006-01-02 15:04:05"))
	fmt.Println("✓ 获取服务器时间成功")
}

// testExchangeInfo 测试获取交易对信息
func testExchangeInfo(client httpclient.Client, ctx context.Context) {
	var exchangeInfo struct {
		Timezone   string `json:"timezone"`
		ServerTime int64  `json:"serverTime"`
		Symbols    []struct {
			Symbol string `json:"symbol"`
			Status string `json:"status"`
		} `json:"symbols"`
	}
	
	err := client.Get(ctx, "https://api.binance.com/api/v3/exchangeInfo", &exchangeInfo)
	if err != nil {
		fmt.Printf("获取交易对信息失败: %v\n", err)
		return
	}
	
	fmt.Printf("交易所时区: %s\n", exchangeInfo.Timezone)
	fmt.Printf("活跃交易对数量: %d\n", len(exchangeInfo.Symbols))
	fmt.Println("✓ 获取交易对信息成功")
}

// testSymbolPrice 测试获取交易对价格
func testSymbolPrice(client httpclient.Client, ctx context.Context) {
	var priceResp struct {
		Symbol string `json:"symbol"`
		Price  string `json:"price"`
	}
	
	err := client.Get(ctx, "https://api.binance.com/api/v3/ticker/price?symbol=BTCUSDT", &priceResp)
	if err != nil {
		fmt.Printf("获取BTC/USDT价格失败: %v\n", err)
		return
	}
	
	fmt.Printf("交易对: %s\n", priceResp.Symbol)
	fmt.Printf("价格: %s USDT\n", priceResp.Price)
	fmt.Println("✓ 获取BTC/USDT价格成功")
}

// testOrderbook 测试获取订单簿
func testOrderbook(client httpclient.Client, ctx context.Context) {
	var orderbookResp struct {
		LastUpdateID int64 `json:"lastUpdateId"`
		Bids         [][]string `json:"bids"`
		Asks         [][]string `json:"asks"`
	}
	
	err := client.Get(ctx, "https://api.binance.com/api/v3/depth?symbol=BTCUSDT&limit=5", &orderbookResp)
	if err != nil {
		fmt.Printf("获取订单簿失败: %v\n", err)
		return
	}
	
	fmt.Printf("最后更新ID: %d\n", orderbookResp.LastUpdateID)
	fmt.Printf("买单数量: %d\n", len(orderbookResp.Bids))
	fmt.Printf("卖单数量: %d\n", len(orderbookResp.Asks))
	
	if len(orderbookResp.Bids) > 0 {
		fmt.Printf("最高买价: %s (数量: %s)\n", 
			orderbookResp.Bids[0][0], orderbookResp.Bids[0][1])
	}
	
	if len(orderbookResp.Asks) > 0 {
		fmt.Printf("最低卖价: %s (数量: %s)\n", 
			orderbookResp.Asks[0][0], orderbookResp.Asks[0][1])
	}
	
	fmt.Println("✓ 获取订单簿成功")
}

// showClientStatus 显示客户端状态
func showClientStatus(client httpclient.Client) {
	status := client.GetStatus()
	
	fmt.Printf("客户端名称: %s\n", status.Name)
	fmt.Printf("运行状态: %v\n", status.Running)
	fmt.Printf("总请求数: %d\n", status.TotalRequests)
	fmt.Printf("成功请求数: %d\n", status.SuccessRequests)
	fmt.Printf("失败请求数: %d\n", status.FailedRequests)
	fmt.Printf("重试次数: %d\n", status.RetryCount)
	
	if status.RateLimit != nil {
		fmt.Printf("速率限制: %d/分钟，剩余: %d\n", 
			status.RateLimit.RequestsPerMinute, 
			status.RateLimit.Remaining)
	}
	
	if status.IPManager != nil {
		ipStatus := status.IPManager
		if running, ok := ipStatus["running"].(bool); ok && running {
			fmt.Printf("IP管理器: 运行中\n")
			if currentIP, ok := ipStatus["current_ip"].(string); ok {
				fmt.Printf("当前IP: %s\n", currentIP)
			}
			if ipCount, ok := ipStatus["ip_count"].(int); ok {
				fmt.Printf("可用IP数量: %d\n", ipCount)
			}
		} else {
			fmt.Printf("IP管理器: 未运行\n")
		}
	}
	
	if status.LastError != "" {
		fmt.Printf("最后错误: %s\n", status.LastError)
	}
	
	fmt.Println("✓ 客户端状态正常")
}
