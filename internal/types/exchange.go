// Package types 定义交易所接口类型
package types

import (
	"context"
	"time"
)

// ExchangeInterface 交易所接口定义
type ExchangeInterface interface {
	// GetName 获取交易所名称
	GetName() Exchange

	// Initialize 初始化交易所
	Initialize(config interface{}) error
	// Close 关闭交易所连接
	Close() error

	// GetTicker 获取单个交易对行情数据
	GetTicker(ctx context.Context, symbol Symbol) (*Ticker, error)
	// GetOrderbook 获取订单簿数据
	GetOrderbook(ctx context.Context, symbol Symbol, depth int) (*Orderbook, error)
	// GetTrades 获取交易数据
	GetTrades(ctx context.Context, symbol Symbol, limit int) ([]Trade, error)
	// GetKlines 获取K线数据
	GetKlines(ctx context.Context, symbol Symbol, interval string, limit int) ([]Kline, error)

	// GetMultipleTickers 批量获取行情数据
	GetMultipleTickers(ctx context.Context, symbols []Symbol) ([]Ticker, error)
	// GetMultipleOrderbooks 批量获取订单簿数据
	GetMultipleOrderbooks(ctx context.Context, symbols []Symbol, depth int) ([]Orderbook, error)

	// SubscribeTicker 订阅行情数据
	SubscribeTicker(symbols []Symbol, callback DataCallback) error
	// SubscribeOrderbook 订阅订单簿数据
	SubscribeOrderbook(symbols []Symbol, callback DataCallback) error
	// SubscribeTrades 订阅交易数据
	SubscribeTrades(symbols []Symbol, callback DataCallback) error
	// SubscribeKlines 订阅K线数据
	SubscribeKlines(symbols []Symbol, intervals []string, callback DataCallback) error

	// UnsubscribeAll 取消所有订阅
	UnsubscribeAll() error

	// IsConnected 检查连接状态
	IsConnected() bool
	// GetLastPing 获取最后ping时间
	GetLastPing() time.Time

	// GetRateLimit 获取速率限制信息
	GetRateLimit() *RateLimit
	// CheckRateLimit 检查速率限制
	CheckRateLimit() error
}

// RateLimit 速率限制结构
type RateLimit struct {
	RequestsPerSecond int       // 每秒请求数限制
	RequestsPerMinute int       // 每分钟请求数限制
	RequestsPerHour   int       // 每小时请求数限制
	LastRequest       time.Time // 最后请求时间
	RequestCount      int       // 请求计数
}

// ExchangeConfig 交易所基础配置接口
type ExchangeConfig interface {
	GetAPIURL() string      // 获取API地址
	GetWebsocketURL() string // 获取WebSocket地址
	GetAPIKey() string      // 获取API密钥
	GetAPISecret() string   // 获取API密钥
	IsEnabled() bool        // 是否启用
}

// DataFetcher 数据获取器接口
type DataFetcher interface {
	// FetchData 获取数据
	FetchData(ctx context.Context, dataType DataType, symbols []Symbol, params map[string]interface{}) ([]MarketData, error)
}

// WebSocketManager WebSocket管理器接口
type WebSocketManager interface {
	Connect(url string) error                                 // 连接WebSocket
	Disconnect() error                                        // 断开连接
	Subscribe(channel string, symbols []Symbol) error        // 订阅频道
	Unsubscribe(channel string, symbols []Symbol) error      // 取消订阅
	IsConnected() bool                                        // 检查连接状态
	SetCallback(callback func([]byte) error)                 // 设置回调函数
}
