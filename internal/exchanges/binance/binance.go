// Package binance 实现Binance交易所公共接口和结构
package binance

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/mooyang-code/data-miner/internal/types"
	"github.com/mooyang-code/data-miner/pkg/cryptotrader/currency"
	"github.com/mooyang-code/data-miner/pkg/cryptotrader/exchanges/asset"
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
	return nil
}

// IsEnabled 返回交易所是否启用
func (b *Binance) IsEnabled() bool {
	return b.Enabled
}

// Close 关闭交易所连接
func (b *Binance) Close() error {
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
	// 转换symbol为currency.Pair
	pair, err := currency.NewPairFromString(string(symbol))
	if err != nil {
		return nil, err
	}

	// 调用RestAPI获取Binance特定的数据
	binanceKlines, err := b.RestAPI.GetKlines(ctx, pair, interval, limit, 0, 0)
	if err != nil {
		return nil, err
	}

	// 转换为通用类型
	klines := make([]types.Kline, len(binanceKlines))
	for i, binanceKline := range binanceKlines {
		klines[i] = types.Kline{
			Exchange:    types.ExchangeBinance,
			Symbol:      symbol,
			Interval:    interval,
			OpenTime:    binanceKline.OpenTime.Time(),
			CloseTime:   binanceKline.CloseTime.Time(),
			OpenPrice:   binanceKline.Open.Float64(),
			HighPrice:   binanceKline.High.Float64(),
			LowPrice:    binanceKline.Low.Float64(),
			ClosePrice:  binanceKline.Close.Float64(),
			Volume:      binanceKline.Volume.Float64(),
			TradeCount:  binanceKline.TradeCount,
			TakerVolume: binanceKline.TakerBuyAssetVolume.Float64(),
		}
	}

	return klines, nil
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

// 工具方法

// FormatSymbol 格式化交易对符号
func FormatSymbol(pair currency.Pair, assetType asset.Item) (string, error) {
	return pair.Base.String() + pair.Quote.String(), nil
}
