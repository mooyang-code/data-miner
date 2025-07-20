package main

import (
	"context"
	"log"
	"time"

	"go.uber.org/zap"

	"github.com/mooyang-code/data-miner/internal/exchanges/binance"
	"github.com/mooyang-code/data-miner/internal/scheduler"
	"github.com/mooyang-code/data-miner/internal/types"
	"github.com/mooyang-code/data-miner/pkg/utils"
)

func main() {
	// 初始化日志
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Sync()

	logger.Info("开始测试简化频控管理器")

	// 加载配置
	config, err := utils.LoadConfig("../config/config.yaml")
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	// 创建Binance交易所实例
	binanceExchange := binance.New()
	if err := binanceExchange.Initialize(config.Exchanges.Binance); err != nil {
		logger.Fatal("Failed to initialize Binance exchange", zap.Error(err))
	}

	// 创建频控管理器
	rateLimitMgr := scheduler.NewRateLimitManager(logger)

	// 测试权重检查功能
	logger.Info("测试权重检查功能")
	ctx := context.Background()
	
	// 检查当前权重
	if err := rateLimitMgr.CheckAndWaitIfNeeded(ctx, binanceExchange); err != nil {
		logger.Error("权重检查失败", zap.Error(err))
	}

	// 显示频控状态
	status := rateLimitMgr.GetStatus()
	logger.Info("频控管理器状态", zap.Any("status", status))

	// 测试少量交易对的批量处理
	logger.Info("测试少量交易对的批量处理")
	
	// 创建小的测试交易对列表
	testSymbols := []types.Symbol{
		"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "XRPUSDT",
	}

	logger.Info("开始批量处理测试", zap.Int("total_symbols", len(testSymbols)))

	// 模拟K线数据处理
	processor := func(batch []types.Symbol) error {
		logger.Info("处理批次", 
			zap.Int("batch_size", len(batch)),
			zap.String("first_symbol", string(batch[0])),
			zap.String("last_symbol", string(batch[len(batch)-1])))

		// 模拟处理每个交易对（不实际调用API）
		for _, symbol := range batch {
			logger.Debug("模拟处理交易对", zap.String("symbol", string(symbol)))
			// 添加小延迟模拟处理时间
			time.Sleep(100 * time.Millisecond)
		}

		return nil
	}

	// 执行批量处理
	startTime := time.Now()
	if err := rateLimitMgr.ProcessInBatches(ctx, testSymbols, binanceExchange, processor); err != nil {
		logger.Error("批量处理失败", zap.Error(err))
	} else {
		duration := time.Since(startTime)
		logger.Info("批量处理完成", 
			zap.Duration("total_duration", duration),
			zap.Float64("symbols_per_second", float64(len(testSymbols))/duration.Seconds()))
	}

	// 显示最终状态
	finalStatus := rateLimitMgr.GetStatus()
	logger.Info("最终频控状态", zap.Any("status", finalStatus))

	// 测试权重估算功能
	logger.Info("测试权重估算功能")
	
	klinesWeight := rateLimitMgr.EstimateWeight("klines", 10)
	tickerWeight := rateLimitMgr.EstimateWeight("ticker", 50)
	orderbookWeight := rateLimitMgr.EstimateWeight("orderbook", 5)
	
	logger.Info("权重估算结果",
		zap.Int("klines_weight_10", klinesWeight),
		zap.Int("ticker_weight_50", tickerWeight),
		zap.Int("orderbook_weight_5", orderbookWeight))

	logger.Info("简化频控管理器测试完成")
}
