// Package scheduler 提供任务调度功能
package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/mooyang-code/data-miner/internal/types"
)

// Scheduler 调度器
type Scheduler struct {
	cron      *cron.Cron
	logger    *zap.Logger
	exchanges map[string]types.ExchangeInterface
	callback  types.DataCallback
	jobs      map[string]*JobInfo
	mutex     sync.RWMutex
}

// JobInfo 任务信息
type JobInfo struct {
	Config     types.JobConfig
	EntryID    cron.EntryID
	Status     JobStatus
	LastRun    time.Time
	NextRun    time.Time
	RunCount   int64
	ErrorCount int64
	LastError  string
}

// JobStatus 任务状态
type JobStatus string

const (
	JobStatusPending JobStatus = "pending" // 等待中
	JobStatusRunning JobStatus = "running" // 运行中
	JobStatusStopped JobStatus = "stopped" // 已停止
	JobStatusFailed  JobStatus = "failed"  // 失败
)

// New 创建新的调度器
func New(logger *zap.Logger, exchanges map[string]types.ExchangeInterface, callback types.DataCallback) *Scheduler {
	return &Scheduler{
		cron:      cron.New(cron.WithSeconds()),
		logger:    logger,
		exchanges: exchanges,
		callback:  callback,
		jobs:      make(map[string]*JobInfo),
	}
}

// AddJob 添加任务
func (s *Scheduler) AddJob(jobConfig types.JobConfig) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 检查交易所是否存在
	exchange, exists := s.exchanges[jobConfig.Exchange]
	if !exists {
		return fmt.Errorf("exchange %s not found", jobConfig.Exchange)
	}

	// 创建任务处理函数
	jobFunc := s.createJobFunc(jobConfig, exchange)

	// 添加到cron
	entryID, err := s.cron.AddFunc(jobConfig.Cron, jobFunc)
	if err != nil {
		return fmt.Errorf("failed to add cron job: %v", err)
	}

	// 保存任务信息
	s.jobs[jobConfig.Name] = &JobInfo{
		Config:     jobConfig,
		EntryID:    entryID,
		Status:     JobStatusPending,
		RunCount:   0,
		ErrorCount: 0,
	}

	s.logger.Info("任务已添加",
		zap.String("name", jobConfig.Name),
		zap.String("cron", jobConfig.Cron),
		zap.String("exchange", jobConfig.Exchange),
		zap.String("dataType", jobConfig.DataType))

	return nil
}

// createJobFunc 创建任务执行函数
func (s *Scheduler) createJobFunc(jobConfig types.JobConfig, exchange types.ExchangeInterface) func() {
	return func() {
		s.mutex.Lock()
		jobInfo := s.jobs[jobConfig.Name]
		jobInfo.Status = JobStatusRunning
		jobInfo.LastRun = time.Now()
		jobInfo.RunCount++
		s.mutex.Unlock()

		s.logger.Debug("开始执行任务",
			zap.String("job", jobConfig.Name),
			zap.String("dataType", jobConfig.DataType))

		// 执行任务
		err := s.executeJob(jobConfig, exchange)

		s.mutex.Lock()
		if err != nil {
			jobInfo.Status = JobStatusFailed
			jobInfo.ErrorCount++
			jobInfo.LastError = err.Error()
			s.logger.Error("任务执行失败",
				zap.String("job", jobConfig.Name),
				zap.Error(err))
		} else {
			jobInfo.Status = JobStatusPending
			jobInfo.LastError = ""
			s.logger.Debug("任务执行成功",
				zap.String("job", jobConfig.Name))
		}
		s.mutex.Unlock()
	}
}

// executeJob 执行具体的任务
func (s *Scheduler) executeJob(jobConfig types.JobConfig, exchange types.ExchangeInterface) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 根据数据类型执行不同的操作
	switch types.DataType(jobConfig.DataType) {
	case types.DataTypeTicker:
		return s.executeTicker(ctx, jobConfig, exchange)
	case types.DataTypeOrderbook:
		return s.executeOrderbook(ctx, jobConfig, exchange)
	case types.DataTypeTrades:
		return s.executeTrades(ctx, jobConfig, exchange)
	case types.DataTypeKlines:
		return s.executeKlines(ctx, jobConfig, exchange)
	default:
		return fmt.Errorf("unsupported data type: %s", jobConfig.DataType)
	}
}

// executeTicker 执行ticker数据获取任务
func (s *Scheduler) executeTicker(ctx context.Context, jobConfig types.JobConfig, exchange types.ExchangeInterface) error {
	// 获取配置中的symbols
	symbols := s.getSymbolsForExchange(jobConfig.Exchange, types.DataTypeTicker)
	if len(symbols) == 0 {
		return fmt.Errorf("no symbols configured for ticker data")
	}

	// 批量获取ticker数据
	tickers, err := exchange.GetMultipleTickers(ctx, symbols)
	if err != nil {
		return fmt.Errorf("failed to get tickers: %v", err)
	}

	// 调用回调函数处理数据
	for _, ticker := range tickers {
		if err := s.callback(&ticker); err != nil {
			s.logger.Error("处理ticker数据失败",
				zap.String("symbol", string(ticker.Symbol)),
				zap.Error(err))
		}
	}

	return nil
}

// executeOrderbook 执行orderbook数据获取任务
func (s *Scheduler) executeOrderbook(ctx context.Context, jobConfig types.JobConfig, exchange types.ExchangeInterface) error {
	symbols := s.getSymbolsForExchange(jobConfig.Exchange, types.DataTypeOrderbook)
	if len(symbols) == 0 {
		return fmt.Errorf("no symbols configured for orderbook data")
	}

	depth := s.getDepthForExchange(jobConfig.Exchange)

	// 批量获取orderbook数据
	orderbooks, err := exchange.GetMultipleOrderbooks(ctx, symbols, depth)
	if err != nil {
		return fmt.Errorf("failed to get orderbooks: %v", err)
	}

	// 调用回调函数处理数据
	for _, orderbook := range orderbooks {
		if err := s.callback(&orderbook); err != nil {
			s.logger.Error("处理orderbook数据失败",
				zap.String("symbol", string(orderbook.Symbol)),
				zap.Error(err))
		}
	}

	return nil
}

// executeTrades 执行trades数据获取任务
func (s *Scheduler) executeTrades(ctx context.Context, jobConfig types.JobConfig, exchange types.ExchangeInterface) error {
	symbols := s.getSymbolsForExchange(jobConfig.Exchange, types.DataTypeTrades)
	if len(symbols) == 0 {
		return fmt.Errorf("no symbols configured for trades data")
	}

	// 为每个symbol获取trades数据
	for _, symbol := range symbols {
		trades, err := exchange.GetTrades(ctx, symbol, 100) // 默认获取100条
		if err != nil {
			s.logger.Error("获取trades数据失败",
				zap.String("symbol", string(symbol)),
				zap.Error(err))
			continue
		}

		// 调用回调函数处理数据
		for _, trade := range trades {
			if err := s.callback(&trade); err != nil {
				s.logger.Error("处理trade数据失败",
					zap.String("symbol", string(trade.Symbol)),
					zap.Error(err))
			}
		}
	}

	return nil
}

// executeKlines 执行klines数据获取任务
func (s *Scheduler) executeKlines(ctx context.Context, jobConfig types.JobConfig, exchange types.ExchangeInterface) error {
	symbols := s.getSymbolsForExchange(jobConfig.Exchange, types.DataTypeKlines)
	intervals := s.getIntervalsForExchange(jobConfig.Exchange)

	if len(symbols) == 0 {
		return fmt.Errorf("no symbols configured for klines data")
	}
	if len(intervals) == 0 {
		intervals = []string{"1m"} // 默认1分钟
	}

	// 为每个symbol和interval组合获取klines数据
	for _, symbol := range symbols {
		for _, interval := range intervals {
			klines, err := exchange.GetKlines(ctx, symbol, interval, 100) // 默认获取100条
			if err != nil {
				s.logger.Error("获取klines数据失败",
					zap.String("symbol", string(symbol)),
					zap.String("interval", interval),
					zap.Error(err))
				continue
			}

			// 调用回调函数处理数据
			for _, kline := range klines {
				if err := s.callback(&kline); err != nil {
					s.logger.Error("处理kline数据失败",
						zap.String("symbol", string(kline.Symbol)),
						zap.String("interval", kline.Interval),
						zap.Error(err))
				}
			}
		}
	}

	return nil
}

// Start 启动调度器
func (s *Scheduler) Start() error {
	s.cron.Start()
	s.logger.Info("调度器已启动")
	return nil
}

// Stop 停止调度器
func (s *Scheduler) Stop(ctx context.Context) error {
	stopCtx := s.cron.Stop()

	select {
	case <-stopCtx.Done():
		s.logger.Info("调度器已停止")
		return nil
	case <-ctx.Done():
		s.logger.Warn("调度器停止超时")
		return ctx.Err()
	}
}

// GetJobStatus 获取任务状态
func (s *Scheduler) GetJobStatus() map[string]*JobInfo {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	result := make(map[string]*JobInfo)
	for name, job := range s.jobs {
		// 更新下次运行时间
		entry := s.cron.Entry(job.EntryID)
		job.NextRun = entry.Next

		result[name] = &JobInfo{
			Config:     job.Config,
			EntryID:    job.EntryID,
			Status:     job.Status,
			LastRun:    job.LastRun,
			NextRun:    job.NextRun,
			RunCount:   job.RunCount,
			ErrorCount: job.ErrorCount,
			LastError:  job.LastError,
		}
	}

	return result
}

// getSymbolsForExchange 从配置中获取交易对列表（这里是简化实现，实际应该从配置文件读取）
func (s *Scheduler) getSymbolsForExchange(exchangeName string, dataType types.DataType) []types.Symbol {
	// TODO: 从配置中读取对应交易所和数据类型的symbols
	// 这里返回一些默认值作为示例
	switch exchangeName {
	case "binance":
		return []types.Symbol{"BTCUSDT", "ETHUSDT", "BNBUSDT"}
	default:
		return []types.Symbol{}
	}
}

// getDepthForExchange 获取订单簿深度
func (s *Scheduler) getDepthForExchange(exchangeName string) int {
	// TODO: 从配置中读取
	return 20
}

// getIntervalsForExchange 获取K线时间间隔
func (s *Scheduler) getIntervalsForExchange(exchangeName string) []string {
	// TODO: 从配置中读取
	return []string{"1m", "5m", "1h"}
}
