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

	logger.Info("开始测试频控管理器")

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

	// 测试批量处理功能
	logger.Info("测试批量处理功能")
	
	// 创建测试交易对列表
	testSymbols := []types.Symbol{
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
		"MLNUSDT", "DEXEUSDT", "C98USDT", "CLVUSDT", "QNTUSDT",
		"FLOWUSDT", "TVKUSDT", "BADGERUSDT", "FISUSDT", "OMAUSDT",
		"PONDUSDT", "DEGOUSDT", "ALICEUSDT", "LINAUSDT", "PERPUSDT",
		"RAMPUSDT", "SUPERUSDT", "CFXUSDT", "EPSUSDT", "AUTOUSDT",
		"TKOUSDT", "PUNDIXUSDT", "TLMUSDT", "MIRUSDT", "BARUSDT",
		"FORTHUSDT", "BAKEUSDT", "BURGERUSDT", "SLPUSDT", "SXPUSDT",
		"CKBUSDT", "TWTUSDT", "FIROUSDT", "LITUSDT", "SFPUSDT",
		"DODOUSDT", "CAKEUSDT", "ACMUSDT", "BADGERUSDT", "FISUSDT",
		"OMAUSDT", "PONDUSDT", "DEGOUSDT", "ALICEUSDT", "LINAUSDT",
	}

	logger.Info("开始批量处理测试", zap.Int("total_symbols", len(testSymbols)))

	// 模拟K线数据处理
	processor := func(batch []types.Symbol) error {
		logger.Info("处理批次", 
			zap.Int("batch_size", len(batch)),
			zap.String("first_symbol", string(batch[0])),
			zap.String("last_symbol", string(batch[len(batch)-1])))

		// 模拟处理每个交易对
		for _, symbol := range batch {
			// 模拟获取K线数据
			klines, err := binanceExchange.GetKlines(ctx, symbol, "1m", 10)
			if err != nil {
				logger.Error("获取K线数据失败", 
					zap.String("symbol", string(symbol)), 
					zap.Error(err))
				continue
			}
			
			logger.Debug("获取K线数据成功", 
				zap.String("symbol", string(symbol)),
				zap.Int("klines_count", len(klines)))
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

	logger.Info("频控管理器测试完成")
}
