// Package scheduler 提供任务调度功能
package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/mooyang-code/data-miner/internal/exchanges/asset"
	"github.com/mooyang-code/data-miner/internal/types"
	"github.com/mooyang-code/data-miner/pkg/cryptotrader/currency"
)

// Scheduler 调度器
type Scheduler struct {
	cron            *cron.Cron
	logger          *zap.Logger
	exchanges       map[string]types.ExchangeInterface
	callback        types.DataCallback
	jobs            map[string]*JobInfo
	mutex           sync.RWMutex
	config          *types.Config // 添加配置字段
	rateLimitMgr    *RateLimitManager // 频控管理器
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
func New(logger *zap.Logger, exchanges map[string]types.ExchangeInterface, callback types.DataCallback, config *types.Config) *Scheduler {
	return &Scheduler{
		cron:         cron.New(cron.WithSeconds()),
		logger:       logger,
		exchanges:    exchanges,
		callback:     callback,
		jobs:         make(map[string]*JobInfo),
		config:       config,
		rateLimitMgr: NewRateLimitManager(logger),
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
	// 根据数据类型设置不同的超时时间
	timeout := s.getTimeoutForDataType(jobConfig.DataType)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
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

// executeKlines 执行klines数据获取任务（智能频控版本）
func (s *Scheduler) executeKlines(ctx context.Context, jobConfig types.JobConfig, exchange types.ExchangeInterface) error {
	s.logger.Info("执行klines数据获取任务（智能频控）")
	symbols := s.getSymbolsForExchange(jobConfig.Exchange, types.DataTypeKlines)
	intervals := s.getIntervalsForExchange(jobConfig.Exchange)

	if len(symbols) == 0 {
		return fmt.Errorf("no symbols configured for klines data")
	}
	if len(intervals) == 0 {
		intervals = []string{"1m"} // 默认1分钟
	}

	s.logger.Info("开始智能批量获取K线数据",
		zap.Int("total_symbols", len(symbols)),
		zap.Strings("intervals", intervals))

	// 为每个interval分别处理
	for _, interval := range intervals {
		s.logger.Info("处理K线间隔", zap.String("interval", interval))

		// 使用频控管理器分批处理
		err := s.rateLimitMgr.ProcessInBatches(ctx, symbols, exchange, func(batch []types.Symbol) error {
			return s.processBatchKlines(ctx, batch, interval, exchange)
		})

		if err != nil {
			s.logger.Error("批量处理K线数据失败",
				zap.String("interval", interval),
				zap.Error(err))
			return err
		}
	}

	s.logger.Info("智能批量获取K线数据完成")
	return nil
}

// processBatchKlines 处理一批K线数据
func (s *Scheduler) processBatchKlines(ctx context.Context, symbols []types.Symbol, interval string, exchange types.ExchangeInterface) error {
	successCount := 0
	errorCount := 0

	for i, symbol := range symbols {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			s.logger.Warn("批次处理被取消",
				zap.String("interval", interval),
				zap.Int("processed", i),
				zap.Int("total", len(symbols)),
				zap.Int("success", successCount),
				zap.Int("errors", errorCount))
			return ctx.Err()
		default:
		}

		// 为单个API调用设置较短的超时时间
		apiCtx, apiCancel := context.WithTimeout(ctx, 30*time.Second)
		klines, err := exchange.GetKlines(apiCtx, symbol, interval, 100) // 默认获取100条
		apiCancel()

		if err != nil {
			errorCount++
			s.logger.Error("获取klines数据失败",
				zap.String("symbol", string(symbol)),
				zap.String("interval", interval),
				zap.Int("symbol_index", i+1),
				zap.Int("total_symbols", len(symbols)),
				zap.Error(err))
			continue
		}

		successCount++
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

	s.logger.Debug("批次K线处理完成",
		zap.String("interval", interval),
		zap.Int("total", len(symbols)),
		zap.Int("success", successCount),
		zap.Int("errors", errorCount))

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

// GetRateLimitStatus 获取频控状态
func (s *Scheduler) GetRateLimitStatus() map[string]interface{} {
	if s.rateLimitMgr == nil {
		return map[string]interface{}{
			"error": "rate limit manager not initialized",
		}
	}
	return s.rateLimitMgr.GetStatus()
}

// getSymbolsForExchange 从配置中获取交易对列表
func (s *Scheduler) getSymbolsForExchange(exchangeName string, dataType types.DataType) []types.Symbol {
	if s.config == nil {
		s.logger.Warn("配置为空，使用默认交易对")
		return []types.Symbol{"BTCUSDT", "ETHUSDT", "BNBUSDT"}
	}

	switch exchangeName {
	case "binance":
		return s.getBinanceSymbols(dataType)
	default:
		s.logger.Warn("不支持的交易所", zap.String("exchange", exchangeName))
		return []types.Symbol{}
	}
}

// getBinanceSymbols 获取Binance交易对列表
func (s *Scheduler) getBinanceSymbols(dataType types.DataType) []types.Symbol {
	binanceConfig := s.config.Exchanges.Binance

	var configSymbols []string
	switch dataType {
	case types.DataTypeTicker:
		configSymbols = binanceConfig.DataTypes.Ticker.Symbols
	case types.DataTypeOrderbook:
		configSymbols = binanceConfig.DataTypes.Orderbook.Symbols
	case types.DataTypeTrades:
		configSymbols = binanceConfig.DataTypes.Trades.Symbols
	case types.DataTypeKlines:
		configSymbols = binanceConfig.DataTypes.Klines.Symbols
	default:
		s.logger.Warn("不支持的数据类型", zap.String("dataType", string(dataType)))
		return []types.Symbol{}
	}

	// 如果配置中包含"*"，则从cache中获取所有可用交易对
	if len(configSymbols) == 1 && configSymbols[0] == "*" {
		s.logger.Debug("从cache获取所有交易对",
			zap.String("dataType", string(dataType)))
		return s.getTradablePairsFromCache(dataType)
	}

	// 转换为Symbol类型
	symbols := make([]types.Symbol, 0, len(configSymbols))
	for _, symbol := range configSymbols {
		symbols = append(symbols, types.Symbol(symbol))
	}

	s.logger.Debug("从配置获取交易对",
		zap.String("dataType", string(dataType)),
		zap.Strings("symbols", configSymbols),
		zap.Int("count", len(symbols)),
		zap.Bool("fetch_from_api", s.config.Exchanges.Binance.TradablePairs.FetchFromAPI))

	return symbols
}

// getTradablePairsFromCache 从cache中获取可交易的交易对
func (s *Scheduler) getTradablePairsFromCache(dataType types.DataType) []types.Symbol {
	// 检查配置中的fetch_from_api开关
	if s.config == nil || !s.config.Exchanges.Binance.TradablePairs.FetchFromAPI {
		s.logger.Warn("fetch_from_api配置未启用，跳过从缓存获取交易对",
			zap.String("dataType", string(dataType)))
		return []types.Symbol{}
	}
	// 获取Binance交易所实例
	binanceExchange, exists := s.exchanges["binance"]
	if !exists {
		s.logger.Error("Binance交易所未找到")
		return []types.Symbol{}
	}

	// 尝试类型断言获取Binance实例
	binanceInterface, ok := binanceExchange.(interface {
		GetTradablePairsFromCache(ctx context.Context, assetType asset.Item) (currency.Pairs, error)
	})
	if !ok {
		s.logger.Error("Binance交易所不支持从cache获取交易对")
		return []types.Symbol{}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 从cache获取现货交易对
	pairs, err := binanceInterface.GetTradablePairsFromCache(ctx, asset.Spot)
	if err != nil {
		s.logger.Error("从cache获取交易对失败", zap.Error(err))
		return []types.Symbol{}
	}

	// 转换为Symbol类型
	symbols := make([]types.Symbol, 0, len(pairs))
	for _, pair := range pairs {
		symbols = append(symbols, types.Symbol(pair.String()))
	}

	s.logger.Info("从cache获取交易对成功",
		zap.String("dataType", string(dataType)),
		zap.Int("count", len(symbols)),
		zap.Bool("fetch_from_api", s.config.Exchanges.Binance.TradablePairs.FetchFromAPI))

	return symbols
}

// getDepthForExchange 获取订单簿深度
func (s *Scheduler) getDepthForExchange(exchangeName string) int {
	if s.config == nil {
		return 20 // 默认深度
	}

	switch exchangeName {
	case "binance":
		return s.config.Exchanges.Binance.DataTypes.Orderbook.Depth
	default:
		return 20 // 默认深度
	}
}

// getIntervalsForExchange 获取K线时间间隔
func (s *Scheduler) getIntervalsForExchange(exchangeName string) []string {
	if s.config == nil {
		return []string{"1m", "5m", "1h"} // 默认间隔
	}

	switch exchangeName {
	case "binance":
		intervals := s.config.Exchanges.Binance.DataTypes.Klines.Intervals
		if len(intervals) == 0 {
			return []string{"1m"} // 默认1分钟
		}
		return intervals
	default:
		return []string{"1m", "5m", "1h"} // 默认间隔
	}
}

// getTimeoutForDataType 根据数据类型获取超时时间
func (s *Scheduler) getTimeoutForDataType(dataType string) time.Duration {
	switch types.DataType(dataType) {
	case types.DataTypeKlines:
		// K线数据需要更长时间，因为可能有多个间隔和大量交易对
		return 5 * time.Minute
	case types.DataTypeTicker:
		// Ticker数据相对简单
		return 2 * time.Minute
	case types.DataTypeOrderbook:
		// Orderbook数据中等复杂度
		return 3 * time.Minute
	case types.DataTypeTrades:
		// Trades数据中等复杂度
		return 3 * time.Minute
	default:
		// 默认超时时间
		return 2 * time.Minute
	}
}
