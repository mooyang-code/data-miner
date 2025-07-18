// Package binance WebSocket连接实现
package binance

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/mooyang-code/data-miner/internal/ipmanager"
	"github.com/mooyang-code/data-miner/internal/types"
	"github.com/mooyang-code/data-miner/pkg/cryptotrader/encoding/json"
	"github.com/mooyang-code/data-miner/pkg/cryptotrader/log"
)

// BinanceWebSocket WebSocket客户端
type BinanceWebSocket struct {
	wsConn        *gws.Conn                     // WebSocket连接
	wsConnected   bool                          // WebSocket连接状态
	lastPing      time.Time                     // 最后ping时间
	ipManager     *ipmanager.Manager            // IP管理器
	subscriptions map[string]types.DataCallback // 订阅回调映射
	mu            sync.RWMutex                  // 读写锁
	done          chan struct{}                 // 停止信号通道
}

// NewWebSocket 创建新的WebSocket客户端
func NewWebSocket() *BinanceWebSocket {
	return &BinanceWebSocket{
		ipManager:     ipmanager.New(ipmanager.DefaultConfig("stream.binance.com")),
		subscriptions: make(map[string]types.DataCallback),
		done:          make(chan struct{}),
	}
}

const (
	binanceWebsocketPort = "9443"        // Binance WebSocket端口
	binanceWebsocketPath = "/stream"     // WebSocket路径
	wsSubscribeMethod    = "SUBSCRIBE"   // 订阅方法
	wsUnsubscribeMethod  = "UNSUBSCRIBE" // 取消订阅方法
)

// WsConnect 初始化WebSocket连接
func (ws *BinanceWebSocket) WsConnect() error {
	return ws.wsConnectWithRetry(3)
}

// wsConnectWithRetry 尝试连接WebSocket，支持重试和IP切换
func (ws *BinanceWebSocket) wsConnectWithRetry(maxRetries int) error {
	// 启动IP管理器（如果还没启动）
	if !ws.ipManager.IsRunning() {
		ctx := context.Background() // 在实际应用中，应该传入合适的context
		if err := ws.ipManager.Start(ctx); err != nil {
			return fmt.Errorf("failed to start IP manager: %v", err)
		}
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		// 获取当前IP
		ip, err := ws.ipManager.GetCurrentIP()
		if err != nil {
			return fmt.Errorf("failed to get IP from manager: %v", err)
		}

		// 构建WebSocket URL
		wsURL := fmt.Sprintf("wss://%s:%s%s", ip, binanceWebsocketPort, binanceWebsocketPath)
		log.Debugf(log.WebsocketMgr, "Attempting to connect to: %s (attempt %d/%d)", wsURL, attempt+1, maxRetries)

		// 尝试连接
		conn, resp, err := ws.dialWebSocket(wsURL)
		if err != nil {
			lastErr = err
			if resp != nil {
				log.Errorf(log.WebsocketMgr, "WebSocket connection failed with status: %s", resp.Status)
			}
			log.Warnf(log.WebsocketMgr, "Connection attempt %d failed: %v", attempt+1, err)

			// 如果不是最后一次尝试，切换到下一个IP
			if attempt < maxRetries-1 {
				nextIP, switchErr := ws.ipManager.GetNextIP()
				if switchErr != nil {
					log.Errorf(log.WebsocketMgr, "Failed to switch to next IP: %v", switchErr)
				} else {
					log.Infof(log.WebsocketMgr, "Switching to next IP: %s", nextIP)
				}
				time.Sleep(time.Second * 2) // 等待2秒后重试
			}
			continue
		}
		if resp != nil {
			log.Infof(log.WebsocketMgr, "WebSocket connection successful with status: %s, IP: %s", resp.Status, ip)
		}

		ws.wsConn = conn
		ws.wsConnected = true
		go ws.wsReadData()
		return nil
	}

	return fmt.Errorf("failed to connect after %d attempts, last error: %v", maxRetries, lastErr)
}

// dialWebSocket 执行实际的WebSocket连接
func (ws *BinanceWebSocket) dialWebSocket(wsURL string) (*gws.Conn, *http.Response, error) {
	// 配置拨号器的TLS设置以处理基于IP的连接
	dialer := gws.Dialer{
		HandshakeTimeout: 30 * time.Second,
		Proxy:            http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         "stream.binance.com",
		},
	}

	// 添加Binance期望的请求头
	headers := http.Header{}
	headers.Set("User-Agent", "crypto-data-miner/1.0.0")
	headers.Set("Host", "stream.binance.com")
	return dialer.Dial(wsURL, headers)
}

// wsReadData 接收并传递WebSocket消息进行处理
func (ws *BinanceWebSocket) wsReadData() {
	defer func() {
		if ws.wsConn != nil {
			ws.wsConn.Close()
		}
		ws.wsConnected = false

		// 尝试重连
		go ws.attemptReconnect()
	}()

	for {
		if !ws.wsConnected {
			return
		}

		_, message, err := ws.wsConn.ReadMessage()
		if err != nil {
			log.Errorf(log.WebsocketMgr, "WebSocket读取错误: %v", err)
			return
		}

		err = ws.wsHandleData(message)
		if err != nil {
			log.Errorf(log.WebsocketMgr, "WebSocket处理数据错误: %v", err)
		}
	}
}

// attemptReconnect 尝试重新连接WebSocket
func (ws *BinanceWebSocket) attemptReconnect() {
	maxReconnectAttempts := 5
	baseDelay := time.Second * 5

	for attempt := 1; attempt <= maxReconnectAttempts; attempt++ {
		log.Infof(log.WebsocketMgr, "Attempting to reconnect WebSocket (attempt %d/%d)", attempt, maxReconnectAttempts)

		// 指数退避延迟
		delay := time.Duration(attempt) * baseDelay
		time.Sleep(delay)

		// 强制更新IP列表
		if ws.ipManager != nil {
			ws.ipManager.ForceUpdate()
			time.Sleep(time.Second * 2) // 等待IP更新
		}

		// 尝试重连
		err := ws.wsConnectWithRetry(2) // 每次重连尝试2个IP
		if err == nil {
			log.Infof(log.WebsocketMgr, "WebSocket reconnected successfully")

			// 重新订阅之前的频道
			if err := ws.resubscribeChannels(); err != nil {
				log.Errorf(log.WebsocketMgr, "Failed to resubscribe channels: %v", err)
			}
			return
		}

		log.Errorf(log.WebsocketMgr, "Reconnection attempt %d failed: %v", attempt, err)
	}
	log.Errorf(log.WebsocketMgr, "Failed to reconnect after %d attempts", maxReconnectAttempts)
}

// resubscribeChannels 重新订阅频道
func (ws *BinanceWebSocket) resubscribeChannels() error {
	ws.mu.RLock()
	channels := make([]string, 0, len(ws.subscriptions))
	for channel := range ws.subscriptions {
		channels = append(channels, channel)
	}
	ws.mu.RUnlock()

	if len(channels) == 0 {
		log.Infof(log.WebsocketMgr, "没有需要重新订阅的频道")
		return nil
	}

	log.Infof(log.WebsocketMgr, "重新订阅 %d 个频道: %v", len(channels), channels)
	return ws.Subscribe(channels)
}

// wsHandleData 处理传入的WebSocket数据
func (ws *BinanceWebSocket) wsHandleData(respRaw []byte) error {
	// 记录所有接收到的数据用于调试
	log.Debugf(log.WebsocketMgr, "接收到WebSocket数据: %s", string(respRaw))

	// 尝试解析为JSON以检查是否有效
	var jsonData interface{}
	if err := json.Unmarshal(respRaw, &jsonData); err != nil {
		log.Errorf(log.WebsocketMgr, "无效的JSON数据: %v", err)
		return fmt.Errorf("无效的JSON数据: %v", err)
	}

	// 检查是否为订阅响应
	if id, err := jsonparser.GetInt(respRaw, "id"); err == nil {
		log.Debugf(log.WebsocketMgr, "接收到订阅响应，ID: %d", id)
		if result, err := jsonparser.GetUnsafeString(respRaw, "result"); err == nil {
			if result == "null" {
				log.Debugf(log.WebsocketMgr, "订阅成功，ID: %d", id)
				return nil
			}
		}
		// 检查响应中的错误
		if errorMsg, err := jsonparser.GetUnsafeString(respRaw, "error", "msg"); err == nil {
			log.Errorf(log.WebsocketMgr, "订阅错误: %s", errorMsg)
			return fmt.Errorf("订阅错误: %s", errorMsg)
		}
		return nil
	}

	// 解析流数据
	streamStr, err := jsonparser.GetUnsafeString(respRaw, "stream")
	if err != nil {
		// 不是流消息，可能是响应或错误
		log.Debugf(log.WebsocketMgr, "未找到stream字段，可能是响应或错误: %s", string(respRaw))
		return nil
	}

	log.Debugf(log.WebsocketMgr, "处理流: %s", streamStr)

	// 从流消息中提取数据
	data, _, _, err := jsonparser.Get(respRaw, "data")
	if err != nil {
		log.Errorf(log.WebsocketMgr, "从流中提取数据失败: %v", err)
		return fmt.Errorf("从流中提取数据失败: %v", err)
	}

	// 基本流类型检测
	streamType := strings.Split(streamStr, "@")
	if len(streamType) <= 1 {
		log.Errorf(log.WebsocketMgr, "无效的流格式: %s", streamStr)
		return fmt.Errorf("无效的流格式: %s", streamStr)
	}

	log.Debugf(log.WebsocketMgr, "流类型: %s", streamType[1])

	// 处理不同的流类型
	switch {
	case strings.Contains(streamType[1], "trade"):
		return ws.handleTradeStream(streamStr, data)
	case strings.Contains(streamType[1], "ticker"):
		return ws.handleTickerStream(streamStr, data)
	case strings.Contains(streamType[1], "kline"):
		return ws.handleKlineStream(streamStr, data)
	case strings.Contains(streamType[1], "depth"):
		return ws.handleDepthStream(streamStr, data)
	default:
		log.Debugf(log.WebsocketMgr, "未处理的流类型: %s", streamType[1])
	}
	return nil
}

// handleTradeStream 处理交易流数据
func (ws *BinanceWebSocket) handleTradeStream(streamName string, data []byte) error {
	log.Debugf(log.WebsocketMgr, "交易流数据: %s", string(data))

	// 查找对应的回调函数
	if callback, exists := ws.getSubscriptionCallback(streamName); exists && callback != nil {
		// 这里应该解析数据为 types.Trade 结构
		// 为了简化，暂时直接打印
		log.Debugf(log.WebsocketMgr, "调用交易数据回调: %s", streamName)
		// TODO: 解析数据并调用 callback
	}

	fmt.Printf("###接收到交易数据: %s\n", string(data))
	return nil
}

// handleTickerStream 处理行情流数据
func (ws *BinanceWebSocket) handleTickerStream(streamName string, data []byte) error {
	log.Debugf(log.WebsocketMgr, "行情流数据: %s", string(data))

	// 查找对应的回调函数
	if callback, exists := ws.getSubscriptionCallback(streamName); exists && callback != nil {
		// 这里应该解析数据为 types.Ticker 结构
		// 为了简化，暂时直接打印
		log.Debugf(log.WebsocketMgr, "调用行情数据回调: %s", streamName)
		// TODO: 解析数据并调用 callback
	}

	fmt.Printf("###接收到行情数据: %s\n", string(data))
	return nil
}

// handleKlineStream 处理K线流数据
func (ws *BinanceWebSocket) handleKlineStream(streamName string, data []byte) error {
	log.Debugf(log.WebsocketMgr, "K线流数据: %s", string(data))

	// 查找对应的回调函数
	if callback, exists := ws.getSubscriptionCallback(streamName); exists && callback != nil {
		// 这里应该解析数据为 types.Kline 结构
		// 为了简化，暂时直接打印
		log.Debugf(log.WebsocketMgr, "调用K线数据回调: %s", streamName)
		// TODO: 解析数据并调用 callback
	}

	fmt.Printf("###接收到K线数据: %s\n", string(data))
	return nil
}

// handleDepthStream 处理深度流数据
func (ws *BinanceWebSocket) handleDepthStream(streamName string, data []byte) error {
	log.Debugf(log.WebsocketMgr, "深度流数据: %s", string(data))

	// 查找对应的回调函数
	if callback, exists := ws.getSubscriptionCallback(streamName); exists && callback != nil {
		// 这里应该解析数据为 types.Orderbook 结构
		// 为了简化，暂时直接打印
		log.Debugf(log.WebsocketMgr, "调用深度数据回调: %s", streamName)
		// TODO: 解析数据并调用 callback
	}

	fmt.Printf("###接收到深度数据: %s\n", string(data))
	return nil
}

// Subscribe 订阅WebSocket频道
func (ws *BinanceWebSocket) Subscribe(channels []string) error {
	if !ws.wsConnected {
		return errors.New("WebSocket未连接")
	}

	// 创建订阅消息
	req := WsPayload{
		ID:     time.Now().Unix(),
		Method: wsSubscribeMethod,
		Params: channels,
	}
	log.Debugf(log.WebsocketMgr, "发送订阅请求: %+v", req)

	err := ws.wsConn.WriteJSON(req)
	if err != nil {
		log.Errorf(log.WebsocketMgr, "发送订阅请求失败: %v", err)
		return fmt.Errorf("发送订阅请求失败: %v", err)
	}
	log.Debugf(log.WebsocketMgr, "订阅请求发送成功")
	return nil
}

// Unsubscribe 取消订阅WebSocket频道
func (ws *BinanceWebSocket) Unsubscribe(channels []string) error {
	if !ws.wsConnected {
		return errors.New("WebSocket未连接")
	}

	// 创建取消订阅消息
	req := WsPayload{
		ID:     time.Now().Unix(),
		Method: wsUnsubscribeMethod,
		Params: channels,
	}
	return ws.wsConn.WriteJSON(req)
}

// WsClose 关闭WebSocket连接
func (ws *BinanceWebSocket) WsClose() error {
	ws.wsConnected = false
	if ws.wsConn != nil {
		return ws.wsConn.Close()
	}
	return nil
}

// IsConnected 返回WebSocket是否已连接
func (ws *BinanceWebSocket) IsConnected() bool {
	return ws.wsConnected
}

// GetLastPing 获取最后ping时间
func (ws *BinanceWebSocket) GetLastPing() time.Time {
	return ws.lastPing
}

// GetIPManagerStatus 获取IP管理器状态信息
func (ws *BinanceWebSocket) GetIPManagerStatus() map[string]interface{} {
	if ws.ipManager == nil {
		return map[string]interface{}{
			"running": false,
			"error":   "IP manager not initialized",
		}
	}
	return ws.ipManager.GetStatus()
}

// buildChannelName 构建Binance WebSocket频道名称
func (ws *BinanceWebSocket) buildChannelName(symbol, streamType, param string) string {
	// 将符号转换为小写（Binance WebSocket要求）
	symbol = strings.ToLower(symbol)

	switch streamType {
	case "ticker":
		return fmt.Sprintf("%s@ticker", symbol)
	case "trade":
		return fmt.Sprintf("%s@trade", symbol)
	case "kline":
		return fmt.Sprintf("%s@kline_%s", symbol, param)
	case "depth", "depth5", "depth10", "depth20":
		if param != "" {
			return fmt.Sprintf("%s@%s@%s", symbol, streamType, param)
		}
		return fmt.Sprintf("%s@%s", symbol, streamType)
	default:
		return fmt.Sprintf("%s@%s", symbol, streamType)
	}
}

// addSubscription 添加订阅到内部映射
func (ws *BinanceWebSocket) addSubscription(channel string, callback types.DataCallback) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.subscriptions[channel] = callback
	log.Debugf(log.WebsocketMgr, "添加订阅: %s", channel)
}

// removeSubscription 从内部映射移除订阅
func (ws *BinanceWebSocket) removeSubscription(channel string) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	delete(ws.subscriptions, channel)
	log.Debugf(log.WebsocketMgr, "移除订阅: %s", channel)
}

// getSubscriptionCallback 获取订阅的回调函数
func (ws *BinanceWebSocket) getSubscriptionCallback(channel string) (types.DataCallback, bool) {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	callback, exists := ws.subscriptions[channel]
	return callback, exists
}

// SubscribeTicker 订阅行情数据
func (ws *BinanceWebSocket) SubscribeTicker(symbols []types.Symbol, callback types.DataCallback) error {
	if !ws.wsConnected {
		return errors.New("WebSocket未连接")
	}

	var channels []string
	for _, symbol := range symbols {
		channel := ws.buildChannelName(string(symbol), "ticker", "")
		channels = append(channels, channel)
		ws.addSubscription(channel, callback)
	}
	return ws.Subscribe(channels)
}

// SubscribeOrderbook 订阅订单簿数据
func (ws *BinanceWebSocket) SubscribeOrderbook(symbols []types.Symbol, callback types.DataCallback) error {
	if !ws.wsConnected {
		return errors.New("WebSocket未连接")
	}

	var channels []string
	for _, symbol := range symbols {
		// 默认使用20档深度，100ms更新频率
		channel := ws.buildChannelName(string(symbol), "depth20", "100ms")
		channels = append(channels, channel)
		ws.addSubscription(channel, callback)
	}
	return ws.Subscribe(channels)
}

// SubscribeTrades 订阅交易数据
func (ws *BinanceWebSocket) SubscribeTrades(symbols []types.Symbol, callback types.DataCallback) error {
	if !ws.wsConnected {
		return errors.New("WebSocket未连接")
	}

	var channels []string
	for _, symbol := range symbols {
		channel := ws.buildChannelName(string(symbol), "trade", "")
		channels = append(channels, channel)
		ws.addSubscription(channel, callback)
	}
	return ws.Subscribe(channels)
}

// SubscribeKlines 订阅K线数据
func (ws *BinanceWebSocket) SubscribeKlines(symbols []types.Symbol, intervals []string, callback types.DataCallback) error {
	if !ws.wsConnected {
		return errors.New("WebSocket未连接")
	}

	var channels []string
	for _, symbol := range symbols {
		for _, interval := range intervals {
			channel := ws.buildChannelName(string(symbol), "kline", interval)
			channels = append(channels, channel)
			ws.addSubscription(channel, callback)
		}
	}
	return ws.Subscribe(channels)
}

// UnsubscribeAll 取消所有订阅
func (ws *BinanceWebSocket) UnsubscribeAll() error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if len(ws.subscriptions) == 0 {
		return nil
	}

	var channels []string
	for channel := range ws.subscriptions {
		channels = append(channels, channel)
	}

	// 清空订阅映射
	ws.subscriptions = make(map[string]types.DataCallback)
	if ws.wsConnected {
		return ws.Unsubscribe(channels)
	}
	return nil
}

// SubscribeTickerWithDepth 订阅行情数据（带深度选项）
func (ws *BinanceWebSocket) SubscribeTickerWithDepth(symbols []types.Symbol, callback types.DataCallback) error {
	return ws.SubscribeTicker(symbols, callback)
}

// SubscribeOrderbookWithDepth 订阅订单簿数据（自定义深度）
func (ws *BinanceWebSocket) SubscribeOrderbookWithDepth(symbols []types.Symbol, depth int, updateSpeed string, callback types.DataCallback) error {
	if !ws.wsConnected {
		return errors.New("WebSocket未连接")
	}

	var channels []string
	for _, symbol := range symbols {
		var streamType string
		switch depth {
		case 5:
			streamType = "depth5"
		case 10:
			streamType = "depth10"
		case 20:
			streamType = "depth20"
		default:
			streamType = "depth"
		}

		channel := ws.buildChannelName(string(symbol), streamType, updateSpeed)
		channels = append(channels, channel)
		ws.addSubscription(channel, callback)
	}
	return ws.Subscribe(channels)
}

// GetActiveSubscriptions 获取当前活跃的订阅列表
func (ws *BinanceWebSocket) GetActiveSubscriptions() []string {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	channels := make([]string, 0, len(ws.subscriptions))
	for channel := range ws.subscriptions {
		channels = append(channels, channel)
	}
	return channels
}

// GetSubscriptionCount 获取当前订阅数量
func (ws *BinanceWebSocket) GetSubscriptionCount() int {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return len(ws.subscriptions)
}
