// Package main 测试fetch_from_api配置开关功能
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.uber.org/zap"

	"github.com/mooyang-code/data-miner/internal/exchanges/binance"
	"github.com/mooyang-code/data-miner/internal/scheduler"
	"github.com/mooyang-code/data-miner/internal/types"
)

func main() {
	// 创建日志记录器
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("创建日志记录器失败:", err)
	}
	defer logger.Sync()

	// 测试场景1: fetch_from_api = true
	fmt.Println("=== 测试场景1: fetch_from_api = true ===")
	testWithFetchFromAPI(logger, true)

	fmt.Println("\n=== 测试场景2: fetch_from_api = false ===")
	testWithFetchFromAPI(logger, false)
}

func testWithFetchFromAPI(logger *zap.Logger, fetchFromAPI bool) {
	// 创建测试配置
	config := &types.Config{
		Exchanges: types.ExchangesConfig{
			Binance: types.BinanceConfig{
				Enabled:      true,
				APIURL:       "https://api.binance.com",
				WebsocketURL: "wss://stream.binance.com:9443",
				UseWebsocket: false,
				TradablePairs: types.TradablePairsConfig{
					FetchFromAPI:     fetchFromAPI,
					UpdateInterval:   1 * time.Hour,
					CacheEnabled:     true,
					CacheTTL:         2 * time.Hour,
					SupportedAssets:  []string{"spot", "margin"},
					AutoUpdate:       true,
				},
				DataTypes: types.BinanceDataTypes{
					Ticker: types.TickerConfig{
						Enabled:  true,
						Symbols:  []string{"*"}, // 使用通配符触发缓存获取
						Interval: "1m",
					},
					Klines: types.KlinesConfig{
						Enabled:   true,
						Symbols:   []string{"*"}, // 使用通配符触发缓存获取
						Intervals: []string{"1m", "5m"},
						Interval:  "1m",
					},
				},
			},
		},
		Scheduler: types.SchedulerConfig{
			Enabled:           true,
			MaxConcurrentJobs: 5,
			Jobs: []types.JobConfig{
				{
					Name:     "test_ticker",
					Exchange: "binance",
					DataType: "ticker",
					Cron:     "0 * * * * *",
				},
			},
		},
	}

	// 创建Binance交易所实例
	b := binance.New()
	b.SetLogger(logger.Named("binance"))

	// 初始化交易所
	if err := b.Initialize(config.Exchanges.Binance); err != nil {
		logger.Error("初始化Binance失败", zap.Error(err))
		return
	}

	// 如果启用了fetch_from_api，启动缓存
	if fetchFromAPI {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := b.StartTradablePairsCache(ctx); err != nil {
			logger.Error("启动交易对缓存失败", zap.Error(err))
			return
		}
		logger.Info("交易对缓存启动成功")

		// 等待缓存初始化
		time.Sleep(3 * time.Second)
	}

	// 创建交易所映射
	exchanges := map[string]types.ExchangeInterface{
		"binance": b,
	}

	// 创建数据回调函数
	dataCallback := func(data types.MarketData) error {
		logger.Info("收到数据",
			zap.String("exchange", string(data.GetExchange())),
			zap.String("symbol", string(data.GetSymbol())),
			zap.String("type", string(data.GetDataType())))
		return nil
	}

	// 创建调度器
	sched := scheduler.New(logger.Named("scheduler"), exchanges, dataCallback, config)

	// 测试获取交易对
	logger.Info("开始测试获取交易对...")

	// 直接调用内部方法进行测试（这里我们需要访问私有方法，所以创建一个简单的测试）
	// 由于getTradablePairsFromCache是私有方法，我们通过添加任务来间接测试
	testJob := types.JobConfig{
		Name:     "test_job",
		Exchange: "binance",
		DataType: "ticker",
		Cron:     "0 * * * * *",
	}

	if err := sched.AddJob(testJob); err != nil {
		logger.Error("添加测试任务失败", zap.Error(err))
		return
	}

	logger.Info("测试完成",
		zap.Bool("fetch_from_api", fetchFromAPI))

	// 清理
	if err := sched.Stop(); err != nil {
		logger.Error("停止调度器失败", zap.Error(err))
	}
}
