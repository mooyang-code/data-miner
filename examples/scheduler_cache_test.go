package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.uber.org/zap"

	"github.com/mooyang-code/data-miner/internal/app"
	"github.com/mooyang-code/data-miner/internal/exchanges/binance"
	"github.com/mooyang-code/data-miner/internal/scheduler"
	"github.com/mooyang-code/data-miner/internal/system"
	"github.com/mooyang-code/data-miner/internal/types"
)

func main() {
	fmt.Println("=== Scheduler Cache 配置测试 ===")

	// 初始化日志
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("初始化日志失败:", err)
	}
	defer logger.Sync()

	// 加载配置
	config, err := system.LoadConfig("examples/test_config.yaml")
	if err != nil {
		logger.Fatal("加载配置失败", zap.Error(err))
	}

	// 创建Binance交易所实例
	binanceExchange := binance.New(logger, &config.Exchanges.Binance)

	// 创建交易所映射
	exchanges := map[string]types.ExchangeInterface{
		"binance": binanceExchange,
	}

	// 创建数据回调函数
	dataCallback := func(data types.MarketData) error {
		logger.Info("收到市场数据",
			zap.String("exchange", string(data.GetExchange())),
			zap.String("symbol", string(data.GetSymbol())),
			zap.String("type", string(data.GetDataType())),
			zap.Time("timestamp", data.GetTimestamp()))
		return nil
	}

	// 创建调度器
	sched := scheduler.New(logger, exchanges, dataCallback, config)

	// 测试从配置读取交易对
	fmt.Println("\n=== 测试从配置读取交易对 ===")
	
	// 测试ticker配置
	fmt.Println("1. 测试Ticker配置:")
	fmt.Printf("   配置的symbols: %v\n", config.Exchanges.Binance.DataTypes.Ticker.Symbols)
	
	// 测试orderbook配置
	fmt.Println("2. 测试Orderbook配置:")
	fmt.Printf("   配置的symbols: %v\n", config.Exchanges.Binance.DataTypes.Orderbook.Symbols)
	fmt.Printf("   配置的depth: %d\n", config.Exchanges.Binance.DataTypes.Orderbook.Depth)
	
	// 测试klines配置
	fmt.Println("3. 测试Klines配置:")
	fmt.Printf("   配置的symbols: %v\n", config.Exchanges.Binance.DataTypes.Klines.Symbols)
	fmt.Printf("   配置的intervals: %v\n", config.Exchanges.Binance.DataTypes.Klines.Intervals)

	// 如果配置中有"*"，测试从cache获取交易对
	if len(config.Exchanges.Binance.DataTypes.Ticker.Symbols) > 0 && 
	   config.Exchanges.Binance.DataTypes.Ticker.Symbols[0] == "*" {
		fmt.Println("\n=== 测试从Cache获取交易对 ===")
		
		// 启动交易对缓存
		ctx := context.Background()
		if err := binanceExchange.StartTradablePairsCache(ctx); err != nil {
			logger.Error("启动交易对缓存失败", zap.Error(err))
		} else {
			logger.Info("交易对缓存启动成功")
			
			// 等待一段时间让缓存加载
			time.Sleep(3 * time.Second)
			
			// 测试获取交易对
			fmt.Println("正在从cache获取交易对...")
		}
	}

	// 添加一个测试任务
	fmt.Println("\n=== 添加测试任务 ===")
	testJob := types.JobConfig{
		Name:     "test_ticker",
		Exchange: "binance",
		DataType: "ticker",
		Cron:     "*/30 * * * * *", // 每30秒执行一次
	}

	if err := sched.AddJob(testJob); err != nil {
		logger.Error("添加测试任务失败", zap.Error(err))
	} else {
		logger.Info("测试任务添加成功")
	}

	// 启动调度器
	fmt.Println("\n=== 启动调度器 ===")
	if err := sched.Start(); err != nil {
		logger.Fatal("启动调度器失败", zap.Error(err))
	}

	// 运行一段时间观察结果
	fmt.Println("调度器运行中，观察60秒...")
	time.Sleep(60 * time.Second)

	// 停止调度器
	fmt.Println("\n=== 停止调度器 ===")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := sched.Stop(ctx); err != nil {
		logger.Error("停止调度器失败", zap.Error(err))
	} else {
		logger.Info("调度器已停止")
	}

	// 显示任务状态
	fmt.Println("\n=== 任务执行状态 ===")
	jobStatus := sched.GetJobStatus()
	for name, info := range jobStatus {
		fmt.Printf("任务: %s\n", name)
		fmt.Printf("  状态: %s\n", info.Status)
		fmt.Printf("  运行次数: %d\n", info.RunCount)
		fmt.Printf("  错误次数: %d\n", info.ErrorCount)
		if info.LastError != "" {
			fmt.Printf("  最后错误: %s\n", info.LastError)
		}
		fmt.Printf("  最后运行: %s\n", info.LastRun.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}

	fmt.Println("=== 测试完成 ===")
}
