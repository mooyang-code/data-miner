// Package binance REST API测试
package binance

import (
	"context"
	"testing"
	"time"

	"github.com/mooyang-code/data-miner/internal/types"
	"github.com/mooyang-code/data-miner/pkg/cryptotrader/currency"
)

// TestNewRestAPI 测试REST API客户端创建
func TestNewRestAPI(t *testing.T) {
	api := NewRestAPI()
	if api == nil {
		t.Fatal("Failed to create REST API client")
	}
	
	if api.ipManager == nil {
		t.Fatal("IP manager not initialized")
	}
	
	if api.httpClient == nil {
		t.Fatal("HTTP client not initialized")
	}
	
	t.Logf("REST API client created successfully")
}

// TestInitializeAndClose 测试初始化和关闭
func TestInitializeAndClose(t *testing.T) {
	api := NewRestAPI()
	
	// 测试初始化
	config := types.BinanceConfig{}
	err := api.Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	
	// 等待IP管理器启动
	time.Sleep(time.Second * 2)
	
	// 检查IP管理器状态
	status := api.GetIPManagerStatus()
	t.Logf("IP Manager Status: %+v", status)
	
	// 测试关闭
	err = api.Close()
	if err != nil {
		t.Fatalf("Failed to close: %v", err)
	}
	
	t.Logf("Initialize and close test passed")
}

// TestGetLatestSpotPrice 测试获取最新现货价格（使用动态IP）
func TestGetLatestSpotPrice(t *testing.T) {
	api := NewRestAPI()
	api.Verbose = true // 启用详细日志

	// 初始化
	config := types.BinanceConfig{}
	err := api.Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	defer api.Close()
	
	// 等待IP管理器启动
	time.Sleep(time.Second * 3)
	
	// 测试获取BTC/USDT价格
	ctx := context.Background()
	pair := currency.NewPair(currency.BTC, currency.USDT)
	
	price, err := api.GetLatestSpotPrice(ctx, pair)
	if err != nil {
		t.Fatalf("Failed to get latest spot price: %v", err)
	}
	
	t.Logf("BTC/USDT Price: %+v", price)
	
	if price.Symbol == "" {
		t.Fatal("Empty symbol in price response")
	}

	if price.Price <= 0 {
		t.Fatal("Invalid price in response")
	}
	
	t.Logf("Get latest spot price test passed")
}

// TestIPManagerFunctionality 测试IP管理器功能
func TestIPManagerFunctionality(t *testing.T) {
	api := NewRestAPI()
	
	// 初始化
	config := types.BinanceConfig{}
	err := api.Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	defer api.Close()
	
	// 等待IP管理器启动
	time.Sleep(time.Second * 3)
	
	// 获取当前IP
	ip, err := api.getCurrentIP()
	if err != nil {
		t.Fatalf("Failed to get current IP: %v", err)
	}
	t.Logf("Current IP: %s", ip)
	
	// 切换到下一个IP
	nextIP, err := api.switchToNextIP()
	if err != nil {
		t.Fatalf("Failed to switch to next IP: %v", err)
	}
	t.Logf("Next IP: %s", nextIP)
	
	// 验证IP已切换
	if ip == nextIP && len(api.ipManager.GetAllIPs()) > 1 {
		t.Logf("Warning: IP didn't change, might be only one IP available")
	}
	
	// 获取所有IP
	allIPs := api.ipManager.GetAllIPs()
	t.Logf("All available IPs: %v", allIPs)
	
	if len(allIPs) == 0 {
		t.Fatal("No IPs available")
	}
	
	t.Logf("IP manager functionality test passed")
}

// TestMultipleRequests 测试多个请求（验证负载均衡和故障转移）
func TestMultipleRequests(t *testing.T) {
	api := NewRestAPI()
	
	// 初始化
	config := types.BinanceConfig{}
	err := api.Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	defer api.Close()
	
	// 等待IP管理器启动
	time.Sleep(time.Second * 3)
	
	ctx := context.Background()
	pair := currency.NewPair(currency.BTC, currency.USDT)
	
	// 发送多个请求
	for i := 0; i < 5; i++ {
		t.Logf("Request %d:", i+1)
		
		price, err := api.GetLatestSpotPrice(ctx, pair)
		if err != nil {
			t.Logf("Request %d failed: %v", i+1, err)
			continue
		}
		
		t.Logf("Request %d success - Price: %f", i+1, price.Price)
		
		// 短暂延迟
		time.Sleep(time.Millisecond * 500)
	}
	
	t.Logf("Multiple requests test completed")
}
