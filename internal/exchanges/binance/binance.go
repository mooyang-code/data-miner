// Package binance 实现Binance交易所接口
package binance

import (
	"context"
	"fmt"
	"sync"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/mooyang-code/data-miner/internal/ipmanager"
	"github.com/mooyang-code/data-miner/internal/types"
	"github.com/mooyang-code/data-miner/pkg/cryptotrader/currency"
	"github.com/mooyang-code/data-miner/pkg/cryptotrader/exchange/websocket"
	"github.com/mooyang-code/data-miner/pkg/cryptotrader/exchanges/asset"
	"github.com/mooyang-code/data-miner/pkg/cryptotrader/exchanges/subscription"
)

// Binance 实现types.ExchangeInterface接口
type Binance struct {
	config      types.BinanceConfig           // Binance配置
	wsConn      *gws.Conn                     // WebSocket连接
	wsConnected bool                          // WebSocket连接状态
	lastPing    time.Time                     // 最后ping时间
	rateLimit   *types.RateLimit              // 速率限制

	// 订阅管理
	subscriptions map[string]types.DataCallback // 订阅回调映射
	mu            sync.RWMutex                  // 读写锁

	// 停止信号
	done chan struct{} // 停止信号通道

	// 请求计数器
	requestCount int64     // 请求计数
	lastReset    time.Time // 最后重置时间

	// IP管理器
	ipManager *ipmanager.Manager // IP管理器

	// websocket相关字段
	Name              string                    // 交易所名称
	Enabled           bool                      // 是否启用
	Verbose           bool                      // 详细日志
	HTTPTimeout       time.Duration             // HTTP超时时间
	Config            *Config                   // 配置
	Websocket         *websocket.Manager        // WebSocket管理器
	ValidateOrderbook bool                      // 验证订单簿
	obm               *orderbookManager         // 订单簿管理器
	Features          *Features                 // 功能特性
}

// New 创建新的Binance交易所实例
func New() *Binance {
	return &Binance{
		rateLimit: &types.RateLimit{
			RequestsPerMinute: 1200,
			LastRequest:       time.Now(),
		},
		subscriptions: make(map[string]types.DataCallback),
		done:          make(chan struct{}),
		lastReset:     time.Now(),
		ipManager:     ipmanager.New(ipmanager.DefaultConfig("stream.binance.com")),
		Name:          "Binance",
		Enabled:       true,
		Verbose:       false,
		HTTPTimeout:   30 * time.Second,
		Config: &Config{
			HTTPTimeout: 30 * time.Second,
		},
		ValidateOrderbook: true,
		Features: &Features{
			Subscriptions: subscription.List{},
		},
	}
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

// Close 关闭交易所连接
func (b *Binance) Close() error {
	close(b.done)
	if b.ipManager != nil {
		b.ipManager.Stop()
	}
	return b.WsClose()
}

// Config 交易所配置设置
type Config struct {
	HTTPTimeout time.Duration // HTTP超时时间
}

// Features 交易所支持的功能特性
type Features struct {
	Subscriptions subscription.List // 订阅列表
}

// orderbookManager 定义管理和维护连接与资产间同步的方式
type orderbookManager struct {
	state map[currency.Code]map[currency.Code]map[asset.Item]*update // 状态映射
	sync.Mutex                                                       // 互斥锁
	jobs chan job                                                    // 任务通道
}

// update 更新结构
type update struct {
	buffer            chan *WebsocketDepthStream // 缓冲通道
	fetchingBook      bool                       // 是否正在获取订单簿
	initialSync       bool                       // 是否初始同步
	needsFetchingBook bool                       // 是否需要获取订单簿
	lastUpdateID      int64                      // 最后更新ID
}

// job 定义同步任务，告诉协程通过REST协议获取订单簿
type job struct {
	Pair currency.Pair // 交易对
}

// IsEnabled 返回交易所是否启用
func (b *Binance) IsEnabled() bool {
	return b.Enabled
}

// GetWsAuthStreamKey 获取WebSocket认证流密钥
func (b *Binance) GetWsAuthStreamKey(ctx context.Context) (string, error) {
	// 简化实现 - 实际场景中应该调用API
	return "", nil
}

// MaintainWsAuthStreamKey 维护WebSocket认证流密钥
func (b *Binance) MaintainWsAuthStreamKey(ctx context.Context) error {
	// 简化实现
	return nil
}

// GetOrderBook 获取订单簿数据
func (b *Binance) GetOrderBook(ctx context.Context, params OrderBookDataRequestParams) (*OrderBook, error) {
	// 简化实现 - 实际场景中应该调用REST API
	return &OrderBook{}, nil
}

// MatchSymbolWithAvailablePairs 匹配交易对与可用交易对
func (b *Binance) MatchSymbolWithAvailablePairs(symbol string, assetType asset.Item, enabled bool) (currency.Pair, error) {
	// 简化实现
	pair, err := currency.NewPairFromString(symbol)
	if err != nil {
		return currency.EMPTYPAIR, err
	}
	return pair, nil
}

// MatchSymbolCheckEnabled 匹配交易对并检查是否启用
func (b *Binance) MatchSymbolCheckEnabled(symbol string, assetType asset.Item, enabled bool) (currency.Pair, bool, error) {
	pair, err := b.MatchSymbolWithAvailablePairs(symbol, assetType, enabled)
	if err != nil {
		return currency.EMPTYPAIR, false, err
	}
	return pair, true, nil
}

// GetRequestFormattedPairAndAssetType 获取格式化的交易对和资产类型
func (b *Binance) GetRequestFormattedPairAndAssetType(symbol string) (currency.Pair, asset.Item, error) {
	pair, err := currency.NewPairFromString(symbol)
	if err != nil {
		return currency.EMPTYPAIR, asset.Empty, err
	}
	return pair, asset.Spot, nil
}

// IsSaveTradeDataEnabled 返回是否启用交易数据保存
func (b *Binance) IsSaveTradeDataEnabled() bool {
	return false
}

// IsTradeFeedEnabled 返回是否启用交易数据推送
func (b *Binance) IsTradeFeedEnabled() bool {
	return true
}

// ParallelChanOp 执行并行通道操作
func (b *Binance) ParallelChanOp(ctx context.Context, channels subscription.List,
	fn func(context.Context, subscription.List) error, batchSize int) error {
	// 简化实现
	return fn(ctx, channels)
}

// CheckRateLimit 检查速率限制
func (b *Binance) CheckRateLimit() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()

	// 每分钟重置计数器
	if now.Sub(b.lastReset) >= time.Minute {
		b.requestCount = 0
		b.lastReset = now
	}

	// 检查是否超过限制
	if b.requestCount >= int64(b.rateLimit.RequestsPerMinute) {
		return fmt.Errorf("速率限制已超出")
	}

	b.requestCount++
	b.rateLimit.LastRequest = now

	return nil
}

// IsConnected 检查是否连接
func (b *Binance) IsConnected() bool {
	return b.wsConnected
}

// GetLastPing 获取最后ping时间
func (b *Binance) GetLastPing() time.Time {
	return b.lastPing
}

// GetRateLimit 获取速率限制信息
func (b *Binance) GetRateLimit() *types.RateLimit {
	return b.rateLimit
}

// GetTicker 获取单个交易对的行情数据
func (b *Binance) GetTicker(ctx context.Context, symbol types.Symbol) (*types.Ticker, error) {
	// 简化实现 - 实际应该调用REST API
	return &types.Ticker{
		Exchange:  types.ExchangeBinance,
		Symbol:    symbol,
		Price:     0,
		Volume:    0,
		Timestamp: time.Now(),
	}, nil
}

// GetOrderbook 获取订单簿数据
func (b *Binance) GetOrderbook(ctx context.Context, symbol types.Symbol, depth int) (*types.Orderbook, error) {
	// 简化实现 - 实际应该调用REST API
	return &types.Orderbook{
		Exchange:  types.ExchangeBinance,
		Symbol:    symbol,
		Bids:      []types.OrderbookEntry{},
		Asks:      []types.OrderbookEntry{},
		Timestamp: time.Now(),
	}, nil
}

// GetTrades 获取交易数据
func (b *Binance) GetTrades(ctx context.Context, symbol types.Symbol, limit int) ([]types.Trade, error) {
	// 简化实现 - 实际应该调用REST API
	return []types.Trade{}, nil
}

// GetKlines 获取K线数据
func (b *Binance) GetKlines(ctx context.Context, symbol types.Symbol, interval string, limit int) ([]types.Kline, error) {
	// 简化实现 - 实际应该调用REST API
	return []types.Kline{}, nil
}

// GetMultipleTickers 批量获取行情数据
func (b *Binance) GetMultipleTickers(ctx context.Context, symbols []types.Symbol) ([]types.Ticker, error) {
	var tickers []types.Ticker
	for _, symbol := range symbols {
		ticker, err := b.GetTicker(ctx, symbol)
		if err != nil {
			return nil, err
		}
		tickers = append(tickers, *ticker)
	}
	return tickers, nil
}

// GetMultipleOrderbooks 批量获取订单簿数据
func (b *Binance) GetMultipleOrderbooks(ctx context.Context, symbols []types.Symbol, depth int) ([]types.Orderbook, error) {
	var orderbooks []types.Orderbook
	for _, symbol := range symbols {
		orderbook, err := b.GetOrderbook(ctx, symbol, depth)
		if err != nil {
			return nil, err
		}
		orderbooks = append(orderbooks, *orderbook)
	}
	return orderbooks, nil
}

// SubscribeTicker 订阅行情数据
func (b *Binance) SubscribeTicker(symbols []types.Symbol, callback types.DataCallback) error {
	// 简化实现 - 实际应该通过websocket订阅
	return nil
}

// SubscribeOrderbook 订阅订单簿数据
func (b *Binance) SubscribeOrderbook(symbols []types.Symbol, callback types.DataCallback) error {
	// 简化实现 - 实际应该通过websocket订阅
	return nil
}

// SubscribeTrades 订阅交易数据
func (b *Binance) SubscribeTrades(symbols []types.Symbol, callback types.DataCallback) error {
	// 简化实现 - 实际应该通过websocket订阅
	return nil
}

// SubscribeKlines 订阅K线数据
func (b *Binance) SubscribeKlines(symbols []types.Symbol, intervals []string, callback types.DataCallback) error {
	// 简化实现 - 实际应该通过websocket订阅
	return nil
}

// UnsubscribeAll 取消所有订阅
func (b *Binance) UnsubscribeAll() error {
	// 简化实现 - 实际应该取消所有websocket订阅
	return nil
}
