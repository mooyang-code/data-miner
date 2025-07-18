package app

import (
	"fmt"
	"go.uber.org/zap"

	"github.com/mooyang-code/data-miner/internal/exchanges/binance"
	"github.com/mooyang-code/data-miner/internal/types"
)

// WebsocketManager WebSocket管理器
type WebsocketManager struct {
	logger *zap.Logger
}

// NewWebsocketManager 创建新的WebSocket管理器
func NewWebsocketManager(logger *zap.Logger) *WebsocketManager {
	return &WebsocketManager{
		logger: logger,
	}
}

// Start 启动WebSocket连接
func (wm *WebsocketManager) Start(config *types.Config, exchanges map[string]types.ExchangeInterface) error {
	// 启动Binance WebSocket（如果启用）
	if config.Exchanges.Binance.Enabled && config.Exchanges.Binance.UseWebsocket {
		if binanceExchange, ok := exchanges["binance"].(*binance.Binance); ok {
			wm.logger.Info("启动Binance WebSocket模式")
			if err := wm.startBinanceWebsocket(binanceExchange, config.Exchanges.Binance); err != nil {
				wm.logger.Error("启动Binance WebSocket失败", zap.Error(err))
				return err
			}
		}
	}

	// 这里可以添加其他交易所的WebSocket启动逻辑

	return nil
}

// startBinanceWebsocket 启动Binance WebSocket连接
func (wm *WebsocketManager) startBinanceWebsocket(exchange *binance.Binance, config types.BinanceConfig) error {
	// 连接WebSocket
	wm.logger.Info("正在连接Binance WebSocket...")
	if err := exchange.WsConnect(); err != nil {
		return fmt.Errorf("连接WebSocket失败: %v", err)
	}
	wm.logger.Info("WebSocket连接成功")

	// 使用封装好的订阅方法
	if err := wm.subscribeToDataTypes(exchange, config); err != nil {
		wm.logger.Error("订阅数据类型失败", zap.Error(err))
		return err
	}

	wm.logger.Info("所有数据类型订阅成功",
		zap.Int("订阅数量", exchange.GetSubscriptionCount()),
		zap.Strings("活跃订阅", exchange.GetActiveSubscriptions()))

	return nil
}

// subscribeToDataTypes 使用封装好的方法订阅各种数据类型
func (wm *WebsocketManager) subscribeToDataTypes(exchange *binance.Binance, config types.BinanceConfig) error {
	// 订阅行情数据
	if config.DataTypes.Ticker.Enabled && len(config.DataTypes.Ticker.Symbols) > 0 {
		symbols := wm.convertToSymbolTypes(config.DataTypes.Ticker.Symbols)
		wm.logger.Info("订阅行情数据", zap.Strings("symbols", config.DataTypes.Ticker.Symbols))

		if err := exchange.SubscribeTicker(symbols, wm.createTickerCallback()); err != nil {
			return fmt.Errorf("订阅行情数据失败: %v", err)
		}
	}

	// 订阅订单簿数据
	if config.DataTypes.Orderbook.Enabled && len(config.DataTypes.Orderbook.Symbols) > 0 {
		symbols := wm.convertToSymbolTypes(config.DataTypes.Orderbook.Symbols)
		wm.logger.Info("订阅订单簿数据",
			zap.Strings("symbols", config.DataTypes.Orderbook.Symbols),
			zap.Int("depth", config.DataTypes.Orderbook.Depth))

		// 使用自定义深度订阅
		if err := exchange.SubscribeOrderbookWithDepth(symbols, config.DataTypes.Orderbook.Depth, "100ms", wm.createOrderbookCallback()); err != nil {
			return fmt.Errorf("订阅订单簿数据失败: %v", err)
		}
	}

	// 订阅K线数据
	if config.DataTypes.Klines.Enabled && len(config.DataTypes.Klines.Symbols) > 0 {
		symbols := wm.convertToSymbolTypes(config.DataTypes.Klines.Symbols)
		wm.logger.Info("订阅K线数据",
			zap.Strings("symbols", config.DataTypes.Klines.Symbols),
			zap.Strings("intervals", config.DataTypes.Klines.Intervals))

		if err := exchange.SubscribeKlines(symbols, config.DataTypes.Klines.Intervals, wm.createKlineCallback()); err != nil {
			return fmt.Errorf("订阅K线数据失败: %v", err)
		}
	}

	// 订阅交易数据
	if config.DataTypes.Trades.Enabled && len(config.DataTypes.Trades.Symbols) > 0 {
		symbols := wm.convertToSymbolTypes(config.DataTypes.Trades.Symbols)
		wm.logger.Info("订阅交易数据", zap.Strings("symbols", config.DataTypes.Trades.Symbols))

		if err := exchange.SubscribeTrades(symbols, wm.createTradeCallback()); err != nil {
			return fmt.Errorf("订阅交易数据失败: %v", err)
		}
	}

	return nil
}

// convertToSymbolTypes 将字符串数组转换为Symbol类型数组
func (wm *WebsocketManager) convertToSymbolTypes(symbols []string) []types.Symbol {
	result := make([]types.Symbol, len(symbols))
	for i, symbol := range symbols {
		result[i] = types.Symbol(symbol)
	}
	return result
}

// createTickerCallback 创建行情数据回调函数
func (wm *WebsocketManager) createTickerCallback() types.DataCallback {
	return func(data types.MarketData) error {
		wm.logger.Debug("收到行情数据",
			zap.String("exchange", string(data.GetExchange())),
			zap.String("symbol", string(data.GetSymbol())),
			zap.String("type", string(data.GetDataType())))
		// 这里可以添加数据处理逻辑，如保存到数据库等
		return nil
	}
}

// createOrderbookCallback 创建订单簿数据回调函数
func (wm *WebsocketManager) createOrderbookCallback() types.DataCallback {
	return func(data types.MarketData) error {
		wm.logger.Debug("收到订单簿数据",
			zap.String("exchange", string(data.GetExchange())),
			zap.String("symbol", string(data.GetSymbol())),
			zap.String("type", string(data.GetDataType())))
		// 这里可以添加数据处理逻辑，如保存到数据库等
		return nil
	}
}

// createKlineCallback 创建K线数据回调函数
func (wm *WebsocketManager) createKlineCallback() types.DataCallback {
	return func(data types.MarketData) error {
		wm.logger.Debug("收到K线数据",
			zap.String("exchange", string(data.GetExchange())),
			zap.String("symbol", string(data.GetSymbol())),
			zap.String("type", string(data.GetDataType())))
		// 这里可以添加数据处理逻辑，如保存到数据库等
		return nil
	}
}

// createTradeCallback 创建交易数据回调函数
func (wm *WebsocketManager) createTradeCallback() types.DataCallback {
	return func(data types.MarketData) error {
		wm.logger.Debug("收到交易数据",
			zap.String("exchange", string(data.GetExchange())),
			zap.String("symbol", string(data.GetSymbol())),
			zap.String("type", string(data.GetDataType())))
		// 这里可以添加数据处理逻辑，如保存到数据库等
		return nil
	}
}
