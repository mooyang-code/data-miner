// Package binance 定义Binance交易所的数据类型
package binance

import (
	"time"

	"github.com/mooyang-code/data-miner/pkg/cryptotrader/currency"
	"github.com/mooyang-code/data-miner/pkg/cryptotrader/encoding/json"
	"github.com/mooyang-code/data-miner/pkg/cryptotrader/types"
	"github.com/shopspring/decimal"
)

// 提现状态码描述
const (
	EmailSent        = iota // 邮件已发送
	Cancelled               // 已取消
	AwaitingApproval        // 等待审批
	Rejected                // 已拒绝
	Processing              // 处理中
	Failure                 // 失败
	Completed               // 已完成
)

// filterType 过滤器类型
type filterType string

const (
	priceFilter              filterType = "PRICE_FILTER"              // 价格过滤器
	lotSizeFilter            filterType = "LOT_SIZE"                  // 数量过滤器
	icebergPartsFilter       filterType = "ICEBERG_PARTS"            // 冰山订单过滤器
	marketLotSizeFilter      filterType = "MARKET_LOT_SIZE"          // 市价单数量过滤器
	trailingDeltaFilter      filterType = "TRAILING_DELTA"           // 跟踪止损过滤器
	percentPriceFilter       filterType = "PERCENT_PRICE"            // 百分比价格过滤器
	percentPriceBySizeFilter filterType = "PERCENT_PRICE_BY_SIDE"    // 按边百分比价格过滤器
	notionalFilter           filterType = "NOTIONAL"                 // 名义价值过滤器
	maxNumOrdersFilter       filterType = "MAX_NUM_ORDERS"           // 最大订单数过滤器
	maxNumAlgoOrdersFilter   filterType = "MAX_NUM_ALGO_ORDERS"      // 最大算法订单数过滤器
)

// ExchangeInfo 交易所完整信息类型
type ExchangeInfo struct {
	Code       int        `json:"code"`       // 状态码
	Msg        string     `json:"msg"`        // 消息
	Timezone   string     `json:"timezone"`   // 时区
	ServerTime types.Time `json:"serverTime"` // 服务器时间
	RateLimits []*struct {
		RateLimitType string `json:"rateLimitType"` // 速率限制类型
		Interval      string `json:"interval"`      // 间隔
		Limit         int    `json:"limit"`         // 限制
	} `json:"rateLimits"` // 速率限制
	ExchangeFilters any `json:"exchangeFilters"` // 交易所过滤器
	Symbols         []*struct {
		Symbol                     string        `json:"symbol"`                     // 交易对
		Status                     string        `json:"status"`                     // 状态
		BaseAsset                  string        `json:"baseAsset"`                  // 基础资产
		BaseAssetPrecision         int           `json:"baseAssetPrecision"`         // 基础资产精度
		QuoteAsset                 string        `json:"quoteAsset"`                 // 计价资产
		QuotePrecision             int           `json:"quotePrecision"`             // 计价精度
		OrderTypes                 []string      `json:"orderTypes"`                 // 订单类型
		IcebergAllowed             bool          `json:"icebergAllowed"`             // 是否允许冰山订单
		OCOAllowed                 bool          `json:"ocoAllowed"`                 // 是否允许OCO订单
		QuoteOrderQtyMarketAllowed bool          `json:"quoteOrderQtyMarketAllowed"` // 是否允许计价数量市价单
		IsSpotTradingAllowed       bool          `json:"isSpotTradingAllowed"`       // 是否允许现货交易
		IsMarginTradingAllowed     bool          `json:"isMarginTradingAllowed"`     // 是否允许保证金交易
		Filters                    []*filterData `json:"filters"`                    // 过滤器
		Permissions                []string      `json:"permissions"`                // 权限
		PermissionSets             [][]string    `json:"permissionSets"`             // 权限集合
	} `json:"symbols"` // 交易对列表
}

// filterData 过滤器数据
type filterData struct {
	FilterType          filterType `json:"filterType"`          // 过滤器类型
	MinPrice            float64    `json:"minPrice,string"`     // 最小价格
	MaxPrice            float64    `json:"maxPrice,string"`     // 最大价格
	TickSize            float64    `json:"tickSize,string"`     // 价格步长
	MultiplierUp        float64    `json:"multiplierUp,string"` // 上涨乘数
	MultiplierDown      float64    `json:"multiplierDown,string"` // 下跌乘数
	AvgPriceMinutes     int64      `json:"avgPriceMins"`        // 平均价格分钟数
	MinQty              float64    `json:"minQty,string"`       // 最小数量
	MaxQty              float64    `json:"maxQty,string"`       // 最大数量
	StepSize            float64    `json:"stepSize,string"`     // 数量步长
	MinNotional         float64    `json:"minNotional,string"`  // 最小名义价值
	ApplyToMarket       bool       `json:"applyToMarket"`       // 是否应用于市价单
	Limit               int64      `json:"limit"`               // 限制
	MaxNumAlgoOrders    int64      `json:"maxNumAlgoOrders"`    // 最大算法订单数
	MaxNumIcebergOrders int64      `json:"maxNumIcebergOrders"` // 最大冰山订单数
	MaxNumOrders        int64      `json:"maxNumOrders"`        // 最大订单数
}

// CoinInfo 存储所有支持币种的信息
type CoinInfo struct {
	Coin              string  `json:"coin"`              // 币种
	DepositAllEnable  bool    `json:"depositAllEnable"`  // 是否启用所有充值
	WithdrawAllEnable bool    `json:"withdrawAllEnable"` // 是否启用所有提现
	Free              float64 `json:"free,string"`       // 可用余额
	Freeze            float64 `json:"freeze,string"`     // 冻结余额
	IPOAble           float64 `json:"ipoable,string"`    // 可IPO数量
	IPOing            float64 `json:"ipoing,string"`     // IPO中数量
	IsLegalMoney      bool    `json:"isLegalMoney"`      // 是否法币
	Locked            float64 `json:"locked,string"`     // 锁定余额
	Name              string  `json:"name"`              // 币种名称
	NetworkList       []struct {
		AddressRegex        string  `json:"addressRegex"`        // 地址正则
		Coin                string  `json:"coin"`                // 币种
		DepositDescription  string  `json:"depositDesc"`         // 充值描述（仅在"depositEnable"为false时显示）
		DepositEnable       bool    `json:"depositEnable"`       // 是否启用充值
		IsDefault           bool    `json:"isDefault"`           // 是否默认
		MemoRegex           string  `json:"memoRegex"`           // 备注正则
		MinimumConfirmation uint16  `json:"minConfirm"`          // 最小确认数
		Name                string  `json:"name"`                // 网络名称
		Network             string  `json:"network"`             // 网络
		ResetAddressStatus  bool    `json:"resetAddressStatus"`  // 重置地址状态
		SpecialTips         string  `json:"specialTips"`         // 特殊提示
		UnlockConfirm       uint16  `json:"unLockConfirm"`       // 解锁确认数
		WithdrawDescription string  `json:"withdrawDesc"`        // 提现描述（仅在"withdrawEnable"为false时显示）
		WithdrawEnable      bool    `json:"withdrawEnable"`      // 是否启用提现
		WithdrawFee         float64 `json:"withdrawFee,string"`  // 提现手续费
		WithdrawMinimum     float64 `json:"withdrawMin,string"`  // 最小提现金额
		WithdrawMaximum     float64 `json:"withdrawMax,string"`  // 最大提现金额
	} `json:"networkList"` // 网络列表
	Storage     float64 `json:"storage,string"`     // 存储
	Trading     bool    `json:"trading"`            // 是否可交易
	Withdrawing float64 `json:"withdrawing,string"` // 提现中
}

// OrderBookDataRequestParams 订单簿数据请求参数
type OrderBookDataRequestParams struct {
	Symbol currency.Pair `json:"symbol"` // 必填字段；示例 LTCBTC,BTCUSDT
	Limit  int           `json:"limit"`  // 默认100；最大5000。如果limit > 5000，响应将截断为5000
}

// OrderbookItem 存储单个订单簿项目
type OrderbookItem struct {
	Price    float64 // 价格
	Quantity float64 // 数量
}

// OrderBookData 订单簿端点的响应数据
type OrderBookData struct {
	Code         int               `json:"code"`         // 状态码
	Msg          string            `json:"msg"`          // 消息
	LastUpdateID int64             `json:"lastUpdateId"` // 最后更新ID
	Bids         [][2]types.Number `json:"bids"`         // 买单
	Asks         [][2]types.Number `json:"asks"`         // 卖单
}

// OrderBook 可用于订单簿的实际结构化数据
type OrderBook struct {
	Symbol       string          // 交易对
	LastUpdateID int64           // 最后更新ID
	Code         int             // 状态码
	Msg          string          // 消息
	Bids         []OrderbookItem // 买单列表
	Asks         []OrderbookItem // 卖单列表
}

// DepthUpdateParams 用作WebsocketDepthStream的嵌入类型
type DepthUpdateParams []struct {
	PriceLevel float64 // 价格级别
	Quantity   float64 // 数量
	ignore     []any   // 忽略字段
}

// WebsocketDepthStream 更新深度流的差异
type WebsocketDepthStream struct {
	Event         string            `json:"e"` // 事件类型
	Timestamp     types.Time        `json:"E"` // 时间戳
	Pair          string            `json:"s"` // 交易对
	FirstUpdateID int64             `json:"U"` // 第一个更新ID
	LastUpdateID  int64             `json:"u"` // 最后更新ID
	UpdateBids    [][2]types.Number `json:"b"` // 更新买单
	UpdateAsks    [][2]types.Number `json:"a"` // 更新卖单
}

// RecentTradeRequestParams 最近交易请求参数
type RecentTradeRequestParams struct {
	Symbol currency.Pair `json:"symbol"` // 必填字段。示例 LTCBTC, BTCUSDT
	Limit  int           `json:"limit"`  // 默认500；最大500
}

// RecentTrade 保存最近交易数据
type RecentTrade struct {
	ID           int64      `json:"id"`           // 交易ID
	Price        float64    `json:"price,string"` // 价格
	Quantity     float64    `json:"qty,string"`   // 数量
	Time         types.Time `json:"time"`         // 时间
	IsBuyerMaker bool       `json:"isBuyerMaker"` // 是否买方挂单
	IsBestMatch  bool       `json:"isBestMatch"`  // 是否最佳匹配
}

// TradeStream 保存交易流数据
type TradeStream struct {
	EventType      string       `json:"e"` // 事件类型
	EventTime      types.Time   `json:"E"` // 事件时间
	Symbol         string       `json:"s"` // 交易对
	TradeID        int64        `json:"t"` // 交易ID
	Price          types.Number `json:"p"` // 价格
	Quantity       types.Number `json:"q"` // 数量
	BuyerOrderID   int64        `json:"b"` // 买方订单ID
	SellerOrderID  int64        `json:"a"` // 卖方订单ID
	TimeStamp      types.Time   `json:"T"` // 时间戳
	IsBuyerMaker   bool         `json:"m"` // 是否买方挂单
	BestMatchPrice bool         `json:"M"` // 最佳匹配价格
}

// KlineStream 保存K线流数据
type KlineStream struct {
	EventType string          `json:"e"` // 事件类型
	EventTime types.Time      `json:"E"` // 事件时间
	Symbol    string          `json:"s"` // 交易对
	Kline     KlineStreamData `json:"k"` // K线数据
}

// KlineStreamData 定义K线流数据
type KlineStreamData struct {
	StartTime                types.Time   `json:"t"` // 开始时间
	CloseTime                types.Time   `json:"T"` // 结束时间
	Symbol                   string       `json:"s"` // 交易对
	Interval                 string       `json:"i"` // 时间间隔
	FirstTradeID             int64        `json:"f"` // 第一个交易ID
	LastTradeID              int64        `json:"L"` // 最后交易ID
	OpenPrice                types.Number `json:"o"` // 开盘价
	ClosePrice               types.Number `json:"c"` // 收盘价
	HighPrice                types.Number `json:"h"` // 最高价
	LowPrice                 types.Number `json:"l"` // 最低价
	Volume                   types.Number `json:"v"` // 成交量
	NumberOfTrades           int64        `json:"n"` // 成交笔数
	KlineClosed              bool         `json:"x"` // K线是否关闭
	Quote                    types.Number `json:"q"` // 成交额
	TakerBuyBaseAssetVolume  types.Number `json:"V"` // 主动买入基础资产成交量
	TakerBuyQuoteAssetVolume types.Number `json:"Q"` // 主动买入计价资产成交量
}

// TickerStream 保存行情流数据
type TickerStream struct {
	EventType              string       `json:"e"` // 事件类型
	EventTime              types.Time   `json:"E"` // 事件时间
	Symbol                 string       `json:"s"` // 交易对
	PriceChange            types.Number `json:"p"` // 价格变化
	PriceChangePercent     types.Number `json:"P"` // 价格变化百分比
	WeightedAvgPrice       types.Number `json:"w"` // 加权平均价
	ClosePrice             types.Number `json:"x"` // 前收盘价
	LastPrice              types.Number `json:"c"` // 最新价格
	LastPriceQuantity      types.Number `json:"Q"` // 最新价格数量
	BestBidPrice           types.Number `json:"b"` // 最佳买价
	BestBidQuantity        types.Number `json:"B"` // 最佳买价数量
	BestAskPrice           types.Number `json:"a"` // 最佳卖价
	BestAskQuantity        types.Number `json:"A"` // 最佳卖价数量
	OpenPrice              types.Number `json:"o"` // 开盘价
	HighPrice              types.Number `json:"h"` // 最高价
	LowPrice               types.Number `json:"l"` // 最低价
	TotalTradedVolume      types.Number `json:"v"` // 总成交量
	TotalTradedQuoteVolume types.Number `json:"q"` // 总成交额
	OpenTime               types.Time   `json:"O"` // 开盘时间
	CloseTime              types.Time   `json:"C"` // 收盘时间
	FirstTradeID           int64        `json:"F"` // 第一个交易ID
	LastTradeID            int64        `json:"L"` // 最后交易ID
	NumberOfTrades         int64        `json:"n"` // 成交笔数
}

// HistoricalTrade 保存历史交易数据
type HistoricalTrade struct {
	ID            int64      `json:"id"`            // 交易ID
	Price         float64    `json:"price,string"`  // 价格
	Quantity      float64    `json:"qty,string"`    // 数量
	QuoteQuantity float64    `json:"quoteQty,string"` // 计价数量
	Time          types.Time `json:"time"`          // 时间
	IsBuyerMaker  bool       `json:"isBuyerMaker"`  // 是否买方挂单
	IsBestMatch   bool       `json:"isBestMatch"`   // 是否最佳匹配
}

// AggregatedTradeRequestParams 保存聚合交易请求参数
type AggregatedTradeRequestParams struct {
	Symbol currency.Pair // 必填字段；示例 LTCBTC, BTCUSDT
	// 要检索的第一个交易
	FromID int64
	// API似乎接受（开始和结束时间）或FromID，不接受其他组合
	StartTime time.Time
	EndTime   time.Time
	// 默认500；最大1000
	Limit int
}

// AggregatedTrade 保存聚合交易信息
type AggregatedTrade struct {
	ATradeID       int64      `json:"a"` // 聚合交易ID
	Price          float64    `json:"p,string"` // 价格
	Quantity       float64    `json:"q,string"` // 数量
	FirstTradeID   int64      `json:"f"` // 第一个交易ID
	LastTradeID    int64      `json:"l"` // 最后交易ID
	TimeStamp      types.Time `json:"T"` // 时间戳
	IsBuyerMaker   bool       `json:"m"` // 是否买方挂单
	BestMatchPrice bool       `json:"M"` // 最佳匹配价格
}

// IndexMarkPrice 存储指数和标记价格数据
type IndexMarkPrice struct {
	Symbol               string       `json:"symbol"`               // 交易对
	Pair                 string       `json:"pair"`                 // 交易对
	MarkPrice            types.Number `json:"markPrice"`            // 标记价格
	IndexPrice           types.Number `json:"indexPrice"`           // 指数价格
	EstimatedSettlePrice types.Number `json:"estimatedSettlePrice"` // 预估结算价格
	LastFundingRate      types.Number `json:"lastFundingRate"`      // 最后资金费率
	NextFundingTime      types.Time   `json:"nextFundingTime"`      // 下次资金费时间
	Time                 types.Time   `json:"time"`                 // 时间
}

// CandleStick 保存K线数据
type CandleStick struct {
	OpenTime                 types.Time   // 开盘时间
	Open                     types.Number // 开盘价
	High                     types.Number // 最高价
	Low                      types.Number // 最低价
	Close                    types.Number // 收盘价
	Volume                   types.Number // 成交量
	CloseTime                types.Time   // 收盘时间
	QuoteAssetVolume         types.Number // 计价资产成交量
	TradeCount               int64        // 成交笔数
	TakerBuyAssetVolume      types.Number // 主动买入基础资产成交量
	TakerBuyQuoteAssetVolume types.Number // 主动买入计价资产成交量
}

// UnmarshalJSON 将JSON数据解组到CandleStick结构体
func (c *CandleStick) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[11]any{
		&c.OpenTime,
		&c.Open,
		&c.High,
		&c.Low,
		&c.Close,
		&c.Volume,
		&c.CloseTime,
		&c.QuoteAssetVolume,
		&c.TradeCount,
		&c.TakerBuyAssetVolume,
		&c.TakerBuyQuoteAssetVolume,
	})
}

// AveragePrice 保存当前平均交易对价格
type AveragePrice struct {
	Mins  int64   `json:"mins"`         // 分钟数
	Price float64 `json:"price,string"` // 价格
}

// PriceChangeStats 包含最近24小时交易统计信息
type PriceChangeStats struct {
	Symbol             string       `json:"symbol"`             // 交易对
	PriceChange        types.Number `json:"priceChange"`        // 价格变化
	PriceChangePercent types.Number `json:"priceChangePercent"` // 价格变化百分比
	WeightedAvgPrice   types.Number `json:"weightedAvgPrice"`   // 加权平均价
	PrevClosePrice     types.Number `json:"prevClosePrice"`     // 前收盘价
	LastPrice          types.Number `json:"lastPrice"`          // 最新价格
	LastQty            types.Number `json:"lastQty"`            // 最新数量
	BidPrice           types.Number `json:"bidPrice"`           // 买价
	AskPrice           types.Number `json:"askPrice"`           // 卖价
	BidQuantity        types.Number `json:"bidQty"`             // 买量
	AskQuantity        types.Number `json:"askQty"`             // 卖量
	OpenPrice          types.Number `json:"openPrice"`          // 开盘价
	HighPrice          types.Number `json:"highPrice"`          // 最高价
	LowPrice           types.Number `json:"lowPrice"`           // 最低价
	Volume             types.Number `json:"volume"`             // 成交量
	QuoteVolume        types.Number `json:"quoteVolume"`        // 成交额
	OpenTime           types.Time   `json:"openTime"`           // 开盘时间
	CloseTime          types.Time   `json:"closeTime"`          // 收盘时间
	FirstID            int64        `json:"firstId"`            // 第一个交易ID
	LastID             int64        `json:"lastId"`             // 最后交易ID
	Count              int64        `json:"count"`              // 成交笔数
}

// SymbolPrice 保存基础交易对价格
type SymbolPrice struct {
	Symbol string  `json:"symbol"`       // 交易对
	Price  float64 `json:"price,string"` // 价格
}

// BestPrice 保存最佳价格数据
type BestPrice struct {
	Symbol   string  `json:"symbol"`         // 交易对
	BidPrice float64 `json:"bidPrice,string"` // 买价
	BidQty   float64 `json:"bidQty,string"`   // 买量
	AskPrice float64 `json:"askPrice,string"` // 卖价
	AskQty   float64 `json:"askQty,string"`   // 卖量
}

// NewOrderRequest 新订单请求类型
type NewOrderRequest struct {
	// Symbol 交易对（要交易的货币对）
	Symbol currency.Pair
	// Side 买入或卖出
	Side string
	// TradeType 交易类型（市价单或限价单）
	TradeType RequestParamsOrderType
	// TimeInForce 指定订单保持有效的时间
	// 示例：Good Till Cancel (GTC), Immediate or Cancel (IOC) 和 Fill Or Kill (FOK)
	TimeInForce string
	// Quantity 订单中花费或接收的基础数量总额
	Quantity float64
	// QuoteOrderQty 市价单中花费或接收的计价数量总额
	QuoteOrderQty    float64
	Price            float64 // 价格
	NewClientOrderID string  // 新客户端订单ID
	StopPrice        float64 // 用于STOP_LOSS, STOP_LOSS_LIMIT, TAKE_PROFIT, 和 TAKE_PROFIT_LIMIT订单
	IcebergQty       float64 // 用于LIMIT, STOP_LOSS_LIMIT, 和 TAKE_PROFIT_LIMIT创建冰山订单
	NewOrderRespType string  // 新订单响应类型
}

// NewOrderResponse 交易所返回的结构化响应
type NewOrderResponse struct {
	Code            int        `json:"code"`            // 状态码
	Msg             string     `json:"msg"`             // 消息
	Symbol          string     `json:"symbol"`          // 交易对
	OrderID         int64      `json:"orderId"`         // 订单ID
	ClientOrderID   string     `json:"clientOrderId"`   // 客户端订单ID
	TransactionTime types.Time `json:"transactTime"`    // 交易时间
	Price           float64    `json:"price,string"`    // 价格
	OrigQty         float64    `json:"origQty,string"`  // 原始数量
	ExecutedQty     float64    `json:"executedQty,string"` // 已执行数量
	// 已花费（买单）或已收到（卖单）的计价资产累计金额
	CumulativeQuoteQty float64 `json:"cummulativeQuoteQty,string"` // 累计计价数量
	Status             string  `json:"status"`      // 状态
	TimeInForce        string  `json:"timeInForce"` // 有效时间
	Type               string  `json:"type"`        // 类型
	Side               string  `json:"side"`        // 买卖方向
	Fills              []struct {
		Price           float64 `json:"price,string"`      // 价格
		Qty             float64 `json:"qty,string"`        // 数量
		Commission      float64 `json:"commission,string"` // 手续费
		CommissionAsset string  `json:"commissionAsset"`   // 手续费资产
	} `json:"fills"` // 成交明细
}

// CancelOrderResponse 交易所返回的取消订单结构化响应
type CancelOrderResponse struct {
	Symbol            string `json:"symbol"`            // 交易对
	OrigClientOrderID string `json:"origClientOrderId"` // 原始客户端订单ID
	OrderID           int64  `json:"orderId"`           // 订单ID
	ClientOrderID     string `json:"clientOrderId"`     // 客户端订单ID
}

// QueryOrderData 保存查询订单数据
type QueryOrderData struct {
	Code                int        `json:"code"`                // 状态码
	Msg                 string     `json:"msg"`                 // 消息
	Symbol              string     `json:"symbol"`              // 交易对
	OrderID             int64      `json:"orderId"`             // 订单ID
	ClientOrderID       string     `json:"clientOrderId"`       // 客户端订单ID
	Price               float64    `json:"price,string"`        // 价格
	OrigQty             float64    `json:"origQty,string"`      // 原始数量
	ExecutedQty         float64    `json:"executedQty,string"`  // 已执行数量
	Status              string     `json:"status"`              // 状态
	TimeInForce         string     `json:"timeInForce"`         // 有效时间
	Type                string     `json:"type"`                // 类型
	Side                string     `json:"side"`                // 买卖方向
	StopPrice           float64    `json:"stopPrice,string"`    // 止损价格
	IcebergQty          float64    `json:"icebergQty,string"`   // 冰山数量
	Time                types.Time `json:"time"`                // 时间
	IsWorking           bool       `json:"isWorking"`           // 是否工作中
	CummulativeQuoteQty float64    `json:"cummulativeQuoteQty,string"` // 累计计价数量
	OrderListID         int64      `json:"orderListId"`         // 订单列表ID
	OrigQuoteOrderQty   float64    `json:"origQuoteOrderQty,string"` // 原始计价订单数量
	UpdateTime          types.Time `json:"updateTime"`          // 更新时间
}

// Balance 保存余额数据
type Balance struct {
	Asset  string          `json:"asset"`  // 资产
	Free   decimal.Decimal `json:"free"`   // 可用余额
	Locked decimal.Decimal `json:"locked"` // 锁定余额
}

// Account 保存账户数据
type Account struct {
	MakerCommission  int        `json:"makerCommission"`  // 挂单手续费
	TakerCommission  int        `json:"takerCommission"`  // 吃单手续费
	BuyerCommission  int        `json:"buyerCommission"`  // 买方手续费
	SellerCommission int        `json:"sellerCommission"` // 卖方手续费
	CanTrade         bool       `json:"canTrade"`         // 可交易
	CanWithdraw      bool       `json:"canWithdraw"`      // 可提现
	CanDeposit       bool       `json:"canDeposit"`       // 可充值
	UpdateTime       types.Time `json:"updateTime"`       // 更新时间
	Balances         []Balance  `json:"balances"`         // 余额列表
}

// MarginAccount 保存保证金账户数据
type MarginAccount struct {
	BorrowEnabled       bool                 `json:"borrowEnabled"`       // 借贷启用
	MarginLevel         float64              `json:"marginLevel,string"`  // 保证金水平
	TotalAssetOfBtc     float64              `json:"totalAssetOfBtc,string"` // BTC总资产
	TotalLiabilityOfBtc float64              `json:"totalLiabilityOfBtc,string"` // BTC总负债
	TotalNetAssetOfBtc  float64              `json:"totalNetAssetOfBtc,string"` // BTC净资产总额
	TradeEnabled        bool                 `json:"tradeEnabled"`        // 交易启用
	TransferEnabled     bool                 `json:"transferEnabled"`     // 转账启用
	UserAssets          []MarginAccountAsset `json:"userAssets"`          // 用户资产
}

// MarginAccountAsset 保存每个单独的保证金账户资产
type MarginAccountAsset struct {
	Asset    string  `json:"asset"`           // 资产
	Borrowed float64 `json:"borrowed,string"` // 借贷
	Free     float64 `json:"free,string"`     // 可用
	Interest float64 `json:"interest,string"` // 利息
	Locked   float64 `json:"locked,string"`   // 锁定
	NetAsset float64 `json:"netAsset,string"` // 净资产
}

// RequestParamsOrderType 交易订单类型
type RequestParamsOrderType string

var (
	// BinanceRequestParamsOrderLimit 限价单
	BinanceRequestParamsOrderLimit = RequestParamsOrderType("LIMIT")

	// BinanceRequestParamsOrderMarket 市价单
	BinanceRequestParamsOrderMarket = RequestParamsOrderType("MARKET")

	// BinanceRequestParamsOrderStopLoss 止损单
	BinanceRequestParamsOrderStopLoss = RequestParamsOrderType("STOP_LOSS")

	// BinanceRequestParamsOrderStopLossLimit 止损限价单
	BinanceRequestParamsOrderStopLossLimit = RequestParamsOrderType("STOP_LOSS_LIMIT")

	// BinanceRequestParamsOrderTakeProfit 止盈单
	BinanceRequestParamsOrderTakeProfit = RequestParamsOrderType("TAKE_PROFIT")

	// BinanceRequestParamsOrderTakeProfitLimit 止盈限价单
	BinanceRequestParamsOrderTakeProfitLimit = RequestParamsOrderType("TAKE_PROFIT_LIMIT")

	// BinanceRequestParamsOrderLimitMarker 限价挂单
	BinanceRequestParamsOrderLimitMarker = RequestParamsOrderType("LIMIT_MAKER")
)

// KlinesRequestParams K线请求数据
type KlinesRequestParams struct {
	Symbol    currency.Pair // 必填字段；示例 LTCBTC, BTCUSDT
	Interval  string        // 时间间隔周期
	Limit     uint64        // 默认500；最大500
	StartTime time.Time     // 开始时间
	EndTime   time.Time     // 结束时间
}

// DepositHistory 存储充值历史信息
type DepositHistory struct {
	Amount        float64    `json:"amount,string"`    // 金额
	Coin          string     `json:"coin"`             // 币种
	Network       string     `json:"network"`          // 网络
	Status        uint8      `json:"status"`           // 状态
	Address       string     `json:"address"`          // 地址
	AddressTag    string     `json:"adressTag"`        // 地址标签
	TransactionID string     `json:"txId"`             // 交易ID
	InsertTime    types.Time `json:"insertTime"`       // 插入时间
	TransferType  uint8      `json:"transferType"`     // 转账类型
	ConfirmTimes  string     `json:"confirmTimes"`     // 确认次数
}

// WithdrawResponse 包含提现请求状态
type WithdrawResponse struct {
	ID string `json:"id"` // ID
}

// WithdrawStatusResponse 定义提现状态响应
type WithdrawStatusResponse struct {
	Address         string     `json:"address"`          // 地址
	Amount          float64    `json:"amount,string"`    // 金额
	ApplyTime       types.Time `json:"applyTime"`        // 申请时间
	Coin            string     `json:"coin"`             // 币种
	ID              string     `json:"id"`               // ID
	WithdrawOrderID string     `json:"withdrawOrderId"`  // 提现订单ID
	Network         string     `json:"network"`          // 网络
	TransferType    uint8      `json:"transferType"`     // 转账类型
	Status          int64      `json:"status"`           // 状态
	TransactionFee  float64    `json:"transactionFee,string"` // 交易手续费
	TransactionID   string     `json:"txId"`             // 交易ID
	ConfirmNumber   int64      `json:"confirmNo"`        // 确认数量
}

// DepositAddress 存储充值地址信息
type DepositAddress struct {
	Address string `json:"address"` // 地址
	Coin    string `json:"coin"`    // 币种
	Tag     string `json:"tag"`     // 标签
	URL     string `json:"url"`     // URL
}

// UserAccountStream 包含维护授权WebSocket连接的密钥
type UserAccountStream struct {
	ListenKey string `json:"listenKey"` // 监听密钥
}

// WsAccountInfoData 定义WebSocket账户信息数据
type WsAccountInfoData struct {
	CanDeposit       bool      `json:"D"` // 可充值
	CanTrade         bool      `json:"T"` // 可交易
	CanWithdraw      bool      `json:"W"` // 可提现
	EventTime        time.Time `json:"E"` // 事件时间
	LastUpdated      time.Time `json:"u"` // 最后更新
	BuyerCommission  float64   `json:"b"` // 买方手续费
	MakerCommission  float64   `json:"m"` // 挂单手续费
	SellerCommission float64   `json:"s"` // 卖方手续费
	TakerCommission  float64   `json:"t"` // 吃单手续费
	EventType        string    `json:"e"` // 事件类型
	Currencies       []struct {
		Asset     string  `json:"a"`        // 资产
		Available float64 `json:"f,string"` // 可用
		Locked    float64 `json:"l,string"` // 锁定
	} `json:"B"` // 货币列表
}

// WsAccountPositionData 定义WebSocket账户持仓数据
type WsAccountPositionData struct {
	Currencies []struct {
		Asset     string  `json:"a"`        // 资产
		Available float64 `json:"f,string"` // 可用
		Locked    float64 `json:"l,string"` // 锁定
	} `json:"B"`                   // 货币列表
	EventTime   types.Time `json:"E"` // 事件时间
	LastUpdated types.Time `json:"u"` // 最后更新
	EventType   string     `json:"e"` // 事件类型
}

// WsBalanceUpdateData 定义WebSocket账户余额数据
type WsBalanceUpdateData struct {
	EventTime    types.Time `json:"E"`        // 事件时间
	ClearTime    types.Time `json:"T"`        // 清算时间
	BalanceDelta float64    `json:"d,string"` // 余额变化
	Asset        string     `json:"a"`        // 资产
	EventType    string     `json:"e"`        // 事件类型
}

// WsOrderUpdateData 定义WebSocket账户订单更新数据
type WsOrderUpdateData struct {
	EventType                         string     `json:"e"` // 事件类型
	EventTime                         types.Time `json:"E"` // 事件时间
	Symbol                            string     `json:"s"` // 交易对
	ClientOrderID                     string     `json:"c"` // 客户端订单ID
	Side                              string     `json:"S"` // 买卖方向
	OrderType                         string     `json:"o"` // 订单类型
	TimeInForce                       string     `json:"f"` // 有效时间
	Quantity                          float64    `json:"q,string"` // 数量
	Price                             float64    `json:"p,string"` // 价格
	StopPrice                         float64    `json:"P,string"` // 止损价格
	IcebergQuantity                   float64    `json:"F,string"` // 冰山数量
	OrderListID                       int64      `json:"g"` // 订单列表ID
	CancelledClientOrderID            string     `json:"C"` // 取消的客户端订单ID
	CurrentExecutionType              string     `json:"x"` // 当前执行类型
	OrderStatus                       string     `json:"X"` // 订单状态
	RejectionReason                   string     `json:"r"` // 拒绝原因
	OrderID                           int64      `json:"i"` // 订单ID
	LastExecutedQuantity              float64    `json:"l,string"` // 最后执行数量
	CumulativeFilledQuantity          float64    `json:"z,string"` // 累计成交数量
	LastExecutedPrice                 float64    `json:"L,string"` // 最后执行价格
	Commission                        float64    `json:"n,string"` // 手续费
	CommissionAsset                   string     `json:"N"` // 手续费资产
	TransactionTime                   types.Time `json:"T"` // 交易时间
	TradeID                           int64      `json:"t"` // 交易ID
	Ignored                           int64      `json:"I"` // 必须明确忽略，否则会覆盖'i'
	IsOnOrderBook                     bool       `json:"w"` // 是否在订单簿上
	IsMaker                           bool       `json:"m"` // 是否挂单方
	Ignored2                          bool       `json:"M"` // 参见"I"的注释
	OrderCreationTime                 types.Time `json:"O"` // 订单创建时间
	WorkingTime                       types.Time `json:"W"` // 工作时间
	CumulativeQuoteTransactedQuantity float64    `json:"Z,string"` // 累计计价成交数量
	LastQuoteAssetTransactedQuantity  float64    `json:"Y,string"` // 最后计价资产成交数量
	QuoteOrderQuantity                float64    `json:"Q,string"` // 计价订单数量
}

// WsListStatusData 定义WebSocket账户列表状态数据
type WsListStatusData struct {
	ListClientOrderID string     `json:"C"` // 列表客户端订单ID
	EventTime         types.Time `json:"E"` // 事件时间
	ListOrderStatus   string     `json:"L"` // 列表订单状态
	Orders            []struct {
		ClientOrderID string `json:"c"` // 客户端订单ID
		OrderID       int64  `json:"i"` // 订单ID
		Symbol        string `json:"s"` // 交易对
	} `json:"O"` // 订单列表
	TransactionTime types.Time `json:"T"` // 交易时间
	ContingencyType string     `json:"c"` // 条件类型
	EventType       string     `json:"e"` // 事件类型
	OrderListID     int64      `json:"g"` // 订单列表ID
	ListStatusType  string     `json:"l"` // 列表状态类型
	RejectionReason string     `json:"r"` // 拒绝原因
	Symbol          string     `json:"s"` // 交易对
}

// WsPayload 定义通过WebSocket连接的负载
type WsPayload struct {
	Method string   `json:"method"` // 方法
	Params []string `json:"params"` // 参数
	ID     int64    `json:"id"`     // ID
}

// CrossMarginInterestData 存储借贷的全仓保证金数据
type CrossMarginInterestData struct {
	Code          int64  `json:"code,string"`    // 状态码
	Message       string `json:"message"`        // 消息
	MessageDetail string `json:"messageDetail"`  // 消息详情
	Data          []struct {
		AssetName string `json:"assetName"` // 资产名称
		Specs     []struct {
			VipLevel          string `json:"vipLevel"`          // VIP等级
			DailyInterestRate string `json:"dailyInterestRate"` // 日利率
			BorrowLimit       string `json:"borrowLimit"`       // 借贷限额
		} `json:"specs"` // 规格
	} `json:"data"`    // 数据
	Success bool `json:"success"` // 成功标志
}

// UserMarginInterestHistoryResponse 用户保证金利息历史响应
type UserMarginInterestHistoryResponse struct {
	Rows  []UserMarginInterestHistory `json:"rows"`  // 行数据
	Total int64                       `json:"total"` // 总数
}

// UserMarginInterestHistory 用户保证金利息历史行
type UserMarginInterestHistory struct {
	TxID                int64      `json:"txId"`                // 交易ID
	InterestAccruedTime types.Time `json:"interestAccuredTime"` // 利息累计时间（文档中的拼写错误，由于API限制无法验证）
	Asset               string     `json:"asset"`               // 资产
	RawAsset            string     `json:"rawAsset"`
	Principal           float64    `json:"principal,string"`
	Interest            float64    `json:"interest,string"`
	InterestRate        float64    `json:"interestRate,string"`
	Type                string     `json:"type"`
	IsolatedSymbol      string     `json:"isolatedSymbol"`
}

// CryptoLoansIncomeHistory 存储加密货币借贷收入历史数据
type CryptoLoansIncomeHistory struct {
	Asset         currency.Code `json:"asset"`  // 资产
	Type          string        `json:"type"`   // 类型
	Amount        float64       `json:"amount,string"` // 金额
	TransactionID int64         `json:"tranId"` // 交易ID
}

// CryptoLoanBorrow 存储加密货币借贷数据
type CryptoLoanBorrow struct {
	LoanCoin           currency.Code `json:"loanCoin"`           // 借贷币种
	Amount             float64       `json:"amount,string"`      // 金额
	CollateralCoin     currency.Code `json:"collateralCoin"`     // 抵押币种
	CollateralAmount   float64       `json:"collateralAmount,string"` // 抵押金额
	HourlyInterestRate float64       `json:"hourlyInterestRate,string"` // 小时利率
	OrderID            int64         `json:"orderId,string"`     // 订单ID
}

// LoanBorrowHistoryItem 存储借贷历史项目数据
type LoanBorrowHistoryItem struct {
	OrderID                 int64         `json:"orderId"`                 // 订单ID
	LoanCoin                currency.Code `json:"loanCoin"`                // 借贷币种
	InitialLoanAmount       float64       `json:"initialLoanAmount,string"` // 初始借贷金额
	HourlyInterestRate      float64       `json:"hourlyInterestRate,string"` // 小时利率
	LoanTerm                int64         `json:"loanTerm,string"`         // 借贷期限
	CollateralCoin          currency.Code `json:"collateralCoin"`          // 抵押币种
	InitialCollateralAmount float64       `json:"initialCollateralAmount,string"` // 初始抵押金额
	BorrowTime              types.Time    `json:"borrowTime"`              // 借贷时间
	Status                  string        `json:"status"`                  // 状态
}

// LoanBorrowHistory 存储借贷历史数据
type LoanBorrowHistory struct {
	Rows  []LoanBorrowHistoryItem `json:"rows"`  // 行数据
	Total int64                   `json:"total"` // 总数
}

// CryptoLoanOngoingOrderItem 存储加密货币借贷进行中订单项目数据
type CryptoLoanOngoingOrderItem struct {
	OrderID          int64         `json:"orderId"`          // 订单ID
	LoanCoin         currency.Code `json:"loanCoin"`         // 借贷币种
	TotalDebt        float64       `json:"totalDebt,string"` // 总债务
	ResidualInterest float64       `json:"residualInterest,string"` // 剩余利息
	CollateralCoin   currency.Code `json:"collateralCoin"`   // 抵押币种
	CollateralAmount float64       `json:"collateralAmount,string"` // 抵押金额
	CurrentLTV       float64       `json:"currentLTV,string"` // 当前LTV
	ExpirationTime   types.Time    `json:"expirationTime"`   // 到期时间
}

// CryptoLoanOngoingOrder 存储加密货币借贷进行中订单数据
type CryptoLoanOngoingOrder struct {
	Rows  []CryptoLoanOngoingOrderItem `json:"rows"`  // 行数据
	Total int64                        `json:"total"` // 总数
}

// CryptoLoanRepay 存储加密货币借贷还款数据
type CryptoLoanRepay struct {
	LoanCoin            currency.Code `json:"loanCoin"`            // 借贷币种
	RemainingPrincipal  float64       `json:"remainingPrincipal,string"` // 剩余本金
	RemainingInterest   float64       `json:"remainingInterest,string"` // 剩余利息
	CollateralCoin      currency.Code `json:"collateralCoin"`      // 抵押币种
	RemainingCollateral float64       `json:"remainingCollateral,string"` // 剩余抵押
	CurrentLTV          float64       `json:"currentLTV,string"`   // 当前LTV
	RepayStatus         string        `json:"repayStatus"`         // 还款状态
}

// CryptoLoanRepayHistoryItem 存储加密货币借贷还款历史项目数据
type CryptoLoanRepayHistoryItem struct {
	LoanCoin         currency.Code `json:"loanCoin"`         // 借贷币种
	RepayAmount      float64       `json:"repayAmount,string"` // 还款金额
	CollateralCoin   currency.Code `json:"collateralCoin"`   // 抵押币种
	CollateralUsed   float64       `json:"collateralUsed,string"` // 使用的抵押
	CollateralReturn float64       `json:"collateralReturn,string"` // 返还的抵押
	RepayType        string        `json:"repayType"`        // 还款类型
	RepayTime        types.Time    `json:"repayTime"`        // 还款时间
	OrderID          int64         `json:"orderId"`          // 订单ID
}

// CryptoLoanRepayHistory 存储加密货币借贷还款历史数据
type CryptoLoanRepayHistory struct {
	Rows  []CryptoLoanRepayHistoryItem `json:"rows"`  // 行数据
	Total int64                        `json:"total"` // 总数
}

// CryptoLoanAdjustLTV 存储加密货币借贷LTV调整数据
type CryptoLoanAdjustLTV struct {
	LoanCoin       currency.Code `json:"loanCoin"`       // 借贷币种
	CollateralCoin currency.Code `json:"collateralCoin"` // 抵押币种
	Direction      string        `json:"direction"`      // 方向
	Amount         float64       `json:"amount,string"`  // 金额
	CurrentLTV     float64       `json:"currentLTV,string"` // 当前LTV
}

// CryptoLoanLTVAdjustmentItem 存储加密货币借贷LTV调整项目数据
type CryptoLoanLTVAdjustmentItem struct {
	LoanCoin       currency.Code `json:"loanCoin"`       // 借贷币种
	CollateralCoin currency.Code `json:"collateralCoin"` // 抵押币种
	Direction      string        `json:"direction"`      // 方向
	Amount         float64       `json:"amount,string"`  // 金额
	PreviousLTV    float64       `json:"preLTV,string"`  // 之前LTV
	AfterLTV       float64       `json:"afterLTV,string"` // 之后LTV
	AdjustTime     types.Time    `json:"adjustTime"`     // 调整时间
	OrderID        int64         `json:"orderId"`        // 订单ID
}

// CryptoLoanLTVAdjustmentHistory 存储加密货币借贷LTV调整历史数据
type CryptoLoanLTVAdjustmentHistory struct {
	Rows  []CryptoLoanLTVAdjustmentItem `json:"rows"`  // 行数据
	Total int64                         `json:"total"` // 总数
}

// LoanableAssetItem 存储可借贷资产项目数据
type LoanableAssetItem struct {
	LoanCoin                             currency.Code `json:"loanCoin"`                     // 借贷币种
	SevenDayHourlyInterestRate           float64       `json:"_7dHourlyInterestRate,string"` // 7天小时利率
	SevenDayDailyInterestRate            float64       `json:"_7dDailyInterestRate,string"`  // 7天日利率
	FourteenDayHourlyInterest            float64       `json:"_14dHourlyInterestRate,string"` // 14天小时利率
	FourteenDayDailyInterest             float64       `json:"_14dDailyInterestRate,string"`  // 14天日利率
	ThirtyDayHourlyInterest              float64       `json:"_30dHourlyInterestRate,string"` // 30天小时利率
	ThirtyDayDailyInterest               float64       `json:"_30dDailyInterestRate,string"`  // 30天日利率
	NinetyDayHourlyInterest              float64       `json:"_90dHourlyInterestRate,string"` // 90天小时利率
	NinetyDayDailyInterest               float64       `json:"_90dDailyInterestRate,string"`  // 90天日利率
	OneHundredAndEightyDayHourlyInterest float64       `json:"_180dHourlyInterestRate,string"` // 180天小时利率
	OneHundredAndEightyDayDailyInterest  float64       `json:"_180dDailyInterestRate,string"`  // 180天日利率
	MinimumLimit                         float64       `json:"minLimit,string"`               // 最小限额
	MaximumLimit                         float64       `json:"maxLimit,string"`               // 最大限额
	VIPLevel                             int64         `json:"vipLevel"`                      // VIP等级
}

// LoanableAssetsData 存储可借贷资产数据
type LoanableAssetsData struct {
	Rows  []LoanableAssetItem `json:"rows"`  // 行数据
	Total int64               `json:"total"` // 总数
}

// CollateralAssetItem 存储抵押资产项目数据
type CollateralAssetItem struct {
	CollateralCoin currency.Code `json:"collateralCoin"`        // 抵押币种
	InitialLTV     float64       `json:"initialLTV,string"`     // 初始LTV
	MarginCallLTV  float64       `json:"marginCallLTV,string"`  // 保证金追缴LTV
	LiquidationLTV float64       `json:"liquidationLTV,string"` // 清算LTV
	MaxLimit       float64       `json:"maxLimit,string"`       // 最大限额
	VIPLevel       int64         `json:"vipLevel"`              // VIP等级
}

// CollateralAssetData 存储抵押资产数据
type CollateralAssetData struct {
	Rows  []CollateralAssetItem `json:"rows"`  // 行数据
	Total int64                 `json:"total"` // 总数
}

// CollateralRepayRate 存储抵押还款利率数据
type CollateralRepayRate struct {
	LoanCoin       currency.Code `json:"loanCoin"`       // 借贷币种
	CollateralCoin currency.Code `json:"collateralCoin"` // 抵押币种
	RepayAmount    float64       `json:"repayAmount,string"` // 还款金额
	Rate           float64       `json:"rate,string"`    // 利率
}

// CustomiseMarginCallItem 存储自定义保证金追缴项目数据
type CustomiseMarginCallItem struct {
	OrderID         int64         `json:"orderId"`         // 订单ID
	CollateralCoin  currency.Code `json:"collateralCoin"`  // 抵押币种
	PreMarginCall   float64       `json:"preMarginCall,string"` // 之前保证金追缴
	AfterMarginCall float64       `json:"afterMarginCall,string"` // 之后保证金追缴
	CustomiseTime   types.Time    `json:"customizeTime"`   // 自定义时间
}

// CustomiseMarginCall 存储自定义保证金追缴数据
type CustomiseMarginCall struct {
	Rows  []CustomiseMarginCallItem `json:"rows"`  // 行数据
	Total int64                     `json:"total"` // 总数
}

// FlexibleLoanBorrow 存储灵活借贷
type FlexibleLoanBorrow struct {
	LoanCoin         currency.Code `json:"loanCoin"`         // 借贷币种
	LoanAmount       float64       `json:"loanAmount,string"` // 借贷金额
	CollateralCoin   currency.Code `json:"collateralCoin"`   // 抵押币种
	CollateralAmount float64       `json:"collateralAmount,string"` // 抵押金额
	Status           string        `json:"status"`           // 状态
}

// FlexibleLoanOngoingOrderItem 存储灵活借贷进行中订单项目
type FlexibleLoanOngoingOrderItem struct {
	LoanCoin         currency.Code `json:"loanCoin"`         // 借贷币种
	TotalDebt        float64       `json:"totalDebt,string"` // 总债务
	CollateralCoin   currency.Code `json:"collateralCoin"`   // 抵押币种
	CollateralAmount float64       `json:"collateralAmount,string"` // 抵押金额
	CurrentLTV       float64       `json:"currentLTV,string"` // 当前LTV
}

// FlexibleLoanOngoingOrder 存储灵活借贷进行中订单
type FlexibleLoanOngoingOrder struct {
	Rows  []FlexibleLoanOngoingOrderItem `json:"rows"`  // 行数据
	Total int64                          `json:"total"` // 总数
}

// FlexibleLoanBorrowHistoryItem 存储灵活借贷历史项目
type FlexibleLoanBorrowHistoryItem struct {
	LoanCoin                currency.Code `json:"loanCoin"`                // 借贷币种
	InitialLoanAmount       float64       `json:"initialLoanAmount,string"` // 初始借贷金额
	CollateralCoin          currency.Code `json:"collateralCoin"`          // 抵押币种
	InitialCollateralAmount float64       `json:"initialCollateralAmount,string"` // 初始抵押金额
	BorrowTime              types.Time    `json:"borrowTime"`              // 借贷时间
	Status                  string        `json:"status"`                  // 状态
}

// FlexibleLoanBorrowHistory 存储灵活借贷历史
type FlexibleLoanBorrowHistory struct {
	Rows  []FlexibleLoanBorrowHistoryItem `json:"rows"`  // 行数据
	Total int64                           `json:"total"` // 总数
}

// FlexibleLoanRepay 存储灵活借贷还款
type FlexibleLoanRepay struct {
	LoanCoin            currency.Code `json:"loanCoin"`            // 借贷币种
	CollateralCoin      currency.Code `json:"collateralCoin"`      // 抵押币种
	RemainingDebt       float64       `json:"remainingDebt,string"` // 剩余债务
	RemainingCollateral float64       `json:"remainingCollateral,string"` // 剩余抵押
	FullRepayment       bool          `json:"fullRepayment"`       // 全额还款
	CurrentLTV          float64       `json:"currentLTV,string"`   // 当前LTV
	RepayStatus         string        `json:"repayStatus"`         // 还款状态
}

// FlexibleLoanRepayHistoryItem 存储灵活借贷还款历史项目
type FlexibleLoanRepayHistoryItem struct {
	LoanCoin         currency.Code `json:"loanCoin"`         // 借贷币种
	RepayAmount      float64       `json:"repayAmount,string"` // 还款金额
	CollateralCoin   currency.Code `json:"collateralCoin"`   // 抵押币种
	CollateralReturn float64       `json:"collateralReturn,string"` // 抵押返还
	RepayStatus      string        `json:"repayStatus"`      // 还款状态
	RepayTime        types.Time    `json:"repayTime"`        // 还款时间
}

// FlexibleLoanRepayHistory 存储灵活借贷还款历史
type FlexibleLoanRepayHistory struct {
	Rows  []FlexibleLoanRepayHistoryItem `json:"rows"`  // 行数据
	Total int64                          `json:"total"` // 总数
}

// FlexibleLoanAdjustLTV 存储灵活借贷LTV调整
type FlexibleLoanAdjustLTV struct {
	LoanCoin       currency.Code `json:"loanCoin"`       // 借贷币种
	CollateralCoin currency.Code `json:"collateralCoin"` // 抵押币种
	Direction      string        `json:"direction"`      // 方向
	Amount         float64       `json:"amount,string"`  // 金额（文档错误：API实际返回"amount"而不是"adjustedAmount"）
	CurrentLTV     float64       `json:"currentLTV,string"` // 当前LTV
	Status         string        `json:"status"`         // 状态
}

// FlexibleLoanLTVAdjustmentHistoryItem 存储灵活借贷LTV调整历史项目
type FlexibleLoanLTVAdjustmentHistoryItem struct {
	LoanCoin         currency.Code `json:"loanCoin"`         // 借贷币种
	CollateralCoin   currency.Code `json:"collateralCoin"`   // 抵押币种
	Direction        string        `json:"direction"`        // 方向
	CollateralAmount float64       `json:"collateralAmount,string"` // 抵押金额
	PreviousLTV      float64       `json:"preLTV,string"`    // 之前LTV
	AfterLTV         float64       `json:"afterLTV,string"`  // 之后LTV
	AdjustTime       types.Time    `json:"adjustTime"`       // 调整时间
}

// FlexibleLoanLTVAdjustmentHistory 存储灵活借贷LTV调整历史
type FlexibleLoanLTVAdjustmentHistory struct {
	Rows  []FlexibleLoanLTVAdjustmentHistoryItem `json:"rows"`  // 行数据
	Total int64                                  `json:"total"` // 总数
}

// FlexibleLoanAssetsDataItem 存储灵活借贷资产数据项目
type FlexibleLoanAssetsDataItem struct {
	LoanCoin             currency.Code `json:"loanCoin"`             // 借贷币种
	FlexibleInterestRate float64       `json:"flexibleInterestRate,string"` // 灵活利率
	FlexibleMinLimit     float64       `json:"flexibleMinLimit,string"` // 灵活最小限额
	FlexibleMaxLimit     float64       `json:"flexibleMaxLimit,string"` // 灵活最大限额
}

// FlexibleLoanAssetsData 存储灵活借贷资产数据
type FlexibleLoanAssetsData struct {
	Rows  []FlexibleLoanAssetsDataItem `json:"rows"`  // 行数据
	Total int64                        `json:"total"` // 总数
}

// FlexibleCollateralAssetsDataItem 存储灵活抵押资产数据项
type FlexibleCollateralAssetsDataItem struct {
	CollateralCoin currency.Code `json:"collateralCoin"`        // 抵押币种
	InitialLTV     float64       `json:"initialLTV,string"`     // 初始LTV
	MarginCallLTV  float64       `json:"marginCallLTV,string"`  // 保证金追缴LTV
	LiquidationLTV float64       `json:"liquidationLTV,string"` // 清算LTV
	MaxLimit       float64       `json:"maxLimit,string"`       // 最大限额
}

// FlexibleCollateralAssetsData 存储灵活抵押资产数据
type FlexibleCollateralAssetsData struct {
	Rows  []FlexibleCollateralAssetsDataItem `json:"rows"`  // 行数据
	Total int64                              `json:"total"` // 总数
}
