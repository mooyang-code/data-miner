// Package app 应用程序核心管理模块
package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/mooyang-code/data-miner/internal/scheduler"
	"github.com/mooyang-code/data-miner/internal/types"
	"github.com/mooyang-code/data-miner/pkg/utils"
)

// Manager 应用程序管理器
type Manager struct {
	config           *types.Config
	logger           *zap.Logger
	exchangeManager  *ExchangeManager
	schedulerManager *SchedulerManager
	serviceManager   *ServiceManager
	websocketManager *WebsocketManager
}

// New 创建新的应用程序管理器
func New() *Manager {
	return &Manager{}
}

// Initialize 初始化应用程序
func (m *Manager) Initialize(configPath string) error {
	// 加载配置
	config, err := utils.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("加载配置失败: %v", err)
	}
	m.config = config

	// 初始化日志
	logger, err := m.initLogger(config.App.LogLevel)
	if err != nil {
		return fmt.Errorf("初始化日志失败: %v", err)
	}
	m.logger = logger

	m.logger.Info("启动加密货币数据采集器",
		zap.String("name", config.App.Name),
		zap.String("version", config.App.Version))

	// 初始化各个管理器
	m.exchangeManager = NewExchangeManager(m.logger)
	m.schedulerManager = NewSchedulerManager(m.logger)
	m.serviceManager = NewServiceManager(m.logger)
	m.websocketManager = NewWebsocketManager(m.logger)
	return nil
}

// Start 启动应用程序
func (m *Manager) Start() error {
	// 初始化交易所
	exchanges, err := m.exchangeManager.Initialize(m.config)
	if err != nil {
		return fmt.Errorf("初始化交易所失败: %v", err)
	}

	// 启动WebSocket连接（如果启用）
	if err := m.websocketManager.Start(m.config, exchanges); err != nil {
		m.logger.Error("启动WebSocket失败", zap.Error(err))
	}

	// 设置调度器
	scheduler, err := m.schedulerManager.Setup(m.config, exchanges)
	if err != nil {
		return fmt.Errorf("设置调度器失败: %v", err)
	}

	// 启动服务
	if err := m.serviceManager.Start(m.config); err != nil {
		return fmt.Errorf("启动服务失败: %v", err)
	}

	// 等待关闭信号
	m.waitForShutdown(scheduler, exchanges)
	return nil
}

// GetLogger 获取日志器
func (m *Manager) GetLogger() *zap.Logger {
	return m.logger
}

// Sync 同步日志
func (m *Manager) Sync() {
	if m.logger != nil {
		m.logger.Sync()
	}
}

// waitForShutdown 等待关闭信号并优雅关闭
func (m *Manager) waitForShutdown(sched *scheduler.Scheduler, exchanges map[string]types.ExchangeInterface) {
	// 等待信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	m.logger.Info("数据采集器启动完成，等待退出信号...")
	<-sigChan
	m.logger.Info("收到退出信号，正在优雅关闭...")

	// 执行优雅关闭
	m.gracefulShutdown(sched, exchanges)
	m.logger.Info("程序已退出")
}

// gracefulShutdown 执行优雅关闭逻辑
func (m *Manager) gracefulShutdown(sched *scheduler.Scheduler, exchanges map[string]types.ExchangeInterface) {
	// 创建超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 停止调度器
	if sched != nil {
		if err := sched.Stop(ctx); err != nil {
			m.logger.Error("停止调度器失败", zap.Error(err))
		} else {
			m.logger.Info("调度器已停止")
		}
	}

	// 关闭交易所连接
	for name, exchange := range exchanges {
		if err := exchange.Close(); err != nil {
			m.logger.Error("关闭交易所连接失败",
				zap.String("exchange", name),
				zap.Error(err))
		} else {
			m.logger.Info("交易所连接已关闭", zap.String("exchange", name))
		}
	}
}

// initLogger 初始化日志配置
func (m *Manager) initLogger(level string) (*zap.Logger, error) {
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
