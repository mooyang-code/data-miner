package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/mooyang-code/data-miner/internal/exchanges/binance"
	"github.com/mooyang-code/data-miner/internal/scheduler"
	"github.com/mooyang-code/data-miner/internal/types"
	"github.com/mooyang-code/data-miner/pkg/utils"
)

var (
	configPath = flag.String("config", "./config.yaml", "配置文件路径")
	version    = flag.Bool("version", false, "显示版本信息")
	help       = flag.Bool("help", false, "显示帮助信息")
)

func main() {
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	if *version {
		showVersion()
		return
	}

	// 加载配置
	config, err := utils.LoadConfig(*configPath)
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	logger, err := initLogger(config.App.LogLevel)
	if err != nil {
		fmt.Printf("初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("启动加密货币数据采集器",
		zap.String("name", config.App.Name),
		zap.String("version", config.App.Version))

	// 初始化交易所
	exchanges := make(map[string]types.ExchangeInterface)

	// 初始化Binance交易所
	if config.Exchanges.Binance.Enabled {
		binanceExchange := binance.New()
		if err := binanceExchange.Initialize(config.Exchanges.Binance); err != nil {
			logger.Fatal("初始化Binance交易所失败", zap.Error(err))
		}
		exchanges["binance"] = binanceExchange
		logger.Info("Binance交易所初始化成功")

		// 如果启用websocket模式，启动websocket连接
		if config.Exchanges.Binance.UseWebsocket {
			logger.Info("启动Binance WebSocket模式")
			if err := startBinanceWebsocket(binanceExchange, config.Exchanges.Binance, logger); err != nil {
				logger.Error("启动Binance WebSocket失败", zap.Error(err))
			}
		} else {
			logger.Info("使用定时API拉取模式")
		}
	}

	// 创建数据处理回调函数
	dataCallback := func(data types.MarketData) error {
		logger.Info("收到市场数据",
			zap.String("exchange", string(data.GetExchange())),
			zap.String("symbol", string(data.GetSymbol())),
			zap.String("type", string(data.GetDataType())),
			zap.Time("timestamp", data.GetTimestamp()))

		// 这里可以添加数据存储逻辑
		return saveData(data, config.Storage)
	}

	// 初始化调度器（仅在非websocket模式下启动）
	var sched *scheduler.Scheduler
	if config.Scheduler.Enabled && !config.Exchanges.Binance.UseWebsocket {
		sched = scheduler.New(logger, exchanges, dataCallback)

		// 添加任务
		for _, job := range config.Scheduler.Jobs {
			if err := sched.AddJob(job); err != nil {
				logger.Error("添加任务失败",
					zap.String("job", job.Name),
					zap.Error(err))
			} else {
				logger.Info("添加任务成功", zap.String("job", job.Name))
			}
		}

		// 启动调度器
		if err := sched.Start(); err != nil {
			logger.Fatal("启动调度器失败", zap.Error(err))
		}
		logger.Info("调度器启动成功")
	} else if config.Exchanges.Binance.UseWebsocket {
		logger.Info("WebSocket模式下跳过调度器启动")
	}

	// 启动健康检查服务（如果启用）
	if config.Monitoring.Enabled {
		go startHealthCheck(config.Monitoring.HealthCheckPort, logger)
		logger.Info("健康检查服务启动",
			zap.Int("port", config.Monitoring.HealthCheckPort))
	}

	// 等待信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("数据采集器启动完成，等待退出信号...")
	<-sigChan

	logger.Info("收到退出信号，正在优雅关闭...")

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 停止调度器
	if sched != nil {
		if err := sched.Stop(ctx); err != nil {
			logger.Error("停止调度器失败", zap.Error(err))
		} else {
			logger.Info("调度器已停止")
		}
	}

	// 关闭交易所连接
	for name, exchange := range exchanges {
		if err := exchange.Close(); err != nil {
			logger.Error("关闭交易所连接失败",
				zap.String("exchange", name),
				zap.Error(err))
		} else {
			logger.Info("交易所连接已关闭", zap.String("exchange", name))
		}
	}

	logger.Info("程序已退出")
}

// initLogger 初始化日志配置
func initLogger(level string) (*zap.Logger, error) {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zapLevel)
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}
	config.Encoding = "console"
	config.EncoderConfig.TimeKey = "time"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	return config.Build()
}

// saveData 保存数据
func saveData(data types.MarketData, storageConfig types.StorageConfig) error {
	// 这里可以实现具体的数据存储逻辑
	// 例如保存到文件、数据库等
	if storageConfig.File.Enabled {
		// 简单的文件存储实现
		// TODO: 实现具体的文件存储逻辑
	}
	fmt.Printf("###data:%+v\n", data)
	return nil
}

// startHealthCheck 启动健康检查服务
func startHealthCheck(port int, logger *zap.Logger) {
	// TODO: 实现HTTP健康检查服务
	logger.Info("健康检查服务占位符", zap.Int("port", port))
}

// showHelp 显示帮助信息
func showHelp() {
	fmt.Println("加密货币数据采集器")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  data-miner [选项]")
	fmt.Println()
	fmt.Println("选项:")
	fmt.Println("  -config string")
	fmt.Println("        配置文件路径 (默认 \"config/config.yaml\")")
	fmt.Println("  -version")
	fmt.Println("        显示版本信息")
	fmt.Println("  -help")
	fmt.Println("        显示此帮助信息")
}

// showVersion 显示版本信息
func showVersion() {
	fmt.Println("加密货币数据采集器 v1.0.0")
	fmt.Println("构建时间: ", time.Now().Format("2006-01-02 15:04:05"))
}

// startBinanceWebsocket 启动Binance WebSocket连接
func startBinanceWebsocket(exchange *binance.Binance, config types.BinanceConfig, logger *zap.Logger) error {
	// 连接WebSocket
	logger.Info("正在连接Binance WebSocket...")
	if err := exchange.WsConnect(); err != nil {
		return fmt.Errorf("连接WebSocket失败: %v", err)
	}

	logger.Info("WebSocket连接成功")

	// 构建订阅频道列表
	var channels []string

	// 添加ticker订阅
	if config.DataTypes.Ticker.Enabled {
		for _, symbol := range config.DataTypes.Ticker.Symbols {
			channel := fmt.Sprintf("%s@ticker", strings.ToLower(symbol))
			channels = append(channels, channel)
		}
	}

	// 添加trade订阅
	if config.DataTypes.Trades.Enabled {
		for _, symbol := range config.DataTypes.Trades.Symbols {
			channel := fmt.Sprintf("%s@trade", strings.ToLower(symbol))
			channels = append(channels, channel)
		}
	}

	// 添加kline订阅
	if config.DataTypes.Klines.Enabled {
		for _, symbol := range config.DataTypes.Klines.Symbols {
			for _, interval := range config.DataTypes.Klines.Intervals {
				channel := fmt.Sprintf("%s@kline_%s", strings.ToLower(symbol), interval)
				channels = append(channels, channel)
			}
		}
	}

	// 添加orderbook订阅
	if config.DataTypes.Orderbook.Enabled {
		for _, symbol := range config.DataTypes.Orderbook.Symbols {
			channel := fmt.Sprintf("%s@depth%d", strings.ToLower(symbol), config.DataTypes.Orderbook.Depth)
			channels = append(channels, channel)
		}
	}

	// 订阅频道
	if len(channels) > 0 {
		logger.Info("订阅WebSocket频道", zap.Strings("channels", channels))
		if err := exchange.Subscribe(channels); err != nil {
			logger.Error("订阅频道失败", zap.Error(err))
			return err
		}
		logger.Info("频道订阅成功")
	} else {
		logger.Warn("没有配置任何订阅频道")
	}

	return nil
}
