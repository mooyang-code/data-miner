// Package binance 提供Binance交易所的交易对缓存管理功能
package binance

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/avast/retry-go/v4"
	"go.uber.org/zap"

	"github.com/mooyang-code/data-miner/internal/exchanges/asset"
	"github.com/mooyang-code/data-miner/pkg/cryptotrader/currency"
)

// TradablePairsCache 交易对缓存管理器
type TradablePairsCache struct {
	binance    *Binance                      // Binance交易所实例
	logger     *zap.Logger                   // 日志记录器
	cache      map[asset.Item]currency.Pairs // 缓存数据，按资产类型分组
	lastUpdate map[asset.Item]time.Time      // 最后更新时间
	mutex      sync.RWMutex                  // 读写锁
	config     TradablePairsCacheConfig      // 缓存配置
	stopChan   chan struct{}                 // 停止信号
	running    bool                          // 是否正在运行
}

// TradablePairsCacheConfig 缓存配置
type TradablePairsCacheConfig struct {
	UpdateInterval  time.Duration // 更新间隔
	CacheTTL        time.Duration // 缓存生存时间
	SupportedAssets []asset.Item  // 支持的资产类型
	AutoUpdate      bool          // 是否自动更新
}

// NewTradablePairsCache 创建新的交易对缓存管理器
func NewTradablePairsCache(binance *Binance, logger *zap.Logger, config TradablePairsCacheConfig) *TradablePairsCache {
	return &TradablePairsCache{
		binance:    binance,
		logger:     logger,
		cache:      make(map[asset.Item]currency.Pairs),
		lastUpdate: make(map[asset.Item]time.Time),
		config:     config,
		stopChan:   make(chan struct{}),
		running:    false,
	}
}

// Start 启动缓存管理器
func (tpc *TradablePairsCache) Start(ctx context.Context) error {
	tpc.mutex.Lock()
	if tpc.running {
		tpc.mutex.Unlock()
		return fmt.Errorf("tradable pairs cache is already running")
	}
	tpc.mutex.Unlock()

	// 初始化缓存数据
	tpc.logger.Info("开始初始化缓存数据...")
	if err := tpc.refreshAllAssets(ctx); err != nil {
		return fmt.Errorf("failed to initialize cache: %w", err)
	}
	tpc.logger.Info("缓存数据初始化完成")

	// 启动自动更新
	if tpc.config.AutoUpdate {
		tpc.logger.Info("启动自动更新循环...")
		go tpc.autoUpdateLoop(ctx)
	}

	tpc.mutex.Lock()
	tpc.running = true
	tpc.mutex.Unlock()

	tpc.logger.Info("Tradable pairs cache started",
		zap.Duration("update_interval", tpc.config.UpdateInterval),
		zap.Duration("cache_ttl", tpc.config.CacheTTL),
		zap.Bool("auto_update", tpc.config.AutoUpdate))
	return nil
}

// Stop 停止缓存管理器
func (tpc *TradablePairsCache) Stop() {
	tpc.mutex.Lock()
	defer tpc.mutex.Unlock()

	if !tpc.running {
		return
	}

	close(tpc.stopChan)
	tpc.running = false
	tpc.logger.Info("Tradable pairs cache stopped")
}

// GetTradablePairs 获取指定资产类型的交易对
func (tpc *TradablePairsCache) GetTradablePairs(ctx context.Context, assetType asset.Item) (currency.Pairs, error) {
	tpc.mutex.RLock()

	// 检查缓存是否存在且未过期
	pairs, exists := tpc.cache[assetType]
	lastUpdate, hasUpdate := tpc.lastUpdate[assetType]

	if exists && hasUpdate && time.Since(lastUpdate) < tpc.config.CacheTTL {
		tpc.mutex.RUnlock()
		tpc.logger.Debug("Returning cached tradable pairs",
			zap.String("asset", assetType.String()),
			zap.Int("count", len(pairs)),
			zap.Time("last_update", lastUpdate))
		return pairs, nil
	}

	tpc.mutex.RUnlock()

	// 缓存过期或不存在，需要刷新
	tpc.logger.Info("Cache expired or missing, refreshing tradable pairs",
		zap.String("asset", assetType.String()))
	return tpc.refreshAsset(ctx, assetType)
}

// RefreshAsset 刷新指定资产类型的交易对，使用 retry 库进行重试
func (tpc *TradablePairsCache) refreshAsset(ctx context.Context, assetType asset.Item) (currency.Pairs, error) {
	var pairs currency.Pairs
	var lastErr error

	// 使用 retry 库进行重试
	err := retry.Do(
		func() error {
			// 从API获取最新数据
			fetchedPairs, err := tpc.binance.FetchTradablePairs(ctx, assetType)
			if err != nil {
				lastErr = err
				tpc.logger.Warn("获取交易对失败，准备重试",
					zap.String("asset", assetType.String()),
					zap.Error(err))
				return err
			}
			pairs = fetchedPairs
			return nil
		},
		retry.Attempts(3),
		retry.Delay(2*time.Second),
		retry.DelayType(retry.BackOffDelay),
		retry.MaxDelay(10*time.Second),
		retry.Context(ctx),
		retry.OnRetry(func(n uint, err error) {
			tpc.logger.Warn("重试获取交易对",
				zap.String("asset", assetType.String()),
				zap.Uint("attempt", n+1),
				zap.Error(err))
		}),
		retry.RetryIf(func(err error) bool {
			// 对网络错误、超时错误等进行重试
			return true
		}),
	)

	if err != nil {
		return nil, fmt.Errorf("moox backend service获取 %s 交易对失败，已重试3次: %w", assetType, lastErr)
	}

	// 更新缓存
	tpc.mutex.Lock()
	tpc.cache[assetType] = pairs
	tpc.lastUpdate[assetType] = time.Now()
	tpc.mutex.Unlock()

	tpc.logger.Info("交易对缓存刷新成功",
		zap.String("asset", assetType.String()),
		zap.Int("count", len(pairs)))
	return pairs, nil
}

// refreshAllAssets 刷新所有支持的资产类型
func (tpc *TradablePairsCache) refreshAllAssets(ctx context.Context) error {
	tpc.logger.Info("开始刷新所有资产类型", zap.Int("asset_count", len(tpc.config.SupportedAssets)))

	var errors []error
	successCount := 0

	for i, assetType := range tpc.config.SupportedAssets {
		tpc.logger.Info("正在刷新资产", zap.Int("index", i), zap.String("asset", assetType.String()))

		// 为每个资产类型创建带超时的上下文
		assetCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		_, err := tpc.refreshAsset(assetCtx, assetType)
		cancel()

		if err != nil {
			tpc.logger.Error("Failed to refresh asset",
				zap.String("asset", assetType.String()),
				zap.Error(err))
			errors = append(errors, fmt.Errorf("asset %s: %w", assetType.String(), err))
		} else {
			tpc.logger.Info("资产刷新完成", zap.String("asset", assetType.String()))
			successCount++
		}
	}

	// 如果至少有一个资产类型成功，则认为初始化成功
	if successCount > 0 {
		tpc.logger.Info("资产类型刷新完成",
			zap.Int("success_count", successCount),
			zap.Int("total_count", len(tpc.config.SupportedAssets)),
			zap.Int("error_count", len(errors)))
		return nil
	}

	// 如果所有资产类型都失败，返回错误
	tpc.logger.Error("所有资产类型刷新失败", zap.Int("error_count", len(errors)))
	if len(errors) > 0 {
		return fmt.Errorf("all asset types failed: %v", errors[0])
	}
	return fmt.Errorf("all asset types failed with unknown errors")
}

// autoUpdateLoop 自动更新循环
func (tpc *TradablePairsCache) autoUpdateLoop(ctx context.Context) {
	ticker := time.NewTicker(tpc.config.UpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			tpc.logger.Debug("Auto updating tradable pairs cache")
			if err := tpc.refreshAllAssets(ctx); err != nil {
				tpc.logger.Error("Failed to auto update tradable pairs cache", zap.Error(err))
			}
		case <-tpc.stopChan:
			tpc.logger.Debug("Auto update loop stopped")
			return
		case <-ctx.Done():
			tpc.logger.Debug("Auto update loop cancelled")
			return
		}
	}
}

// GetCacheStats 获取缓存统计信息
func (tpc *TradablePairsCache) GetCacheStats() map[string]interface{} {
	tpc.mutex.RLock()
	defer tpc.mutex.RUnlock()

	stats := make(map[string]interface{})
	stats["running"] = tpc.running
	stats["cache_ttl"] = tpc.config.CacheTTL.String()
	stats["update_interval"] = tpc.config.UpdateInterval.String()
	stats["auto_update"] = tpc.config.AutoUpdate

	assetStats := make(map[string]interface{})
	for assetType, pairs := range tpc.cache {
		lastUpdate, exists := tpc.lastUpdate[assetType]
		assetInfo := map[string]interface{}{
			"count":       len(pairs),
			"last_update": lastUpdate,
			"age":         time.Since(lastUpdate).String(),
			"expired":     !exists || time.Since(lastUpdate) >= tpc.config.CacheTTL,
		}
		assetStats[assetType.String()] = assetInfo
	}
	stats["assets"] = assetStats

	return stats
}

// IsSymbolSupported 检查指定交易对是否被支持
func (tpc *TradablePairsCache) IsSymbolSupported(ctx context.Context, symbol currency.Pair, assetType asset.Item) (bool, error) {
	pairs, err := tpc.GetTradablePairs(ctx, assetType)
	if err != nil {
		return false, err
	}

	for _, pair := range pairs {
		if pair.Equal(symbol) {
			return true, nil
		}
	}
	return false, nil
}

// GetSupportedSymbols 获取支持的交易对列表（字符串格式）
func (tpc *TradablePairsCache) GetSupportedSymbols(ctx context.Context, assetType asset.Item) ([]string, error) {
	pairs, err := tpc.GetTradablePairs(ctx, assetType)
	if err != nil {
		return nil, err
	}

	symbols := make([]string, len(pairs))
	for i, pair := range pairs {
		symbols[i] = pair.String()
	}
	return symbols, nil
}

// ForceRefresh 强制刷新指定资产类型的缓存
func (tpc *TradablePairsCache) ForceRefresh(ctx context.Context, assetType asset.Item) error {
	_, err := tpc.refreshAsset(ctx, assetType)
	return err
}

// ForceRefreshAll 强制刷新所有资产类型的缓存
func (tpc *TradablePairsCache) ForceRefreshAll(ctx context.Context) error {
	return tpc.refreshAllAssets(ctx)
}
