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

	logger.Info("开始测试超时修复")

	// 加载配置
	config, err := utils.LoadConfig("../config/config.yaml")
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	// 创建Binance交易所
	binanceExchange := binance.New()
	if err := binanceExchange.Initialize(config.Exchanges.Binance); err != nil {
		logger.Fatal("Failed to initialize Binance exchange", zap.Error(err))
	}

	// 创建交易所映射
	exchanges := map[string]types.ExchangeInterface{
		"binance": binanceExchange,
	}

	// 创建数据处理回调函数
	dataCallback := func(data types.MarketData) error {
		switch v := data.(type) {
		case *types.Kline:
			logger.Debug("接收到K线数据",
				zap.String("symbol", string(v.Symbol)),
				zap.String("interval", v.Interval),
				zap.Time("open_time", v.OpenTime))
		default:
			logger.Debug("接收到其他数据")
		}
		return nil
	}

	// 创建调度器
	sched := scheduler.New(logger, exchanges, dataCallback, config)

	// 测试大量交易对的处理（模拟原始错误场景）
	logger.Info("测试大量交易对的超时处理")

	// 临时设置大量交易对来测试超时处理
	originalSymbols := config.Exchanges.Binance.DataTypes.Klines.Symbols
	testSymbols := []string{
		"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "XRPUSDT",
		"SOLUSDT", "DOTUSDT", "DOGEUSDT", "AVAXUSDT", "SHIBUSDT",
		"MATICUSDT", "LTCUSDT", "TRXUSDT", "ETCUSDT", "LINKUSDT",
		"BCHUSDT", "XLMUSDT", "ATOMUSDT", "FILUSDT", "VETUSDT",
		"ICPUSDT", "FTMUSDT", "HBARUSDT", "ALGOUSDT", "AXSUSDT",
		"SANDUSDT", "MANAUSDT", "THETAUSDT", "EGLDUSDT", "KLAYUSDT",
		"ROSEUSDT", "NEARUSDT", "FLOWUSDT", "CHZUSDT", "ENJUSDT",
		"GALAUSDT", "LRCUSDT", "BATUSDT", "ZILUSDT", "WAVESUSDT",
		"ONEUSDT", "HOTUSDT", "ZECUSDT", "OMGUSDT", "QTUMUSDT",
		"ICXUSDT", "RVNUSDT", "IOSTUSDT", "CELRUSDT", "ZENUSDT",
		"FETUSDT", "CVCUSDT", "REQUSDT", "LSKUSDT", "BNTUSDT",
		"STORJUSDT", "MTLUSDT", "KNCUSDT", "REPUSDT", "LENDUSDT",
		"COMPUSDT", "SNXUSDT", "MKRUSDT", "DAIUSDT", "AAVEUSDT",
		"YFIUSDT", "BALUSDT", "CRVUSDT", "SUSHIUSDT", "UNIUSDT",
		"ALPHAUSDT", "AUDIOUSDT", "CTSIUSDT", "DUSKUSDT", "BELUSDT",
		"WINGUSDT", "CREAMUSDT", "HEGICUSDT", "PEARLUSDT", "INJUSDT",
		"AEROUSDT", "NKNUSDT", "OGNUSDT", "LTOLUSDT", "NBSUSDT",
		"OXTUSDT", "SUNUSDT", "AVAUSDT", "KEYUSDT", "TRBUSDT",
		"BZRXUSDT", "SUSHIBUSDT", "YFIIUSDT", "KSMAUSDT", "EGGSUSDT",
		"DIAUSDT", "RUNEUSDT", "THORUSDT", "CTXCUSDT", "BONDUSDT",
	}
	
	config.Exchanges.Binance.DataTypes.Klines.Symbols = testSymbols
	
	logger.Info("设置测试配置",
		zap.Strings("original_symbols", originalSymbols),
		zap.Int("test_symbols_count", len(testSymbols)))

	// 创建测试任务
	testJob := types.JobConfig{
		Name:     "test_timeout_fix",
		Exchange: "binance",
		DataType: "klines",
		Cron:     "0 * * * * *", // 立即执行一次
	}

	// 添加任务
	if err := sched.AddJob(testJob); err != nil {
		logger.Fatal("添加任务失败", zap.Error(err))
	}

	// 启动调度器
	if err := sched.Start(); err != nil {
		logger.Fatal("启动调度器失败", zap.Error(err))
	}

	logger.Info("调度器已启动，等待任务执行...")

	// 等待任务执行完成
	time.Sleep(10 * time.Minute) // 等待足够长的时间

	// 检查任务状态
	jobStatus := sched.GetJobStatus()
	for name, job := range jobStatus {
		logger.Info("任务执行结果",
			zap.String("name", name),
			zap.String("status", string(job.Status)),
			zap.Int64("run_count", job.RunCount),
			zap.Int64("error_count", job.ErrorCount),
			zap.String("last_error", job.LastError))
	}

	// 显示频控状态
	rateLimitStatus := sched.GetRateLimitStatus()
	logger.Info("最终频控状态", zap.Any("rate_limit_status", rateLimitStatus))

	// 停止调度器
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	if err := sched.Stop(ctx); err != nil {
		logger.Error("停止调度器失败", zap.Error(err))
	} else {
		logger.Info("调度器已停止")
	}

	logger.Info("超时修复测试完成")
}
