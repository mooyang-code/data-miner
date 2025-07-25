// Package binance 实现Binance REST API接口（重构版本，使用通用HTTP客户端）
package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/mooyang-code/data-miner/internal/exchanges/asset"
	"github.com/mooyang-code/data-miner/internal/exchanges/httpclient"
	"github.com/mooyang-code/data-miner/internal/ipmanager"
	"github.com/mooyang-code/data-miner/internal/types"
	"github.com/mooyang-code/data-miner/pkg/cryptotrader/currency"
	"github.com/mooyang-code/data-miner/pkg/cryptotrader/log"
)

// API 路径常量
const (
	// 基础URL
	apiURL = "https://api.binance.com"

	// 公共接口路径
	exchangeInfo     = "/api/v3/exchangeInfo"
	orderBookDepth   = "/api/v3/depth"
	recentTrades     = "/api/v3/trades"
	aggregatedTrades = "/api/v3/aggTrades"
	candleStick      = "/api/v3/klines"
	averagePrice     = "/api/v3/avgPrice"
	priceChange      = "/api/v3/ticker/24hr"
	symbolPrice      = "/api/v3/ticker/price"
	bestPrice        = "/api/v3/ticker/bookTicker"
	historicalTrades = "/api/v3/historicalTrades"

	// 认证接口路径
	userAccountStream = "/api/v3/userDataStream"
	allOrders         = "/api/v3/allOrders"
	orderEndpoint     = "/api/v3/order"
)

// BinanceRestAPI REST API 客户端（重构版本）
type BinanceRestAPI struct {
	config     types.BinanceConfig // Binance配置
	httpClient httpclient.Client   // HTTP客户端

	// 状态管理
	mu      sync.RWMutex // 读写锁
	Name    string       // 交易所名称
	Enabled bool         // 是否启用
	Verbose bool         // 详细日志
}

// NewRestAPI 创建新的Binance REST API客户端实例（重构版本）
func NewRestAPI() *BinanceRestAPI {
	// 创建HTTP客户端
	httpClient, err := NewHTTPClient()
	if err != nil {
		log.Errorf(log.ExchangeSys, "Failed to create HTTP client for Binance: %v", err)
		return nil
	}

	// 设置默认请求头
	httpClient.SetHeaders(map[string]string{
		"Content-Type": "application/json",
		"User-Agent":   "crypto-data-miner/1.0.0",
	})

	// 创建REST API实例
	api := &BinanceRestAPI{
		httpClient: httpClient,
		Name:       "Binance",
		Enabled:    true,
		Verbose:    false,
	}
	log.Infof(log.ExchangeSys, "Binance REST API client created successfully")
	return api
}

// GetName 返回交易所名称
func (b *BinanceRestAPI) GetName() types.Exchange {
	return types.ExchangeBinance
}

// Initialize 初始化交易所
func (b *BinanceRestAPI) Initialize(config interface{}) error {
	binanceConfig, ok := config.(types.BinanceConfig)
	if !ok {
		b.config = types.BinanceConfig{} // 使用默认配置
	} else {
		b.config = binanceConfig
	}

	log.Infof(log.ExchangeSys, "Binance REST API initialized successfully")
	return nil
}

// Close 关闭REST API客户端
func (b *BinanceRestAPI) Close() error {
	if b.httpClient != nil {
		if err := b.httpClient.Close(); err != nil {
			log.Errorf(log.ExchangeSys, "Failed to close HTTP client: %v", err)
		}
		log.Infof(log.ExchangeSys, "Binance REST API client closed")
	}
	return nil
}

// IsEnabled 返回交易所是否启用
func (b *BinanceRestAPI) IsEnabled() bool {
	return b.Enabled
}

// SendHTTPRequest 发送未认证的HTTP请求，支持重试和超时
func (b *BinanceRestAPI) SendHTTPRequest(ctx context.Context, path string, result interface{}) error {
	fullURL := apiURL + path

	if b.Verbose {
		log.Debugf(log.ExchangeSys, "Making GET request to %s", fullURL)
	}

	// 使用重试机制
	return b.sendHTTPRequestWithRetry(ctx, fullURL, result, 3)
}

// sendHTTPRequestWithRetry 使用 retry 库发送HTTP请求并支持重试
func (b *BinanceRestAPI) sendHTTPRequestWithRetry(ctx context.Context, fullURL string, result interface{}, maxRetries int) error {
	var lastErr error

	err := retry.Do(
		func() error {
			// 为每次重试创建新的上下文，设置超时时间
			requestCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			// 执行HTTP请求
			err := b.httpClient.Get(requestCtx, fullURL, result)
			if err != nil {
				lastErr = err
				log.Warnf(log.ExchangeSys, "Binance REST API request failed: %v", err)
				return err
			}

			if b.Verbose {
				log.Debugf(log.ExchangeSys, "Binance: Request successful")
			}
			return nil
		},
		retry.Attempts(uint(maxRetries)),
		retry.Delay(2*time.Second),
		retry.DelayType(retry.BackOffDelay),
		retry.MaxDelay(10*time.Second),
		retry.Context(ctx),
		retry.OnRetry(func(n uint, err error) {
			log.Warnf(log.ExchangeSys, "Binance REST API retry attempt %d/%d: %v", n+1, maxRetries, err)
		}),
		retry.RetryIf(func(err error) bool {
			// 可以根据错误类型决定是否重试
			// 例如：网络错误、超时错误等可以重试，但认证错误等不应该重试
			return true // 暂时对所有错误都重试
		}),
	)

	if err != nil {
		return fmt.Errorf("httpClient 请求失败，已重试 %d 次: %w", maxRetries, lastErr)
	}
	return nil
}

// GetOrderbook 获取订单簿
func (b *BinanceRestAPI) GetOrderbook(ctx context.Context, symbol currency.Pair, limit int) (OrderBook, error) {
	var resp OrderBookData
	urlParams := url.Values{}

	symbolValue, err := FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return OrderBook{}, err
	}
	urlParams.Set("symbol", symbolValue)

	if limit > 0 {
		urlParams.Set("limit", strconv.Itoa(limit))
	}
	path := orderBookDepth + "?" + urlParams.Encode()
	if err := b.SendHTTPRequest(ctx, path, &resp); err != nil {
		return OrderBook{}, err
	}

	// 转换响应格式
	orderbook := OrderBook{
		Symbol:       symbol.String(),
		LastUpdateID: resp.LastUpdateID,
		Code:         resp.Code,
		Msg:          resp.Msg,
		Bids:         make([]OrderbookItem, len(resp.Bids)),
		Asks:         make([]OrderbookItem, len(resp.Asks)),
	}

	// 转换买单
	for i, bid := range resp.Bids {
		orderbook.Bids[i] = OrderbookItem{
			Price:    bid[0].Float64(),
			Quantity: bid[1].Float64(),
		}
	}

	// 转换卖单
	for i, ask := range resp.Asks {
		orderbook.Asks[i] = OrderbookItem{
			Price:    ask[0].Float64(),
			Quantity: ask[1].Float64(),
		}
	}
	return orderbook, nil
}

// GetKlines 获取K线数据
func (b *BinanceRestAPI) GetKlines(ctx context.Context, symbol currency.Pair, interval string, limit int, startTime, endTime int64) ([]CandleStick, error) {
	urlParams := url.Values{}
	symbolValue, err := FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	urlParams.Set("symbol", symbolValue)
	urlParams.Set("interval", interval)

	if limit > 0 {
		urlParams.Set("limit", strconv.Itoa(limit))
	}
	if startTime > 0 {
		urlParams.Set("startTime", strconv.FormatInt(startTime, 10))
	}
	if endTime > 0 {
		urlParams.Set("endTime", strconv.FormatInt(endTime, 10))
	}

	var resp []CandleStick
	path := candleStick + "?" + urlParams.Encode()
	if err := b.SendHTTPRequest(ctx, path, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetLatestSpotPrice 获取最新现货价格
func (b *BinanceRestAPI) GetLatestSpotPrice(ctx context.Context, symbol currency.Pair) (SymbolPrice, error) {
	resp := SymbolPrice{}
	urlParams := url.Values{}

	if !symbol.IsEmpty() {
		symbolValue, err := FormatSymbol(symbol, asset.Spot)
		if err != nil {
			return resp, err
		}
		urlParams.Set("symbol", symbolValue)
	}

	path := symbolPrice + "?" + urlParams.Encode()
	if err := b.SendHTTPRequest(ctx, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetExchangeInfo 获取交易所信息
func (b *BinanceRestAPI) GetExchangeInfo(ctx context.Context) (ExchangeInfo, error) {
	var resp ExchangeInfo
	if err := b.SendHTTPRequest(ctx, exchangeInfo, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetTickers 获取24小时价格变化统计
func (b *BinanceRestAPI) GetTickers(ctx context.Context, symbols ...currency.Pair) ([]PriceChangeStats, error) {
	var resp []PriceChangeStats
	urlParams := url.Values{}

	if len(symbols) == 1 {
		symbolValue, err := FormatSymbol(symbols[0], asset.Spot)
		if err != nil {
			return nil, err
		}
		urlParams.Set("symbol", symbolValue)
	}

	path := priceChange + "?" + urlParams.Encode()
	if err := b.SendHTTPRequest(ctx, path, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// CheckRateLimit 检查速率限制
func (b *BinanceRestAPI) CheckRateLimit() error {
	// 新的HTTP客户端内部处理速率限制
	return nil
}

// IsConnected 检查连接状态
func (b *BinanceRestAPI) IsConnected() bool {
	if b.httpClient == nil {
		return false
	}
	status := b.httpClient.GetStatus()
	return status.Running
}

// GetTicker 获取单个交易对的价格统计
func (b *BinanceRestAPI) GetTicker(ctx context.Context, symbol string) (PriceChangeStats, error) {
	pair, err := currency.NewPairFromString(symbol)
	if err != nil {
		return PriceChangeStats{}, err
	}

	tickers, err := b.GetTickers(ctx, pair)
	if err != nil {
		return PriceChangeStats{}, err
	}

	if len(tickers) == 0 {
		return PriceChangeStats{}, fmt.Errorf("no ticker data found for symbol %s", symbol)
	}
	return tickers[0], nil
}

// GetTrades 获取交易数据
func (b *BinanceRestAPI) GetTrades(ctx context.Context, symbol string) ([]RecentTrade, error) {
	// 这个方法需要实现，暂时返回空
	return []RecentTrade{}, fmt.Errorf("GetTrades method not implemented yet")
}

// GetMultipleTickers 获取多个交易对的价格统计
func (b *BinanceRestAPI) GetMultipleTickers(ctx context.Context, symbols []string) ([]PriceChangeStats, error) {
	if len(symbols) == 0 {
		return b.GetTickers(ctx)
	}

	var pairs []currency.Pair
	for _, symbol := range symbols {
		pair, err := currency.NewPairFromString(symbol)
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, pair)
	}
	return b.GetTickers(ctx, pairs...)
}

// GetMultipleOrderbooks 获取多个交易对的订单簿
func (b *BinanceRestAPI) GetMultipleOrderbooks(ctx context.Context, symbols []string, limit int) ([]OrderBook, error) {
	var orderbooks []OrderBook

	for _, symbol := range symbols {
		pair, err := currency.NewPairFromString(symbol)
		if err != nil {
			return nil, err
		}

		orderbook, err := b.GetOrderbook(ctx, pair, limit)
		if err != nil {
			return nil, err
		}

		orderbooks = append(orderbooks, orderbook)
	}
	return orderbooks, nil
}

// GetStatus 获取客户端状态
func (b *BinanceRestAPI) GetStatus() map[string]interface{} {
	if b.httpClient == nil {
		return map[string]interface{}{
			"name":    b.Name,
			"enabled": b.Enabled,
			"error":   "HTTP client not initialized",
		}
	}

	status := b.httpClient.GetStatus()
	return map[string]interface{}{
		"name":        b.Name,
		"enabled":     b.Enabled,
		"http_client": status,
	}
}

// HTTP客户端配置相关函数

// NewHTTPClient 创建Binance专用的HTTP客户端
func NewHTTPClient() (httpclient.Client, error) {
	config := createBinanceHTTPConfig()
	return httpclient.New(config)
}

// createBinanceHTTPConfig 创建Binance专用的HTTP客户端配置
func createBinanceHTTPConfig() *httpclient.Config {
	config := httpclient.DefaultConfig("binance")

	// 启用动态IP
	config.DynamicIP.Enabled = true
	config.DynamicIP.Hostname = "api.binance.com"
	config.DynamicIP.IPManager = ipmanager.DefaultConfig("api.binance.com")

	// 调整重试配置
	config.Retry.MaxAttempts = 5
	config.Retry.InitialDelay = time.Second
	config.Retry.MaxDelay = 8 * time.Second

	// 调整速率限制（Binance限制）
	config.RateLimit.RequestsPerMinute = 1200

	// 启用调试日志
	config.Debug = false
	return config
}

// NewHTTPClientWithCustomConfig 使用自定义配置创建HTTP客户端
func NewHTTPClientWithCustomConfig(enableDynamicIP bool, debug bool) (httpclient.Client, error) {
	config := createBinanceHTTPConfig()

	// 应用自定义配置
	config.DynamicIP.Enabled = enableDynamicIP
	config.Debug = debug
	return httpclient.New(config)
}

// 适配器方法 - 将internal/types接口转换为Binance特定的实现

// GetTickerBySymbol 获取单个交易对的行情数据（适配器方法）
func (b *BinanceRestAPI) GetTickerBySymbol(ctx context.Context, symbol string) (PriceChangeStats, error) {
	pair, err := currency.NewPairFromString(symbol)
	if err != nil {
		return PriceChangeStats{}, err
	}

	tickers, err := b.GetTickers(ctx, pair)
	if err != nil {
		return PriceChangeStats{}, err
	}

	if len(tickers) == 0 {
		return PriceChangeStats{}, fmt.Errorf("no ticker data found for symbol %s", symbol)
	}
	return tickers[0], nil
}

// GetTradesBySymbol 获取交易数据（适配器方法）
func (b *BinanceRestAPI) GetTradesBySymbol(ctx context.Context, symbol string) ([]RecentTrade, error) {
	// 解析交易对
	pair, err := currency.NewPairFromString(symbol)
	if err != nil {
		return nil, fmt.Errorf("无效的交易对格式: %v", err)
	}

	// 格式化交易对符号
	formattedSymbol, err := FormatSymbol(pair, asset.Spot)
	if err != nil {
		return nil, err
	}

	// 构建URL参数
	urlParams := url.Values{}
	urlParams.Set("symbol", formattedSymbol)
	urlParams.Set("limit", "500") // 默认获取500条交易记录

	// 构建请求路径
	path := recentTrades + "?" + urlParams.Encode()

	// 发送HTTP请求
	var resp []RecentTrade
	if err := b.SendHTTPRequest(ctx, path, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetTimeAndWeight 获取服务器时间和当前权重使用情况
func (b *BinanceRestAPI) GetTimeAndWeight(ctx context.Context) (int64, int, error) {
	var resp struct {
		ServerTime int64 `json:"serverTime"`
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL+"/api/v3/time", nil)
	if err != nil {
		return 0, 0, err
	}

	// 发送请求
	httpResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer httpResp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return 0, 0, err
	}

	// 解析JSON响应
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, 0, err
	}

	// 从响应头获取权重信息
	weightStr := httpResp.Header.Get("X-MBX-USED-WEIGHT-1M")
	weight := 0
	if weightStr != "" {
		if w, err := strconv.Atoi(weightStr); err == nil {
			weight = w
		}
	}

	return resp.ServerTime, weight, nil
}

// GetKlinesForSymbol 获取K线数据（types.Symbol版本）
func (b *BinanceRestAPI) GetKlinesForSymbol(ctx context.Context, symbol types.Symbol, interval string, limit int) ([]types.Kline, error) {
	// 转换符号格式
	pair, err := currency.NewPairFromString(string(symbol))
	if err != nil {
		return nil, fmt.Errorf("无效的交易对格式: %v", err)
	}

	// 调用内部方法获取K线数据
	klines, err := b.GetKlines(ctx, pair, interval, limit, 0, 0)
	if err != nil {
		return nil, err
	}

	// 转换为通用类型
	result := make([]types.Kline, len(klines))
	for i, kline := range klines {
		result[i] = types.Kline{
			Exchange:    types.ExchangeBinance,
			Symbol:      symbol,
			Interval:    interval,
			OpenTime:    kline.OpenTime.Time(),
			CloseTime:   kline.CloseTime.Time(),
			OpenPrice:   kline.Open.Float64(),
			HighPrice:   kline.High.Float64(),
			LowPrice:    kline.Low.Float64(),
			ClosePrice:  kline.Close.Float64(),
			Volume:      kline.Volume.Float64(),
			TradeCount:  kline.TradeCount,
			TakerVolume: kline.TakerBuyAssetVolume.Float64(),
		}
	}

	return result, nil
}
