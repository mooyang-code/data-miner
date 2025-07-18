package app

import (
	"go.uber.org/zap"

	"github.com/mooyang-code/data-miner/internal/exchanges/binance"
	"github.com/mooyang-code/data-miner/internal/types"
)

// ExchangeManager 交易所管理器
type ExchangeManager struct {
	logger *zap.Logger
}

// NewExchangeManager 创建新的交易所管理器
func NewExchangeManager(logger *zap.Logger) *ExchangeManager {
	return &ExchangeManager{
		logger: logger,
	}
}

// Initialize 初始化交易所
func (em *ExchangeManager) Initialize(config *types.Config) (map[string]types.ExchangeInterface, error) {
	exchanges := make(map[string]types.ExchangeInterface)

	// 初始化Binance交易所
	if config.Exchanges.Binance.Enabled {
		binanceExchange := binance.New()
		if err := binanceExchange.Initialize(config.Exchanges.Binance); err != nil {
			em.logger.Fatal("初始化Binance交易所失败", zap.Error(err))
			return nil, err
		}
		exchanges["binance"] = binanceExchange
		em.logger.Info("Binance交易所初始化成功")

		// 记录模式信息
		if config.Exchanges.Binance.UseWebsocket {
			em.logger.Info("Binance配置为WebSocket模式")
		} else {
			em.logger.Info("Binance配置为定时API拉取模式")
		}
	}

	// 这里可以添加其他交易所的初始化逻辑
	// if config.Exchanges.OtherExchange.Enabled {
	//     // 初始化其他交易所
	// }

	return exchanges, nil
}
