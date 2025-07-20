package binance

import (
	"context"
	"testing"

	"github.com/mooyang-code/data-miner/internal/exchanges/asset"
)

func TestFetchTradablePairs(t *testing.T) {
	// 创建Binance实例
	b := New()

	// 测试现货交易对
	t.Run("Spot Asset", func(t *testing.T) {
		pairs, err := b.FetchTradablePairs(context.Background(), asset.Spot)
		if err != nil {
			t.Logf("Expected error in offline test: %v", err)
			// 在离线测试中，我们期望会有网络错误
			return
		}

		if len(pairs) == 0 {
			t.Error("Expected to get some trading pairs for spot asset")
		}

		t.Logf("Found %d spot trading pairs", len(pairs))

		// 打印前几个交易对作为示例
		for i, pair := range pairs {
			if i >= 5 { // 只打印前5个
				break
			}
			t.Logf("Pair %d: %s", i+1, pair.String())
		}
	})

	// 测试保证金交易对
	t.Run("Margin Asset", func(t *testing.T) {
		pairs, err := b.FetchTradablePairs(context.Background(), asset.Margin)
		if err != nil {
			t.Logf("Expected error in offline test: %v", err)
			// 在离线测试中，我们期望会有网络错误
			return
		}

		if len(pairs) == 0 {
			t.Error("Expected to get some trading pairs for margin asset")
		}

		t.Logf("Found %d margin trading pairs", len(pairs))
	})

	// 测试不支持的资产类型
	t.Run("Unsupported Asset", func(t *testing.T) {
		_, err := b.FetchTradablePairs(context.Background(), asset.Futures)
		if err == nil {
			t.Error("Expected error for unsupported asset type")
		}
		t.Logf("Got expected error for unsupported asset: %v", err)
	})

	// 测试未初始化的REST API
	t.Run("Uninitialized REST API", func(t *testing.T) {
		emptyBinance := &Binance{}
		_, err := emptyBinance.FetchTradablePairs(context.Background(), asset.Spot)
		if err == nil {
			t.Error("Expected error for uninitialized REST API")
		}
		t.Logf("Got expected error for uninitialized REST API: %v", err)
	})
}

func TestGetExchangeInfo(t *testing.T) {
	// 创建Binance实例
	b := New()

	// 测试获取交易所信息
	exchangeInfo, err := b.RestAPI.GetExchangeInfo(context.Background())
	if err != nil {
		t.Logf("Expected error in offline test: %v", err)
		// 在离线测试中，我们期望会有网络错误
		return
	}

	if len(exchangeInfo.Symbols) == 0 {
		t.Error("Expected to get some symbols from exchange info")
	}

	t.Logf("Exchange timezone: %s", exchangeInfo.Timezone)
	t.Logf("Total symbols: %d", len(exchangeInfo.Symbols))

	// 检查一些交易对的属性
	spotCount := 0
	marginCount := 0
	for _, symbol := range exchangeInfo.Symbols {
		if symbol.IsSpotTradingAllowed {
			spotCount++
		}
		if symbol.IsMarginTradingAllowed {
			marginCount++
		}
	}

	t.Logf("Spot trading allowed symbols: %d", spotCount)
	t.Logf("Margin trading allowed symbols: %d", marginCount)
}
