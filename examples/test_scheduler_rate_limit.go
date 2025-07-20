package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
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

	logger.Info("开始测试调度器频控功能")

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
				zap.Time("open_time", v.OpenTime),
				zap.Float64("close_price", v.ClosePrice))
		case *types.Ticker:
			logger.Debug("接收到行情数据",
				zap.String("symbol", string(v.Symbol)),
				zap.Float64("price", v.Price))
		default:
			logger.Debug("接收到其他数据", zap.Any("data", data))
		}
		return nil
	}

	// 创建调度器
	sched := scheduler.New(logger, exchanges, dataCallback, config)

	// 创建测试任务配置（少量交易对，避免频控）
	testJobs := []types.JobConfig{
		{
			Name:     "test_binance_klines_small",
			Exchange: "binance",
			DataType: "klines",
			Cron:     "*/30 * * * * *", // 每30秒执行一次
		},
	}

	// 临时修改配置，减少交易对数量
	originalSymbols := config.Exchanges.Binance.DataTypes.Klines.Symbols
	config.Exchanges.Binance.DataTypes.Klines.Symbols = []string{"BTCUSDT", "ETHUSDT", "BNBUSDT"}
	
	logger.Info("临时修改配置",
		zap.Strings("original_symbols", originalSymbols),
		zap.Strings("test_symbols", config.Exchanges.Binance.DataTypes.Klines.Symbols))

	// 添加测试任务
	for _, job := range testJobs {
		if err := sched.AddJob(job); err != nil {
			logger.Error("添加任务失败", zap.String("job", job.Name), zap.Error(err))
		} else {
			logger.Info("添加任务成功", zap.String("job", job.Name))
		}
	}

	// 启动调度器
	if err := sched.Start(); err != nil {
		logger.Fatal("启动调度器失败", zap.Error(err))
	}

	logger.Info("调度器已启动，开始监控频控状态...")

	// 创建上下文和取消函数
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听系统信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 定期显示频控状态
	statusTicker := time.NewTicker(30 * time.Second)
	defer statusTicker.Stop()

	// 运行监控循环
	for {
		select {
		case <-sigChan:
			logger.Info("接收到停止信号，正在关闭...")
			cancel()
			
			// 停止调度器
			stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer stopCancel()
			
			if err := sched.Stop(stopCtx); err != nil {
				logger.Error("停止调度器失败", zap.Error(err))
			} else {
				logger.Info("调度器已停止")
			}
			return

		case <-statusTicker.C:
			// 显示任务状态
			jobStatus := sched.GetJobStatus()
			logger.Info("任务状态报告")
			for name, job := range jobStatus {
				logger.Info("任务详情",
					zap.String("name", name),
					zap.String("status", string(job.Status)),
					zap.Int64("run_count", job.RunCount),
					zap.Int64("error_count", job.ErrorCount),
					zap.Time("last_run", job.LastRun),
					zap.Time("next_run", job.NextRun),
					zap.String("last_error", job.LastError))
			}

			// 显示频控状态
			rateLimitStatus := sched.GetRateLimitStatus()
			logger.Info("频控状态报告", zap.Any("rate_limit_status", rateLimitStatus))

		case <-ctx.Done():
			logger.Info("上下文已取消，退出监控循环")
			return
		}
	}
}
