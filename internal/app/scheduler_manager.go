package app

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/mooyang-code/data-miner/internal/scheduler"
	"github.com/mooyang-code/data-miner/internal/types"
)

// SchedulerManager 调度器管理器
type SchedulerManager struct {
	logger *zap.Logger
}

// NewSchedulerManager 创建新的调度器管理器
func NewSchedulerManager(logger *zap.Logger) *SchedulerManager {
	return &SchedulerManager{
		logger: logger,
	}
}

// Setup 设置调度器
func (sm *SchedulerManager) Setup(config *types.Config, exchanges map[string]types.ExchangeInterface) (*scheduler.Scheduler, error) {
	sm.logger.Info("开始设置调度器...",
		zap.Bool("scheduler_enabled", config.Scheduler.Enabled),
		zap.Bool("use_websocket", config.Exchanges.Binance.UseWebsocket))

	// 创建数据处理回调函数
	dataCallback := sm.createDataCallback(config)

	// 初始化调度器（仅在非websocket模式下启动）
	var sched *scheduler.Scheduler
	if config.Scheduler.Enabled && !config.Exchanges.Binance.UseWebsocket {
		sm.logger.Info("创建调度器实例...")
		sched = scheduler.New(sm.logger, exchanges, dataCallback, config)

		// 添加任务
		sm.logger.Info("开始添加任务...", zap.Int("job_count", len(config.Scheduler.Jobs)))
		for _, job := range config.Scheduler.Jobs {
			sm.logger.Info("正在添加任务",
				zap.String("job_name", job.Name),
				zap.String("exchange", job.Exchange),
				zap.String("data_type", job.DataType),
				zap.String("cron", job.Cron))

			if err := sched.AddJob(job); err != nil {
				sm.logger.Error("添加任务失败",
					zap.String("job", job.Name),
					zap.Error(err))
			} else {
				sm.logger.Info("添加任务成功", zap.String("job", job.Name))
			}
		}

		// 启动调度器
		sm.logger.Info("启动调度器...")
		if err := sched.Start(); err != nil {
			sm.logger.Fatal("启动调度器失败", zap.Error(err))
			return nil, err
		}
		sm.logger.Info("调度器启动成功")
	} else if config.Exchanges.Binance.UseWebsocket {
		sm.logger.Info("WebSocket模式下跳过调度器启动")
	} else {
		sm.logger.Info("调度器未启用或条件不满足",
			zap.Bool("scheduler_enabled", config.Scheduler.Enabled),
			zap.Bool("use_websocket", config.Exchanges.Binance.UseWebsocket))
	}

	return sched, nil
}

// createDataCallback 创建数据处理回调函数
func (sm *SchedulerManager) createDataCallback(config *types.Config) func(types.MarketData) error {
	return func(data types.MarketData) error {
		sm.logger.Info("收到市场数据",
			zap.String("exchange", string(data.GetExchange())),
			zap.String("symbol", string(data.GetSymbol())),
			zap.String("type", string(data.GetDataType())),
			zap.Time("timestamp", data.GetTimestamp()))

		// 这里可以添加数据存储逻辑
		return sm.saveData(data, config.Storage)
	}
}

// saveData 保存数据
func (sm *SchedulerManager) saveData(data types.MarketData, storageConfig types.StorageConfig) error {
	// 这里可以实现具体的数据存储逻辑
	// 例如保存到文件、数据库等
	if storageConfig.File.Enabled {
		// 简单的文件存储实现
		// TODO: 实现具体的文件存储逻辑
	}
	fmt.Printf("###data:%+v\n", data)
	return nil
}
