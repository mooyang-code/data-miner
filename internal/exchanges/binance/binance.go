// Package binance 实现Binance交易所公共接口和结构
package binance

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/mooyang-code/data-miner/internal/exchanges/asset"
	"github.com/mooyang-code/data-miner/internal/types"
	"github.com/mooyang-code/data-miner/pkg/cryptotrader/currency"
)

// Binance 主要的交易所结构体，包含REST API和WebSocket客户端
type Binance struct {
	RestAPI   *BinanceRestAPI     // REST API 客户端
	WebSocket *BinanceWebSocket   // WebSocket 客户端
	config    types.BinanceConfig // Binance公共配置

	rateLimit    *types.RateLimit // 速率限制
	requestCount int64            // 请求计数
	lastReset    time.Time        // 最后重置时间
	mu           sync.RWMutex     // 读写锁
	Name         string           // 交易所名称
	Enabled      bool             // 是否启用
	Verbose      bool             // 详细日志
	HTTPTimeout  time.Duration    // HTTP超时时间

	tradablePairsCache *TradablePairsCache // 交易对缓存管理器
	logger             *zap.Logger
}

// New 创建新的Binance交易所实例
func New() *Binance {
	b := &Binance{
		rateLimit: &types.RateLimit{
			RequestsPerMinute: 1200,
			LastRequest:       time.Now(),
		},
		lastReset:   time.Now(),
		Name:        "Binance",
		Enabled:     true,
		Verbose:     false,
		HTTPTimeout: 30 * time.Second,
	}

	// 初始化REST API客户端
	b.RestAPI = NewRestAPI()

	// 初始化WebSocket客户端
	b.WebSocket = NewWebSocket()

	// 初始化日志记录器（默认使用nop logger）
	b.logger = zap.NewNop()

	return b
}

// GetName 返回交易所名称
func (b *Binance) GetName() types.Exchange {
	return types.ExchangeBinance
}

// Initialize 初始化交易所
func (b *Binance) Initialize(config interface{}) error {
	binanceConfig, ok := config.(types.BinanceConfig)
	if !ok {
		b.config = types.BinanceConfig{} // 使用默认配置
	} else {
		b.config = binanceConfig
	}

	// 初始化交易对缓存管理器（如果配置启用）
	if b.config.TradablePairs.FetchFromAPI {
		if err := b.initializeTradablePairsCache(); err != nil {
			return fmt.Errorf("failed to initialize tradable pairs cache: %w", err)
		}
	}

	return nil
}

// SetLogger 设置日志记录器
func (b *Binance) SetLogger(logger *zap.Logger) {
	if logger != nil {
		b.logger = logger
	}
}

// initializeTradablePairsCache 初始化交易对缓存管理器
func (b *Binance) initializeTradablePairsCache() error {
	// 解析支持的资产类型
	supportedAssets := make([]asset.Item, 0, len(b.config.TradablePairs.SupportedAssets))
	for _, assetStr := range b.config.TradablePairs.SupportedAssets {
		switch assetStr {
		case "spot":
			supportedAssets = append(supportedAssets, asset.Spot)
		case "margin":
			supportedAssets = append(supportedAssets, asset.Margin)
		default:
			b.logger.Warn("Unsupported asset type in config", zap.String("asset", assetStr))
		}
	}

	// 如果没有配置支持的资产类型，默认支持现货
	if len(supportedAssets) == 0 {
		supportedAssets = []asset.Item{asset.Spot}
	}

	// 创建缓存配置
	cacheConfig := TradablePairsCacheConfig{
		UpdateInterval:  b.config.TradablePairs.UpdateInterval,
		CacheTTL:        b.config.TradablePairs.CacheTTL,
		SupportedAssets: supportedAssets,
		AutoUpdate:      b.config.TradablePairs.AutoUpdate,
	}

	// 设置默认值
	if cacheConfig.UpdateInterval == 0 {
		cacheConfig.UpdateInterval = 1 * time.Hour // 默认1小时更新一次
	}
	if cacheConfig.CacheTTL == 0 {
		cacheConfig.CacheTTL = 2 * time.Hour // 默认缓存2小时
	}

	// 创建缓存管理器
	b.tradablePairsCache = NewTradablePairsCache(b, b.logger, cacheConfig)
	return nil
}

// IsEnabled 返回交易所是否启用
func (b *Binance) IsEnabled() bool {
	return b.Enabled
}

// Close 关闭交易所连接
func (b *Binance) Close() error {
	// 停止交易对缓存管理器
	if b.tradablePairsCache != nil {
		b.tradablePairsCache.Stop()
	}

	// 关闭WebSocket连接
	if b.WebSocket != nil {
		if err := b.WebSocket.WsClose(); err != nil {
			return err
		}
	}

	// 关闭REST API客户端
	if b.RestAPI != nil {
		if err := b.RestAPI.Close(); err != nil {
			return err
		}
	}
	return nil
}

// CheckRateLimit 检查速率限制
func (b *Binance) CheckRateLimit() error {
	return b.RestAPI.CheckRateLimit()
}

// IsConnected 检查连接状态
func (b *Binance) IsConnected() bool {
	restConnected := b.RestAPI != nil && b.RestAPI.IsConnected()
	wsConnected := b.WebSocket != nil && b.WebSocket.IsConnected()
	return restConnected || wsConnected
}

// GetLastPing 获取最后ping时间
func (b *Binance) GetLastPing() time.Time {
	if b.WebSocket != nil {
		return b.WebSocket.GetLastPing()
	}
	return time.Time{}
}

// GetRateLimit 获取速率限制信息
func (b *Binance) GetRateLimit() *types.RateLimit {
	return b.rateLimit
}

// REST API 方法代理 - 将调用转发到RestAPI客户端

// GetTicker 获取单个交易对的行情数据
func (b *Binance) GetTicker(ctx context.Context, symbol types.Symbol) (*types.Ticker, error) {
	// 调用RestAPI获取Binance特定的数据
	binanceTicker, err := b.RestAPI.GetTickerBySymbol(ctx, string(symbol))
	if err != nil {
		return nil, err
	}

	// 转换为通用类型
	ticker := &types.Ticker{
		Exchange:  types.ExchangeBinance,
		Symbol:    symbol,
		Price:     binanceTicker.LastPrice.Float64(),
		Volume:    binanceTicker.Volume.Float64(),
		High24h:   binanceTicker.HighPrice.Float64(),
		Low24h:    binanceTicker.LowPrice.Float64(),
		Change24h: binanceTicker.PriceChangePercent.Float64(),
		Timestamp: time.Now(),
	}
	return ticker, nil
}

// GetOrderbook 获取订单簿数据
func (b *Binance) GetOrderbook(ctx context.Context, symbol types.Symbol, depth int) (*types.Orderbook, error) {
	// 转换symbol为currency.Pair
	pair, err := currency.NewPairFromString(string(symbol))
	if err != nil {
		return nil, err
	}

	// 调用RestAPI获取Binance特定的数据
	binanceOrderbook, err := b.RestAPI.GetOrderbook(ctx, pair, depth)
	if err != nil {
		return nil, err
	}

	// 转换为通用类型
	orderbook := &types.Orderbook{
		Exchange:  types.ExchangeBinance,
		Symbol:    symbol,
		Bids:      make([]types.OrderbookEntry, len(binanceOrderbook.Bids)),
		Asks:      make([]types.OrderbookEntry, len(binanceOrderbook.Asks)),
		Timestamp: time.Now(),
	}

	// 转换买单
	for i, bid := range binanceOrderbook.Bids {
		orderbook.Bids[i] = types.OrderbookEntry{
			Price:    bid.Price,
			Quantity: bid.Quantity,
		}
	}

	// 转换卖单
	for i, ask := range binanceOrderbook.Asks {
		orderbook.Asks[i] = types.OrderbookEntry{
			Price:    ask.Price,
			Quantity: ask.Quantity,
		}
	}
	return orderbook, nil
}

// GetTrades 获取交易数据
func (b *Binance) GetTrades(ctx context.Context, symbol types.Symbol, limit int) ([]types.Trade, error) {
	// 调用RestAPI获取Binance特定的数据
	binanceTrades, err := b.RestAPI.GetTradesBySymbol(ctx, string(symbol))
	if err != nil {
		return nil, err
	}

	// 转换为通用类型
	trades := make([]types.Trade, len(binanceTrades))
	for i, binanceTrade := range binanceTrades {
		trades[i] = types.Trade{
			Exchange:  types.ExchangeBinance,
			Symbol:    symbol,
			ID:        fmt.Sprintf("%d", binanceTrade.ID),
			Price:     binanceTrade.Price,
			Quantity:  binanceTrade.Quantity,
			Side:      getSideFromBuyer(binanceTrade.IsBuyerMaker),
			Timestamp: binanceTrade.Time.Time(),
		}
	}

	return trades, nil
}

// GetKlines 获取K线数据
func (b *Binance) GetKlines(ctx context.Context, symbol types.Symbol, interval string, limit int) ([]types.Kline, error) {
	// 直接调用RestAPI的GetKlinesForSymbol方法
	return b.RestAPI.GetKlinesForSymbol(ctx, symbol, interval, limit)
}

// GetTimeAndWeight 获取服务器时间和当前权重使用情况
func (b *Binance) GetTimeAndWeight(ctx context.Context) (int64, int, error) {
	return b.RestAPI.GetTimeAndWeight(ctx)
}

// GetMultipleTickers 批量获取行情数据
func (b *Binance) GetMultipleTickers(ctx context.Context, symbols []types.Symbol) ([]types.Ticker, error) {
	// 转换symbols为字符串数组
	symbolStrings := make([]string, len(symbols))
	for i, symbol := range symbols {
		symbolStrings[i] = string(symbol)
	}

	// 调用RestAPI获取Binance特定的数据
	binanceTickers, err := b.RestAPI.GetMultipleTickers(ctx, symbolStrings)
	if err != nil {
		return nil, err
	}

	// 转换为通用类型
	tickers := make([]types.Ticker, len(binanceTickers))
	for i, binanceTicker := range binanceTickers {
		tickers[i] = types.Ticker{
			Exchange:  types.ExchangeBinance,
			Symbol:    types.Symbol(binanceTicker.Symbol),
			Price:     binanceTicker.LastPrice.Float64(),
			Volume:    binanceTicker.Volume.Float64(),
			High24h:   binanceTicker.HighPrice.Float64(),
			Low24h:    binanceTicker.LowPrice.Float64(),
			Change24h: binanceTicker.PriceChangePercent.Float64(),
			Timestamp: time.Now(),
		}
	}

	return tickers, nil
}

// GetMultipleOrderbooks 批量获取订单簿数据
func (b *Binance) GetMultipleOrderbooks(ctx context.Context, symbols []types.Symbol, depth int) ([]types.Orderbook, error) {
	// 转换symbols为字符串数组
	symbolStrings := make([]string, len(symbols))
	for i, symbol := range symbols {
		symbolStrings[i] = string(symbol)
	}

	// 调用RestAPI获取Binance特定的数据
	binanceOrderbooks, err := b.RestAPI.GetMultipleOrderbooks(ctx, symbolStrings, depth)
	if err != nil {
		return nil, err
	}

	// 转换为通用类型
	orderbooks := make([]types.Orderbook, len(binanceOrderbooks))
	for i, binanceOrderbook := range binanceOrderbooks {
		orderbooks[i] = types.Orderbook{
			Exchange:  types.ExchangeBinance,
			Symbol:    types.Symbol(binanceOrderbook.Symbol),
			Bids:      make([]types.OrderbookEntry, len(binanceOrderbook.Bids)),
			Asks:      make([]types.OrderbookEntry, len(binanceOrderbook.Asks)),
			Timestamp: time.Now(),
		}

		// 转换买单
		for j, bid := range binanceOrderbook.Bids {
			orderbooks[i].Bids[j] = types.OrderbookEntry{
				Price:    bid.Price,
				Quantity: bid.Quantity,
			}
		}

		// 转换卖单
		for j, ask := range binanceOrderbook.Asks {
			orderbooks[i].Asks[j] = types.OrderbookEntry{
				Price:    ask.Price,
				Quantity: ask.Quantity,
			}
		}
	}
	return orderbooks, nil
}

// 辅助函数

// parseFloat64 安全地将字符串转换为float64
func parseFloat64(s string) float64 {
	if s == "" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

// getSideFromBuyer 根据isBuyerMaker判断交易方向
func getSideFromBuyer(isBuyerMaker bool) string {
	if isBuyerMaker {
		return "buy"
	}
	return "sell"
}

// WebSocket 方法代理 - 将调用转发到WebSocket客户端

// SubscribeTicker 订阅行情数据
func (b *Binance) SubscribeTicker(symbols []types.Symbol, callback types.DataCallback) error {
	return b.WebSocket.SubscribeTicker(symbols, callback)
}

// SubscribeOrderbook 订阅订单簿数据
func (b *Binance) SubscribeOrderbook(symbols []types.Symbol, callback types.DataCallback) error {
	return b.WebSocket.SubscribeOrderbook(symbols, callback)
}

// SubscribeTrades 订阅交易数据
func (b *Binance) SubscribeTrades(symbols []types.Symbol, callback types.DataCallback) error {
	return b.WebSocket.SubscribeTrades(symbols, callback)
}

// SubscribeKlines 订阅K线数据
func (b *Binance) SubscribeKlines(symbols []types.Symbol, intervals []string, callback types.DataCallback) error {
	return b.WebSocket.SubscribeKlines(symbols, intervals, callback)
}

// UnsubscribeAll 取消所有订阅
func (b *Binance) UnsubscribeAll() error {
	return b.WebSocket.UnsubscribeAll()
}

// WsConnect 连接WebSocket
func (b *Binance) WsConnect() error {
	return b.WebSocket.WsConnect()
}

// WsClose 关闭WebSocket连接
func (b *Binance) WsClose() error {
	return b.WebSocket.WsClose()
}

// IsWsConnected 返回WebSocket是否已连接
func (b *Binance) IsWsConnected() bool {
	return b.WebSocket.IsConnected()
}

// Subscribe 订阅WebSocket频道
func (b *Binance) Subscribe(channels []string) error {
	return b.WebSocket.Subscribe(channels)
}

// Unsubscribe 取消订阅WebSocket频道
func (b *Binance) Unsubscribe(channels []string) error {
	return b.WebSocket.Unsubscribe(channels)
}

// GetIPManagerStatus 获取IP管理器状态信息
func (b *Binance) GetIPManagerStatus() map[string]interface{} {
	status := make(map[string]interface{})

	// WebSocket IP管理器状态
	status["websocket"] = b.WebSocket.GetIPManagerStatus()

	// REST API IP管理器状态
	if b.RestAPI != nil {
		// 从RestAPI的状态中获取IP管理器信息
		restStatus := b.RestAPI.GetStatus()
		if httpClient, ok := restStatus["http_client"]; ok {
			if clientStatus, ok := httpClient.(map[string]interface{}); ok {
				if ipManager, ok := clientStatus["ip_manager"]; ok {
					status["restapi"] = ipManager
				} else {
					status["restapi"] = map[string]interface{}{
						"running": false,
						"error":   "IP manager not available",
					}
				}
			} else {
				status["restapi"] = map[string]interface{}{
					"running": false,
					"error":   "HTTP client status not available",
				}
			}
		} else {
			status["restapi"] = map[string]interface{}{
				"running": false,
				"error":   "HTTP client not available",
			}
		}
	} else {
		status["restapi"] = map[string]interface{}{
			"running": false,
			"error":   "REST API not initialized",
		}
	}
	return status
}

// SubscribeOrderbookWithDepth 订阅订单簿数据（自定义深度）
func (b *Binance) SubscribeOrderbookWithDepth(symbols []types.Symbol, depth int, updateSpeed string, callback types.DataCallback) error {
	return b.WebSocket.SubscribeOrderbookWithDepth(symbols, depth, updateSpeed, callback)
}

// GetActiveSubscriptions 获取当前活跃的订阅列表
func (b *Binance) GetActiveSubscriptions() []string {
	return b.WebSocket.GetActiveSubscriptions()
}

// GetSubscriptionCount 获取当前订阅数量
func (b *Binance) GetSubscriptionCount() int {
	return b.WebSocket.GetSubscriptionCount()
}

// FetchTradablePairs 获取交易所可交易的交易对列表
func (b *Binance) FetchTradablePairs(ctx context.Context, assetType asset.Item) (currency.Pairs, error) {
	b.logger.Info("Fetching tradable pairs", zap.String("asset", assetType.String()))
	if b.RestAPI == nil {
		return nil, fmt.Errorf("REST API not initialized")
	}

	// 获取交易所信息
	exchangeInfo, err := b.RestAPI.GetExchangeInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get exchange info: %w", err)
	}
	b.logger.Info("Exchange info fetched", zap.Int("symbols", len(exchangeInfo.Symbols)))

	tradingStatus := "TRADING"
	var pairs []currency.Pair

	switch assetType {
	case asset.Spot:
		pairs = make([]currency.Pair, 0, len(exchangeInfo.Symbols))
		for _, symbol := range exchangeInfo.Symbols {
			// 只返回状态为TRADING且允许现货交易的交易对
			if symbol.Status != tradingStatus || !symbol.IsSpotTradingAllowed {
				continue
			}

			pair, err := currency.NewPairFromStrings(symbol.BaseAsset, symbol.QuoteAsset)
			if err != nil {
				return nil, fmt.Errorf("failed to create pair from %s/%s: %w",
					symbol.BaseAsset, symbol.QuoteAsset, err)
			}
			pairs = append(pairs, pair)
		}
	case asset.Margin:
		pairs = make([]currency.Pair, 0, len(exchangeInfo.Symbols))
		for _, symbol := range exchangeInfo.Symbols {
			// 只返回状态为TRADING且允许保证金交易的交易对
			if symbol.Status != tradingStatus || !symbol.IsMarginTradingAllowed {
				continue
			}

			pair, err := currency.NewPairFromStrings(symbol.BaseAsset, symbol.QuoteAsset)
			if err != nil {
				return nil, fmt.Errorf("failed to create pair from %s/%s: %w",
					symbol.BaseAsset, symbol.QuoteAsset, err)
			}
			pairs = append(pairs, pair)
		}
	default:
		return nil, fmt.Errorf("unsupported asset type: %v", assetType)
	}

	b.logger.Info("Tradable pairs fetched", zap.String("asset", assetType.String()), zap.Int("count", len(pairs)))
	return pairs, nil
}

// StartTradablePairsCache 启动交易对缓存管理器
func (b *Binance) StartTradablePairsCache(ctx context.Context) error {
	if b.tradablePairsCache == nil {
		return fmt.Errorf("tradable pairs cache not initialized")
	}
	return b.tradablePairsCache.Start(ctx)
}

// GetTradablePairsFromCache 从缓存获取交易对
func (b *Binance) GetTradablePairsFromCache(ctx context.Context, assetType asset.Item) (currency.Pairs, error) {
	if b.tradablePairsCache == nil {
		// 如果缓存未启用，直接调用API
		return b.FetchTradablePairs(ctx, assetType)
	}
	return b.tradablePairsCache.GetTradablePairs(ctx, assetType)
}

// GetTradablePairsStats 获取交易对缓存统计信息
func (b *Binance) GetTradablePairsStats() map[string]interface{} {
	if b.tradablePairsCache == nil {
		return map[string]interface{}{
			"cache_enabled": false,
		}
	}
	return b.tradablePairsCache.GetCacheStats()
}

// IsSymbolSupported 检查指定交易对是否被支持
func (b *Binance) IsSymbolSupported(ctx context.Context, symbol currency.Pair, assetType asset.Item) (bool, error) {
	if b.tradablePairsCache == nil {
		// 如果缓存未启用，直接调用API检查
		pairs, err := b.FetchTradablePairs(ctx, assetType)
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
	return b.tradablePairsCache.IsSymbolSupported(ctx, symbol, assetType)
}

// ResolveTradingPairs 解析交易对配置，支持["*"]从API获取所有交易对
func (b *Binance) ResolveTradingPairs(ctx context.Context, symbols []string, assetType asset.Item) ([]string, error) {
	// 如果配置为["*"]，从API获取所有交易对
	if len(symbols) == 1 && symbols[0] == "*" {
		if b.config.TradablePairs.FetchFromAPI && b.tradablePairsCache != nil {
			// 从缓存获取
			return b.tradablePairsCache.GetSupportedSymbols(ctx, assetType)
		} else {
			// 直接从API获取
			pairs, err := b.FetchTradablePairs(ctx, assetType)
			if err != nil {
				return nil, err
			}
			result := make([]string, len(pairs))
			for i, pair := range pairs {
				result[i] = pair.String()
			}
			return result, nil
		}
	}

	// 返回原始配置的交易对
	return symbols, nil
}

// 工具方法

// FormatSymbol 格式化交易对符号
func FormatSymbol(pair currency.Pair, assetType asset.Item) (string, error) {
	return pair.Base.String() + pair.Quote.String(), nil
}
