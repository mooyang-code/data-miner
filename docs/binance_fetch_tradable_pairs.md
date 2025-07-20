# Binance FetchTradablePairs 实现文档

## 概述

本文档描述了在 `data-miner` 项目中为 Binance 交易所实现的 `FetchTradablePairs` 方法，该方法用于获取交易所可交易的交易对列表。

## 实现位置

- **主要实现**: `internal/exchanges/binance/binance.go`
- **API方法**: `internal/exchanges/binance/restapi.go`
- **数据类型**: `internal/exchanges/binance/types.go`
- **测试文件**: `internal/exchanges/binance/fetch_tradable_pairs_test.go`
- **使用示例**: `examples/binance_fetch_tradable_pairs_example.go`

## 方法签名

```go
func (b *Binance) FetchTradablePairs(ctx context.Context, assetType asset.Item) (currency.Pairs, error)
```

## 支持的资产类型

- `asset.Spot` - 现货交易对
- `asset.Margin` - 保证金交易对

## 实现特性

### 1. 资产类型过滤
- 现货交易：只返回 `IsSpotTradingAllowed = true` 的交易对
- 保证金交易：只返回 `IsMarginTradingAllowed = true` 的交易对

### 2. 状态过滤
- 只返回状态为 `"TRADING"` 的交易对
- 自动过滤掉暂停交易或其他状态的交易对

### 3. 错误处理
- REST API 未初始化检查
- 不支持的资产类型错误
- 网络请求错误处理
- 交易对解析错误处理

## 使用示例

### 基本用法

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/mooyang-code/data-miner/internal/exchanges/binance"
    "github.com/mooyang-code/data-miner/pkg/cryptotrader/exchanges/asset"
)

func main() {
    // 创建 Binance 实例
    b := binance.New()
    
    // 获取现货交易对
    pairs, err := b.FetchTradablePairs(context.Background(), asset.Spot)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("找到 %d 个现货交易对\n", len(pairs))
    for i, pair := range pairs {
        if i >= 10 { // 只显示前10个
            break
        }
        fmt.Printf("%d. %s\n", i+1, pair.String())
    }
}
```

### 获取保证金交易对

```go
marginPairs, err := b.FetchTradablePairs(context.Background(), asset.Margin)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("找到 %d 个保证金交易对\n", len(marginPairs))
```

## 测试结果

根据最新的测试结果：

- **现货交易对**: 1,473 个
- **保证金交易对**: 782 个
- **总交易对数量**: 3,180 个（包括所有状态）

### 主要计价货币统计

| 计价货币 | 交易对数量 |
|---------|-----------|
| USDT    | 405       |
| TRY     | 266       |
| BTC     | 211       |
| USDC    | 207       |
| FDUSD   | 136       |
| BNB     | 71        |
| ETH     | 51        |

## API 依赖

### GetExchangeInfo API

该方法依赖于 `GetExchangeInfo` API 来获取交易所的完整交易对信息：

```go
func (b *BinanceRestAPI) GetExchangeInfo(ctx context.Context) (ExchangeInfo, error)
```

### 数据结构

主要使用的数据结构：

```go
type ExchangeInfo struct {
    Timezone   string   `json:"timezone"`
    ServerTime JSONTime `json:"serverTime"`
    Symbols    []Symbol `json:"symbols"`
}

type Symbol struct {
    Symbol                 string `json:"symbol"`
    Status                 string `json:"status"`
    BaseAsset              string `json:"baseAsset"`
    QuoteAsset             string `json:"quoteAsset"`
    IsSpotTradingAllowed   bool   `json:"isSpotTradingAllowed"`
    IsMarginTradingAllowed bool   `json:"isMarginTradingAllowed"`
}
```

## 性能考虑

- 方法会一次性获取所有交易对信息，适合批量处理
- 建议在应用启动时调用一次，然后缓存结果
- 如需实时更新，可以定期调用该方法刷新交易对列表

## 错误处理

常见错误类型：

1. **REST API 未初始化**: `"REST API not initialized"`
2. **不支持的资产类型**: `"unsupported asset type: %v"`
3. **网络错误**: 来自 HTTP 请求的各种网络错误
4. **解析错误**: 交易对字符串解析失败

## 扩展性

该实现为将来支持更多资产类型（如期货、期权等）预留了扩展空间，只需在 switch 语句中添加新的 case 分支即可。
