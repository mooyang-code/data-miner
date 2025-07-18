// Package main 测试Binance专用HTTP客户端配置
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mooyang-code/data-miner/internal/exchanges/binance"
)

func main() {
	fmt.Println("=== 测试Binance专用HTTP客户端配置 ===")
	
	// 使用Binance模块的专用配置创建HTTP客户端
	client, err := binance.NewHTTPClient()
	if err != nil {
		log.Fatalf("创建Binance HTTP客户端失败: %v", err)
	}
	defer client.Close()
	
	// 等待IP管理器启动
	fmt.Println("等待IP管理器启动...")
	time.Sleep(3 * time.Second)
	
	ctx := context.Background()
	
	// 测试1: 获取服务器时间
	fmt.Println("\n1. 测试获取服务器时间")
	testServerTime(client, ctx)
	
	// 测试2: 获取BTC/USDT价格
	fmt.Println("\n2. 测试获取BTC/USDT价格")
	testSymbolPrice(client, ctx)
	
	// 测试3: 查看客户端状态
	fmt.Println("\n3. 查看客户端状态")
	showClientStatus(client)
	
	// 测试4: 使用自定义配置
	fmt.Println("\n4. 测试自定义配置（禁用动态IP）")
	testCustomConfig()
	
	fmt.Println("\n=== 测试完成 ===")
}

// testServerTime 测试获取服务器时间
func testServerTime(client interface{}, ctx context.Context) {
	// 这里需要类型断言，因为我们返回的是interface{}
	httpClient := client.(interface {
		Get(ctx context.Context, url string, result interface{}) error
	})
	
	var timeResp struct {
		ServerTime int64 `json:"serverTime"`
	}
	
	err := httpClient.Get(ctx, "https://api.binance.com/api/v3/time", &timeResp)
	if err != nil {
		fmt.Printf("获取服务器时间失败: %v\n", err)
		return
	}
	
	serverTime := time.Unix(timeResp.ServerTime/1000, 0)
	fmt.Printf("Binance服务器时间: %s\n", serverTime.Format("2006-01-02 15:04:05"))
	fmt.Println("✓ 获取服务器时间成功")
}

// testSymbolPrice 测试获取交易对价格
func testSymbolPrice(client interface{}, ctx context.Context) {
	httpClient := client.(interface {
		Get(ctx context.Context, url string, result interface{}) error
	})
	
	var priceResp struct {
		Symbol string `json:"symbol"`
		Price  string `json:"price"`
	}
	
	err := httpClient.Get(ctx, "https://api.binance.com/api/v3/ticker/price?symbol=BTCUSDT", &priceResp)
	if err != nil {
		fmt.Printf("获取BTC/USDT价格失败: %v\n", err)
		return
	}
	
	fmt.Printf("交易对: %s\n", priceResp.Symbol)
	fmt.Printf("价格: %s USDT\n", priceResp.Price)
	fmt.Println("✓ 获取BTC/USDT价格成功")
}

// showClientStatus 显示客户端状态
func showClientStatus(client interface{}) {
	statusClient := client.(interface {
		GetStatus() interface{}
	})
	
	statusInterface := statusClient.GetStatus()
	
	// 这里需要进行类型断言来访问状态信息
	if status, ok := statusInterface.(interface {
		Name            string
		Running         bool
		TotalRequests   int64
		SuccessRequests int64
		FailedRequests  int64
		RetryCount      int64
	}); ok {
		fmt.Printf("客户端名称: %s\n", status.Name)
		fmt.Printf("运行状态: %v\n", status.Running)
		fmt.Printf("总请求数: %d\n", status.TotalRequests)
		fmt.Printf("成功请求数: %d\n", status.SuccessRequests)
		fmt.Printf("失败请求数: %d\n", status.FailedRequests)
		fmt.Printf("重试次数: %d\n", status.RetryCount)
		fmt.Println("✓ 客户端状态正常")
	} else {
		fmt.Printf("客户端状态: %+v\n", statusInterface)
	}
}

// testCustomConfig 测试自定义配置
func testCustomConfig() {
	// 创建禁用动态IP的客户端
	client, err := binance.NewHTTPClientWithCustomConfig(false, true)
	if err != nil {
		fmt.Printf("创建自定义配置客户端失败: %v\n", err)
		return
	}
	defer client.Close()
	
	ctx := context.Background()
	var timeResp struct {
		ServerTime int64 `json:"serverTime"`
	}
	
	err = client.Get(ctx, "https://api.binance.com/api/v3/time", &timeResp)
	if err != nil {
		fmt.Printf("自定义配置客户端请求失败: %v\n", err)
		return
	}
	
	fmt.Printf("自定义配置客户端请求成功，服务器时间: %d\n", timeResp.ServerTime)
	fmt.Println("✓ 自定义配置测试成功")
}
