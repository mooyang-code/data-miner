package main

import (
	"context"
	"fmt"
	"github.com/mooyang-code/data-miner/internal/app"
	"log"
	"time"

	"go.uber.org/zap"

	"github.com/mooyang-code/data-miner/internal/exchanges/binance"
	"github.com/mooyang-code/data-miner/internal/types"
	"github.com/mooyang-code/data-miner/pkg/cryptotrader/currency"
	"github.com/mooyang-code/data-miner/pkg/cryptotrader/exchanges/asset"
)

func main() {
	fmt.Println("=== Binance 交易对缓存管理示例 ===")

	// 创建日志记录器
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("Failed to create logger:", err)
	}
	defer logger.Sync()

	// 创建配置
	config := &types.Config{
		Exchanges: types.ExchangesConfig{
			Binance: types.BinanceConfig{
				Enabled:      true,
				APIURL:       "https://api.binance.com",
				WebsocketURL: "wss://stream.binance.com:9443",
				TradablePairs: types.TradablePairsConfig{
					FetchFromAPI:    true,
					UpdateInterval:  30 * time.Second, // 演示用较短间隔
					CacheEnabled:    true,
					CacheTTL:        2 * time.Minute,
					SupportedAssets: []string{"spot", "margin"},
					AutoUpdate:      true,
				},
			},
		},
	}

	ctx := context.Background()

	// 1. 使用系统初始化器
	fmt.Println("\n1. 初始化系统...")
	initializer := app.NewSystemInitializer(logger, config)

	// 验证配置
	if err := initializer.ValidateConfiguration(); err != nil {
		log.Fatal("Configuration validation failed:", err)
	}

	// 初始化系统
	components, err := initializer.InitializeSystem(ctx)
	if err != nil {
		log.Fatal("Failed to initialize system:", err)
	}
	defer components.Shutdown()

	// 获取Binance交易所实例
	binanceExchange, err := components.GetBinanceExchange()
	if err != nil {
		log.Fatal("Failed to get Binance exchange:", err)
	}

	// 2. 演示缓存功能
	fmt.Println("\n2. 演示缓存功能...")

	// 获取缓存统计信息
	stats := binanceExchange.GetTradablePairsStats()
	fmt.Printf("缓存统计信息: %+v\n", stats)

	// 3. 获取交易对
	fmt.Println("\n3. 获取交易对...")

	// 从缓存获取现货交易对
	spotPairs, err := binanceExchange.GetTradablePairsFromCache(ctx, asset.Spot)
	if err != nil {
		log.Printf("获取现货交易对失败: %v", err)
		return
	}
	fmt.Printf("现货交易对数量: %d\n", len(spotPairs))

	// 显示前10个现货交易对
	fmt.Println("前10个现货交易对:")
	for i, pair := range spotPairs {
		if i >= 10 {
			break
		}
		fmt.Printf("  %d. %s\n", i+1, pair.String())
	}

	// 从缓存获取保证金交易对
	marginPairs, err := binanceExchange.GetTradablePairsFromCache(ctx, asset.Margin)
	if err != nil {
		log.Printf("获取保证金交易对失败: %v", err)
		return
	}
	fmt.Printf("保证金交易对数量: %d\n", len(marginPairs))

	// 4. 演示交易对解析功能
	fmt.Println("\n4. 演示交易对解析功能...")

	// 使用["*"]获取所有交易对
	allSymbols, err := binanceExchange.ResolveTradingPairs(ctx, []string{"*"}, asset.Spot)
	if err != nil {
		log.Printf("解析所有交易对失败: %v", err)
		return
	}
	fmt.Printf("使用['*']解析得到 %d 个交易对\n", len(allSymbols))

	// 使用具体交易对列表
	specificSymbols, err := binanceExchange.ResolveTradingPairs(ctx, []string{"BTCUSDT", "ETHUSDT", "BNBUSDT"}, asset.Spot)
	if err != nil {
		log.Printf("解析具体交易对失败: %v", err)
		return
	}
	fmt.Printf("具体交易对: %v\n", specificSymbols)

	// 5. 演示交易对支持检查
	fmt.Println("\n5. 演示交易对支持检查...")

	testSymbols := []string{"BTCUSDT", "ETHUSDT", "FAKEUSDT"}
	for _, symbolStr := range testSymbols {
		pair, err := currency.NewPairFromString(symbolStr)
		if err != nil {
			fmt.Printf("创建交易对 %s 失败: %v\n", symbolStr, err)
			continue
		}

		supported, err := binanceExchange.IsSymbolSupported(ctx, pair, asset.Spot)
		if err != nil {
			fmt.Printf("检查交易对 %s 支持状态失败: %v\n", symbolStr, err)
			continue
		}

		status := "不支持"
		if supported {
			status = "支持"
		}
		fmt.Printf("  %s: %s\n", symbolStr, status)
	}

	// 6. 演示缓存性能
	fmt.Println("\n6. 演示缓存性能...")

	// 第一次调用（可能需要从API获取）
	start := time.Now()
	_, err = binanceExchange.GetTradablePairsFromCache(ctx, asset.Spot)
	if err != nil {
		log.Printf("第一次获取失败: %v", err)
		return
	}
	firstCallDuration := time.Since(start)
	fmt.Printf("第一次调用耗时: %v\n", firstCallDuration)

	// 第二次调用（从缓存获取）
	start = time.Now()
	_, err = binanceExchange.GetTradablePairsFromCache(ctx, asset.Spot)
	if err != nil {
		log.Printf("第二次获取失败: %v", err)
		return
	}
	secondCallDuration := time.Since(start)
	fmt.Printf("第二次调用耗时: %v\n", secondCallDuration)

	fmt.Printf("缓存加速比: %.2fx\n", float64(firstCallDuration)/float64(secondCallDuration))

	// 7. 演示系统状态
	fmt.Println("\n7. 系统状态...")
	systemStatus := components.GetSystemStatus()
	fmt.Printf("系统状态: %+v\n", systemStatus)

	// 8. 演示自动更新
	fmt.Println("\n8. 演示自动更新...")
	fmt.Println("等待自动更新（30秒间隔）...")

	// 获取当前统计
	beforeStats := binanceExchange.GetTradablePairsStats()
	fmt.Printf("更新前统计: %+v\n", beforeStats)

	// 等待自动更新
	time.Sleep(35 * time.Second)

	// 获取更新后统计
	afterStats := binanceExchange.GetTradablePairsStats()
	fmt.Printf("更新后统计: %+v\n", afterStats)

	// 9. 演示配置不同场景
	fmt.Println("\n9. 演示不同配置场景...")

	// 创建不启用缓存的配置
	noCacheConfig := types.BinanceConfig{
		Enabled: true,
		APIURL:  "https://api.binance.com",
		TradablePairs: types.TradablePairsConfig{
			FetchFromAPI: false, // 不启用缓存
		},
	}

	// 创建新的Binance实例
	noCacheBinance := binance.New()
	noCacheBinance.SetLogger(logger.Named("no-cache"))

	if err := noCacheBinance.Initialize(noCacheConfig); err != nil {
		log.Printf("初始化无缓存实例失败: %v", err)
		return
	}

	// 测试无缓存模式
	start = time.Now()
	noCachePairs, err := noCacheBinance.GetTradablePairsFromCache(ctx, asset.Spot)
	if err != nil {
		log.Printf("无缓存模式获取交易对失败: %v", err)
		return
	}
	noCacheDuration := time.Since(start)

	fmt.Printf("无缓存模式获取 %d 个交易对，耗时: %v\n", len(noCachePairs), noCacheDuration)

	// 对比缓存和无缓存的性能
	start = time.Now()
	cachedPairs, err := binanceExchange.GetTradablePairsFromCache(ctx, asset.Spot)
	if err != nil {
		log.Printf("缓存模式获取交易对失败: %v", err)
		return
	}
	cachedDuration := time.Since(start)

	fmt.Printf("缓存模式获取 %d 个交易对，耗时: %v\n", len(cachedPairs), cachedDuration)
	fmt.Printf("缓存性能提升: %.2fx\n", float64(noCacheDuration)/float64(cachedDuration))

	fmt.Println("\n=== 示例完成 ===")
}
