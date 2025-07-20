package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/mooyang-code/data-miner/internal/app"
	"github.com/mooyang-code/data-miner/internal/scheduler"
	"github.com/mooyang-code/data-miner/internal/types"
	"github.com/mooyang-code/data-miner/pkg/utils"
)

var (
	configPath = flag.String("config", "./config/config.yaml", "配置文件路径")
	version    = flag.Bool("version", false, "显示版本信息")
	help       = flag.Bool("help", false, "显示帮助信息")
)

func main() {
	// 解析命令行参数
	if shouldExit := parseFlags(); shouldExit {
		return
	}

	// 加载配置
	config, err := utils.LoadConfig(*configPath)
	if err != nil {
		fmt.Printf("data-miner service配置加载失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	logger, err := initLogger(config.App.LogLevel)
	if err != nil {
		fmt.Printf("data-miner service日志初始化失败: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("启动加密货币数据采集器",
		zap.String("name", config.App.Name),
		zap.String("version", config.App.Version))

	// 初始化系统组件
	ctx := context.Background()
	systemInit := app.NewSystemInitializer(logger, config)

	if err := systemInit.ValidateConfiguration(); err != nil {
		logger.Fatal("data-miner service配置验证失败", zap.Error(err))
	}

	components, err := systemInit.InitializeSystem(ctx)
	if err != nil {
		logger.Fatal("data-miner service系统初始化失败", zap.Error(err))
	}

	logger.Info("系统初始化完成，开始启动应用程序...")

	// 启动应用程序
	if err := startApplication(logger, config, components); err != nil {
		logger.Fatal("data-miner service启动失败", zap.Error(err))
	}
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

// startApplication 启动应用程序
func startApplication(logger *zap.Logger, config *types.Config,
	components *app.SystemComponents) error {

	logger.Info("开始启动应用程序组件...")

	// 初始化各个管理器
	schedulerManager := app.NewSchedulerManager(logger)
	serviceManager := app.NewServiceManager(logger)
	websocketManager := app.NewWebsocketManager(logger)

	logger.Info("管理器初始化完成，开始启动WebSocket...")

	// 启动WebSocket连接（如果启用）
	if err := websocketManager.Start(config, components.Exchanges); err != nil {
		logger.Error("启动WebSocket失败", zap.Error(err))
	}

	logger.Info("WebSocket启动完成，开始设置调度器...")

	// 设置调度器
	sched, err := schedulerManager.Setup(config, components.Exchanges)
	if err != nil {
		return fmt.Errorf("设置调度器失败: %w", err)
	}

	logger.Info("调度器设置完成，开始启动服务...")

	// 启动服务
	if err := serviceManager.Start(config); err != nil {
		return fmt.Errorf("启动服务失败: %w", err)
	}

	logger.Info("所有服务启动完成，进入等待状态...")

	// 等待关闭信号并优雅关闭
	waitForShutdown(logger, sched, components)
	return nil
}

// waitForShutdown 等待关闭信号并优雅关闭
func waitForShutdown(logger *zap.Logger, sched *scheduler.Scheduler,
	components *app.SystemComponents) {

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("数据采集器启动完成，等待退出信号...")
	<-sigChan
	logger.Info("收到退出信号，正在优雅关闭...")

	gracefulShutdown(logger, sched, components)
	logger.Info("程序已退出")
}

// gracefulShutdown 执行优雅关闭逻辑
func gracefulShutdown(logger *zap.Logger, sched *scheduler.Scheduler,
	components *app.SystemComponents) {

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

	// 关闭系统组件
	if err := components.Shutdown(); err != nil {
		logger.Error("关闭系统组件失败", zap.Error(err))
	}
}

// parseFlags 解析命令行参数
func parseFlags() bool {
	flag.Parse()

	if *help {
		showHelp()
		return true
	}

	if *version {
		showVersion()
		return true
	}
	return false
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
	fmt.Println("        配置文件路径 (默认 \"./config.yaml\")")
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
