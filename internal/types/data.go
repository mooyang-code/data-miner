// Package types 定义数据挖掘器的数据类型
package types

import (
	"time"
)

// DataType 数据类型枚举
type DataType string

const (
	DataTypeTicker    DataType = "ticker"    // 行情数据
	DataTypeOrderbook DataType = "orderbook" // 订单簿数据
	DataTypeTrades    DataType = "trades"    // 交易数据
	DataTypeKlines    DataType = "klines"    // K线数据
)

// Exchange 交易所枚举
type Exchange string

const (
	ExchangeBinance Exchange = "binance" // Binance交易所
)

// Symbol 交易对符号
type Symbol string

// Ticker 行情数据
type Ticker struct {
	Exchange  Exchange  `json:"exchange"`   // 交易所
	Symbol    Symbol    `json:"symbol"`     // 交易对
	Price     float64   `json:"price"`      // 当前价格
	Volume    float64   `json:"volume"`     // 成交量
	High24h   float64   `json:"high_24h"`   // 24小时最高价
	Low24h    float64   `json:"low_24h"`    // 24小时最低价
	Change24h float64   `json:"change_24h"` // 24小时涨跌幅
	Timestamp time.Time `json:"timestamp"`  // 时间戳
}

// OrderbookEntry 订单簿条目
type OrderbookEntry struct {
	Price    float64 `json:"price"`    // 价格
	Quantity float64 `json:"quantity"` // 数量
}

// Orderbook 订单簿数据
type Orderbook struct {
	Exchange  Exchange         `json:"exchange"`  // 交易所
	Symbol    Symbol           `json:"symbol"`    // 交易对
	Bids      []OrderbookEntry `json:"bids"`      // 买单列表
	Asks      []OrderbookEntry `json:"asks"`      // 卖单列表
	Timestamp time.Time        `json:"timestamp"` // 时间戳
}

// Trade 交易数据
type Trade struct {
	Exchange  Exchange  `json:"exchange"`  // 交易所
	Symbol    Symbol    `json:"symbol"`    // 交易对
	ID        string    `json:"id"`        // 交易ID
	Price     float64   `json:"price"`     // 成交价格
	Quantity  float64   `json:"quantity"`  // 成交数量
	Side      string    `json:"side"`      // 买卖方向 ("buy" or "sell")
	Timestamp time.Time `json:"timestamp"` // 时间戳
}

// Kline K线数据
type Kline struct {
	Exchange    Exchange  `json:"exchange"`     // 交易所
	Symbol      Symbol    `json:"symbol"`       // 交易对
	Interval    string    `json:"interval"`     // 时间间隔 ("1m", "5m", "1h", "1d" etc.)
	OpenTime    time.Time `json:"open_time"`    // 开盘时间
	CloseTime   time.Time `json:"close_time"`   // 收盘时间
	OpenPrice   float64   `json:"open_price"`   // 开盘价
	HighPrice   float64   `json:"high_price"`   // 最高价
	LowPrice    float64   `json:"low_price"`    // 最低价
	ClosePrice  float64   `json:"close_price"`  // 收盘价
	Volume      float64   `json:"volume"`       // 成交量
	TradeCount  int64     `json:"trade_count"`  // 成交笔数
	TakerVolume float64   `json:"taker_volume"` // 主动买入成交量
}

// MarketData 通用市场数据接口
type MarketData interface {
	GetExchange() Exchange   // 获取交易所
	GetSymbol() Symbol       // 获取交易对
	GetTimestamp() time.Time // 获取时间戳
	GetDataType() DataType   // 获取数据类型
}

// Ticker实现MarketData接口
func (t *Ticker) GetExchange() Exchange   { return t.Exchange }
func (t *Ticker) GetSymbol() Symbol       { return t.Symbol }
func (t *Ticker) GetTimestamp() time.Time { return t.Timestamp }
func (t *Ticker) GetDataType() DataType   { return DataTypeTicker }

// Orderbook实现MarketData接口
func (o *Orderbook) GetExchange() Exchange   { return o.Exchange }
func (o *Orderbook) GetSymbol() Symbol       { return o.Symbol }
func (o *Orderbook) GetTimestamp() time.Time { return o.Timestamp }
func (o *Orderbook) GetDataType() DataType   { return DataTypeOrderbook }

// Trade实现MarketData接口
func (t *Trade) GetExchange() Exchange   { return t.Exchange }
func (t *Trade) GetSymbol() Symbol       { return t.Symbol }
func (t *Trade) GetTimestamp() time.Time { return t.Timestamp }
func (t *Trade) GetDataType() DataType   { return DataTypeTrades }

// Kline实现MarketData接口
func (k *Kline) GetExchange() Exchange   { return k.Exchange }
func (k *Kline) GetSymbol() Symbol       { return k.Symbol }
func (k *Kline) GetTimestamp() time.Time { return k.OpenTime }
func (k *Kline) GetDataType() DataType   { return DataTypeKlines }

// DataCallback 数据回调函数类型
type DataCallback func(data MarketData) error
