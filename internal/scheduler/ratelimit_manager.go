// Package scheduler 提供智能频控管理功能
package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/mooyang-code/data-miner/internal/types"
)

// RateLimitManager 频控管理器
type RateLimitManager struct {
	logger *zap.Logger
	mu     sync.RWMutex

	// 权重配置
	maxWeightPerMinute int     // 每分钟最大权重
	safetyThreshold    float64 // 安全阈值（0.9表示90%）
	batchSize          int     // 每批处理的交易对数量

	// 状态跟踪
	lastWeightCheck time.Time
	currentWeight   int
	serverTime      int64
}

// NewRateLimitManager 创建新的频控管理器
func NewRateLimitManager(logger *zap.Logger) *RateLimitManager {
	return &RateLimitManager{
		logger:             logger,
		maxWeightPerMinute: 1200,  // Binance默认限制
		safetyThreshold:    0.9,   // 90%安全阈值
		batchSize:          80,    // 每批80个交易对
		lastWeightCheck:    time.Now(),
		currentWeight:      0,
	}
}

// CheckAndWaitIfNeeded 检查权重使用情况，如果需要则等待
func (r *RateLimitManager) CheckAndWaitIfNeeded(ctx context.Context, exchange types.ExchangeInterface) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 尝试获取权重信息
	if binanceExchange, ok := exchange.(interface {
		GetTimeAndWeight(ctx context.Context) (int64, int, error)
	}); ok {
		serverTime, weight, err := binanceExchange.GetTimeAndWeight(ctx)
		if err != nil {
			r.logger.Warn("获取权重信息失败，使用本地估算", zap.Error(err))
		} else {
			r.currentWeight = weight
			r.serverTime = serverTime
			r.lastWeightCheck = time.Now()
			
			r.logger.Debug("权重检查",
				zap.Int("current_weight", weight),
				zap.Int("max_weight", r.maxWeightPerMinute),
				zap.Float64("usage_percent", float64(weight)/float64(r.maxWeightPerMinute)*100))
		}
	}

	// 检查是否超过安全阈值
	if float64(r.currentWeight) > float64(r.maxWeightPerMinute)*r.safetyThreshold {
		// 计算需要等待的时间
		waitTime := r.calculateWaitTime()

		// 限制最大等待时间，避免长时间阻塞
		maxWaitTime := 90 * time.Second
		if waitTime > maxWaitTime {
			waitTime = maxWaitTime
		}

		r.logger.Info("权重使用接近限制，等待下一分钟",
			zap.Int("current_weight", r.currentWeight),
			zap.Int("max_weight", r.maxWeightPerMinute),
			zap.Duration("wait_time", waitTime),
			zap.Duration("max_wait_time", maxWaitTime))

		// 等待到下一分钟
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// 重置权重计数
			r.currentWeight = 0
			r.lastWeightCheck = time.Now()
		}
	}

	return nil
}

// calculateWaitTime 计算需要等待的时间
func (r *RateLimitManager) calculateWaitTime() time.Duration {
	now := time.Now()
	
	// 计算到下一分钟的时间
	nextMinute := now.Truncate(time.Minute).Add(time.Minute)
	waitTime := nextMinute.Sub(now)
	
	// 添加一些缓冲时间
	waitTime += 2 * time.Second
	
	return waitTime
}

// GetBatchSize 获取批处理大小
func (r *RateLimitManager) GetBatchSize() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.batchSize
}

// EstimateWeight 估算操作权重
func (r *RateLimitManager) EstimateWeight(operation string, count int) int {
	switch operation {
	case "klines":
		return count * 2 // 每个K线请求权重为2
	case "ticker":
		if count <= 20 {
			return count * 1 // 单个ticker权重为1
		} else if count <= 100 {
			return 40 // 批量ticker权重为40
		} else {
			return 80 // 全部ticker权重为80
		}
	case "orderbook":
		return count * 10 // 每个orderbook权重为10
	case "trades":
		return count * 1 // 每个trades权重为1
	default:
		return count * 1 // 默认权重
	}
}

// ProcessInBatches 分批处理交易对
func (r *RateLimitManager) ProcessInBatches(ctx context.Context, symbols []types.Symbol, 
	exchange types.ExchangeInterface, processor func([]types.Symbol) error) error {
	
	totalSymbols := len(symbols)
	if totalSymbols == 0 {
		return nil
	}

	batchSize := r.GetBatchSize()
	r.logger.Info("开始分批处理",
		zap.Int("total_symbols", totalSymbols),
		zap.Int("batch_size", batchSize),
		zap.Int("estimated_batches", (totalSymbols+batchSize-1)/batchSize))

	for i := 0; i < totalSymbols; i += batchSize {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 计算当前批次的范围
		end := i + batchSize
		if end > totalSymbols {
			end = totalSymbols
		}

		batch := symbols[i:end]
		batchNum := (i / batchSize) + 1
		totalBatches := (totalSymbols + batchSize - 1) / batchSize

		// 检查并等待权重限制
		if err := r.CheckAndWaitIfNeeded(ctx, exchange); err != nil {
			r.logger.Error("权重检查失败",
				zap.Int("batch_num", batchNum),
				zap.Error(err))
			return err
		}

		r.logger.Debug("处理批次",
			zap.Int("batch_num", batchNum),
			zap.Int("total_batches", totalBatches),
			zap.Int("batch_size", len(batch)),
			zap.String("first_symbol", string(batch[0])),
			zap.String("last_symbol", string(batch[len(batch)-1])))

		// 处理当前批次，记录处理时间
		batchStartTime := time.Now()
		if err := processor(batch); err != nil {
			batchDuration := time.Since(batchStartTime)
			r.logger.Error("批次处理失败",
				zap.Int("batch_num", batchNum),
				zap.Duration("batch_duration", batchDuration),
				zap.Error(err))
			return fmt.Errorf("batch %d processing failed: %w", batchNum, err)
		}
		batchDuration := time.Since(batchStartTime)

		// 更新权重估算
		estimatedWeight := r.EstimateWeight("klines", len(batch))
		r.mu.Lock()
		r.currentWeight += estimatedWeight
		r.mu.Unlock()

		r.logger.Debug("批次处理完成",
			zap.Int("batch_num", batchNum),
			zap.Duration("batch_duration", batchDuration),
			zap.Int("estimated_weight_used", estimatedWeight),
			zap.Int("total_estimated_weight", r.currentWeight))

		// 如果不是最后一批，添加小延迟避免过于频繁的请求
		if end < totalSymbols {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
			}
		}
	}

	r.logger.Info("分批处理完成",
		zap.Int("total_symbols", totalSymbols),
		zap.Int("final_estimated_weight", r.currentWeight))

	return nil
}

// GetStatus 获取频控管理器状态
func (r *RateLimitManager) GetStatus() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return map[string]interface{}{
		"max_weight_per_minute": r.maxWeightPerMinute,
		"current_weight":        r.currentWeight,
		"safety_threshold":      r.safetyThreshold,
		"batch_size":           r.batchSize,
		"last_weight_check":    r.lastWeightCheck,
		"server_time":          r.serverTime,
		"usage_percent":        float64(r.currentWeight) / float64(r.maxWeightPerMinute) * 100,
	}
}
