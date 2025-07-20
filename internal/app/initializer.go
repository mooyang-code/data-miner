// Package app 提供系统初始化功能
package app

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/avast/retry-go/v4"
	"go.uber.org/zap"

	"github.com/mooyang-code/data-miner/internal/exchanges/binance"
	"github.com/mooyang-code/data-miner/internal/types"
)

// SystemInitializer 系统初始化器
type SystemInitializer struct {
	logger *zap.Logger
	config *types.Config
}

// NewSystemInitializer 创建新的系统初始化器
func NewSystemInitializer(logger *zap.Logger, config *types.Config) *SystemInitializer {
	return &SystemInitializer{
		logger: logger,
		config: config,
	}
}

// InitializeExchanges 初始化所有交易所
func (si *SystemInitializer) InitializeExchanges(ctx context.Context) (map[string]types.ExchangeInterface, error) {
	exchanges := make(map[string]types.ExchangeInterface)

	// 初始化Binance交易所
	if si.config.Exchanges.Binance.Enabled {
		binanceExchange, err := si.initBinance(ctx)
		if err != nil {
			return nil, fmt.Errorf("moox backend service初始化Binance交易所失败: %w", err)
		}
		exchanges["binance"] = binanceExchange
		si.logger.Info("Binance交易所初始化成功")

		// 记录模式信息
		if si.config.Exchanges.Binance.UseWebsocket {
			si.logger.Info("Binance配置为WebSocket模式")
		} else {
			si.logger.Info("Binance配置为定时API拉取模式")
		}
	}

	return exchanges, nil
}

// initBinance 初始化Binance交易所
func (si *SystemInitializer) initBinance(ctx context.Context) (*binance.Binance, error) {
	b := binance.New()
	b.SetLogger(si.logger.Named("binance"))

	if err := b.Initialize(si.config.Exchanges.Binance); err != nil {
		return nil, fmt.Errorf("moox backend service配置Binance失败: %w", err)
	}

	// 启动交易对缓存（如果启用）
	if si.config.Exchanges.Binance.TradablePairs.FetchFromAPI {
		if err := si.startTradablePairsCache(ctx, b); err != nil {
			return nil, err
		}
	}
	return b, nil
}

// startTradablePairsCache 启动交易对缓存
func (si *SystemInitializer) startTradablePairsCache(ctx context.Context, b *binance.Binance) error {
	si.logger.Info("启动Binance交易对缓存...")

	// 检查网络连接
	if err := si.checkNetworkConnectivity(ctx); err != nil {
		si.logger.Warn("网络连接检查失败，将跳过交易对缓存初始化", zap.Error(err))
		if si.config.Exchanges.Binance.TradablePairs.SkipOnNetworkError {
			si.logger.Info("配置允许跳过网络错误，继续启动...")
			return nil
		}
		return fmt.Errorf("网络连接检查失败: %w", err)
	}

	// 使用带超时的上下文
	cacheCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := b.StartTradablePairsCache(cacheCtx); err != nil {
		si.logger.Error("启动交易对缓存失败", zap.Error(err))

		// 检查是否允许跳过缓存初始化失败
		if si.config.Exchanges.Binance.TradablePairs.SkipOnNetworkError {
			si.logger.Warn("配置允许跳过缓存初始化失败，继续启动...")
			return nil
		}
		return fmt.Errorf("启动交易对缓存失败: %w", err)
	}

	si.logger.Info("交易对缓存启动调用完成，等待初始化...")

	// 等待缓存初始化完成
	time.Sleep(2 * time.Second)

	stats := b.GetTradablePairsStats()
	si.logger.Info("交易对缓存启动成功", zap.Any("stats", stats))
	return nil
}

// InitializeSystem 初始化整个系统
func (si *SystemInitializer) InitializeSystem(ctx context.Context) (*SystemComponents, error) {
	si.logger.Info("开始系统初始化...")

	exchanges, err := si.InitializeExchanges(ctx)
	if err != nil {
		return nil, fmt.Errorf("moox backend service交易所初始化失败: %w", err)
	}

	si.logger.Info("交易所初始化完成，创建系统组件...")

	components := &SystemComponents{
		Exchanges: exchanges,
		Logger:    si.logger,
		Config:    si.config,
	}

	si.logger.Info("系统初始化完成", zap.Int("exchanges_count", len(exchanges)))
	return components, nil
}

// SystemComponents 系统组件
type SystemComponents struct {
	Exchanges map[string]types.ExchangeInterface
	Logger    *zap.Logger
	Config    *types.Config
}

// Shutdown 关闭系统组件
func (sc *SystemComponents) Shutdown() error {
	sc.Logger.Info("正在关闭系统组件...")

	for name, exchange := range sc.Exchanges {
		sc.Logger.Info("关闭交易所", zap.String("name", name))
		if err := exchange.Close(); err != nil {
			sc.Logger.Error("moox backend service关闭交易所失败",
				zap.String("name", name), zap.Error(err))
		}
	}

	sc.Logger.Info("系统关闭完成")
	return nil
}

// GetExchange 获取指定名称的交易所
func (sc *SystemComponents) GetExchange(name string) (types.ExchangeInterface, bool) {
	exchange, exists := sc.Exchanges[name]
	return exchange, exists
}

// GetBinanceExchange 获取Binance交易所实例
func (sc *SystemComponents) GetBinanceExchange() (*binance.Binance, error) {
	exchange, exists := sc.Exchanges["binance"]
	if !exists {
		return nil, fmt.Errorf("moox backend service未找到Binance交易所")
	}

	binanceExchange, ok := exchange.(*binance.Binance)
	if !ok {
		return nil, fmt.Errorf("moox backend service交易所类型错误")
	}

	return binanceExchange, nil
}

// ValidateConfiguration 验证配置
func (si *SystemInitializer) ValidateConfiguration() error {
	if si.config.Exchanges.Binance.Enabled {
		if err := si.validateBinanceConfig(); err != nil {
			return err
		}
	}
	return nil
}

// validateBinanceConfig 验证Binance配置
func (si *SystemInitializer) validateBinanceConfig() error {
	if si.config.Exchanges.Binance.APIURL == "" {
		return fmt.Errorf("moox backend service需要配置Binance API URL")
	}

	// 验证交易对缓存配置
	if si.config.Exchanges.Binance.TradablePairs.FetchFromAPI {
		si.validateTradablePairsConfig()
	}
	return nil
}

// validateTradablePairsConfig 验证交易对配置
func (si *SystemInitializer) validateTradablePairsConfig() {
	if si.config.Exchanges.Binance.TradablePairs.UpdateInterval == 0 {
		si.logger.Warn("交易对更新间隔未设置，使用默认值1小时")
	}
	if si.config.Exchanges.Binance.TradablePairs.CacheTTL == 0 {
		si.logger.Warn("交易对缓存TTL未设置，使用默认值2小时")
	}
	if len(si.config.Exchanges.Binance.TradablePairs.SupportedAssets) == 0 {
		si.logger.Warn("未配置支持的资产类型，使用默认值[spot]")
	}
}

// GetSystemStatus 获取系统状态
func (sc *SystemComponents) GetSystemStatus() map[string]interface{} {
	status := make(map[string]interface{})

	// 交易所状态
	exchangeStatus := make(map[string]interface{})
	for name, exchange := range sc.Exchanges {
		exchangeInfo := map[string]interface{}{
			"name":    exchange.GetName(),
			"enabled": true, // 如果在exchanges map中，说明已启用
		}

		// 如果是Binance交易所，获取额外信息
		if binanceExchange, ok := exchange.(*binance.Binance); ok {
			exchangeInfo["tradable_pairs_stats"] = binanceExchange.GetTradablePairsStats()
		}

		exchangeStatus[name] = exchangeInfo
	}
	status["exchanges"] = exchangeStatus

	// 系统信息
	status["system"] = map[string]interface{}{
		"initialized": true,
		"timestamp":   time.Now(),
	}

	return status
}

// checkNetworkConnectivity 使用 retry 库检查网络连接
func (si *SystemInitializer) checkNetworkConnectivity(ctx context.Context) error {
	si.logger.Info("检查网络连接...")

	// 使用 retry 库检查DNS解析
	err := retry.Do(
		func() error {
			return si.checkDNSResolution("api.binance.com")
		},
		retry.Attempts(3),
		retry.Delay(1*time.Second),
		retry.DelayType(retry.FixedDelay),
		retry.Context(ctx),
		retry.OnRetry(func(n uint, err error) {
			si.logger.Warn("DNS解析重试", zap.Uint("attempt", n+1), zap.Error(err))
		}),
	)
	if err != nil {
		return fmt.Errorf("moox backend serviceDNS解析失败，已重试3次: %w", err)
	}

	// 使用 retry 库检查HTTP连接
	err = retry.Do(
		func() error {
			return si.checkHTTPConnectivity(ctx, "https://api.binance.com")
		},
		retry.Attempts(3),
		retry.Delay(2*time.Second),
		retry.DelayType(retry.BackOffDelay),
		retry.MaxDelay(10*time.Second),
		retry.Context(ctx),
		retry.OnRetry(func(n uint, err error) {
			si.logger.Warn("HTTP连接重试", zap.Uint("attempt", n+1), zap.Error(err))
		}),
	)
	if err != nil {
		return fmt.Errorf("moox backend serviceHTTP连接失败，已重试3次: %w", err)
	}

	si.logger.Info("网络连接检查通过")
	return nil
}

// checkDNSResolution 检查DNS解析
func (si *SystemInitializer) checkDNSResolution(hostname string) error {
	si.logger.Debug("检查DNS解析", zap.String("hostname", hostname))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := net.DefaultResolver.LookupHost(ctx, hostname)
	if err != nil {
		return fmt.Errorf("无法解析主机名 %s: %w", hostname, err)
	}

	si.logger.Debug("DNS解析成功", zap.String("hostname", hostname))
	return nil
}

// checkHTTPConnectivity 检查HTTP连接
func (si *SystemInitializer) checkHTTPConnectivity(ctx context.Context, url string) error {
	si.logger.Debug("检查HTTP连接", zap.String("url", url))

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout: 5 * time.Second,
		},
	}

	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP响应错误: %d %s", resp.StatusCode, resp.Status)
	}

	si.logger.Debug("HTTP连接成功", zap.String("url", url), zap.Int("status", resp.StatusCode))
	return nil
}
