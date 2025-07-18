// Package binance WebSocket连接实现
package binance

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/mooyang-code/data-miner/pkg/cryptotrader/encoding/json"
	"github.com/mooyang-code/data-miner/pkg/cryptotrader/log"
)

const (
	binanceWebsocketPort = "9443"        // Binance WebSocket端口
	binanceWebsocketPath = "/stream"     // WebSocket路径
	wsSubscribeMethod    = "SUBSCRIBE"   // 订阅方法
	wsUnsubscribeMethod  = "UNSUBSCRIBE" // 取消订阅方法
)

// WsConnect 初始化WebSocket连接
func (b *Binance) WsConnect() error {
	return b.wsConnectWithRetry(3)
}

// wsConnectWithRetry 尝试连接WebSocket，支持重试和IP切换
func (b *Binance) wsConnectWithRetry(maxRetries int) error {
	// 启动IP管理器（如果还没启动）
	if !b.ipManager.IsRunning() {
		ctx := context.Background() // 在实际应用中，应该传入合适的context
		if err := b.ipManager.Start(ctx); err != nil {
			return fmt.Errorf("failed to start IP manager: %v", err)
		}
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		// 获取当前IP
		ip, err := b.ipManager.GetCurrentIP()
		if err != nil {
			return fmt.Errorf("failed to get IP from manager: %v", err)
		}

		// 构建WebSocket URL
		wsURL := fmt.Sprintf("wss://%s:%s%s", ip, binanceWebsocketPort, binanceWebsocketPath)

		log.Debugf(log.WebsocketMgr, "Attempting to connect to: %s (attempt %d/%d)", wsURL, attempt+1, maxRetries)

		// 尝试连接
		conn, resp, err := b.dialWebSocket(wsURL)
		if err != nil {
			lastErr = err
			if resp != nil {
				log.Errorf(log.WebsocketMgr, "WebSocket connection failed with status: %s", resp.Status)
			}
			log.Warnf(log.WebsocketMgr, "Connection attempt %d failed: %v", attempt+1, err)

			// 如果不是最后一次尝试，切换到下一个IP
			if attempt < maxRetries-1 {
				nextIP, switchErr := b.ipManager.GetNextIP()
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

		b.wsConn = conn
		b.wsConnected = true
		go b.wsReadData()
		return nil
	}

	return fmt.Errorf("failed to connect after %d attempts, last error: %v", maxRetries, lastErr)
}

// dialWebSocket 执行实际的WebSocket连接
func (b *Binance) dialWebSocket(wsURL string) (*gws.Conn, *http.Response, error) {
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
func (b *Binance) wsReadData() {
	defer func() {
		if b.wsConn != nil {
			b.wsConn.Close()
		}
		b.wsConnected = false

		// 尝试重连
		go b.attemptReconnect()
	}()

	for {
		if !b.wsConnected {
			return
		}

		_, message, err := b.wsConn.ReadMessage()
		if err != nil {
			log.Errorf(log.WebsocketMgr, "WebSocket读取错误: %v", err)
			return
		}

		err = b.wsHandleData(message)
		if err != nil {
			log.Errorf(log.WebsocketMgr, "WebSocket处理数据错误: %v", err)
		}
	}
}

// attemptReconnect 尝试重新连接WebSocket
func (b *Binance) attemptReconnect() {
	maxReconnectAttempts := 5
	baseDelay := time.Second * 5

	for attempt := 1; attempt <= maxReconnectAttempts; attempt++ {
		log.Infof(log.WebsocketMgr, "Attempting to reconnect WebSocket (attempt %d/%d)", attempt, maxReconnectAttempts)

		// 指数退避延迟
		delay := time.Duration(attempt) * baseDelay
		time.Sleep(delay)

		// 强制更新IP列表
		if b.ipManager != nil {
			b.ipManager.ForceUpdate()
			time.Sleep(time.Second * 2) // 等待IP更新
		}

		// 尝试重连
		err := b.wsConnectWithRetry(2) // 每次重连尝试2个IP
		if err == nil {
			log.Infof(log.WebsocketMgr, "WebSocket reconnected successfully")

			// 重新订阅之前的频道
			if err := b.resubscribeChannels(); err != nil {
				log.Errorf(log.WebsocketMgr, "Failed to resubscribe channels: %v", err)
			}
			return
		}

		log.Errorf(log.WebsocketMgr, "Reconnection attempt %d failed: %v", attempt, err)
	}

	log.Errorf(log.WebsocketMgr, "Failed to reconnect after %d attempts", maxReconnectAttempts)
}

// resubscribeChannels 重新订阅频道（需要在主程序中实现具体逻辑）
func (b *Binance) resubscribeChannels() error {
	// 这里应该重新订阅之前订阅的频道
	// 在实际应用中，需要保存订阅的频道列表
	log.Infof(log.WebsocketMgr, "Resubscribing to channels...")
	return nil
}

// wsHandleData 处理传入的WebSocket数据
func (b *Binance) wsHandleData(respRaw []byte) error {
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
		return b.handleTradeStream(data)
	case strings.Contains(streamType[1], "ticker"):
		return b.handleTickerStream(data)
	case strings.Contains(streamType[1], "kline"):
		return b.handleKlineStream(data)
	case strings.Contains(streamType[1], "depth"):
		return b.handleDepthStream(data)
	default:
		log.Debugf(log.WebsocketMgr, "未处理的流类型: %s", streamType[1])
	}

	return nil
}

// handleTradeStream 处理交易流数据
func (b *Binance) handleTradeStream(data []byte) error {
	log.Debugf(log.WebsocketMgr, "交易流数据: %s", string(data))
	fmt.Printf("###接收到交易数据: %s\n", string(data))
	return nil
}

// handleTickerStream 处理行情流数据
func (b *Binance) handleTickerStream(data []byte) error {
	log.Debugf(log.WebsocketMgr, "行情流数据: %s", string(data))
	fmt.Printf("###接收到行情数据: %s\n", string(data))
	return nil
}

// handleKlineStream 处理K线流数据
func (b *Binance) handleKlineStream(data []byte) error {
	log.Debugf(log.WebsocketMgr, "K线流数据: %s", string(data))
	fmt.Printf("###接收到K线数据: %s\n", string(data))
	return nil
}

// handleDepthStream 处理深度流数据
func (b *Binance) handleDepthStream(data []byte) error {
	log.Debugf(log.WebsocketMgr, "深度流数据: %s", string(data))
	fmt.Printf("###接收到深度数据: %s\n", string(data))
	return nil
}

// Subscribe 订阅WebSocket频道
func (b *Binance) Subscribe(channels []string) error {
	if !b.wsConnected {
		return errors.New("WebSocket未连接")
	}

	// 创建订阅消息
	req := WsPayload{
		ID:     time.Now().Unix(),
		Method: wsSubscribeMethod,
		Params: channels,
	}

	log.Debugf(log.WebsocketMgr, "发送订阅请求: %+v", req)

	err := b.wsConn.WriteJSON(req)
	if err != nil {
		log.Errorf(log.WebsocketMgr, "发送订阅请求失败: %v", err)
		return fmt.Errorf("发送订阅请求失败: %v", err)
	}

	log.Debugf(log.WebsocketMgr, "订阅请求发送成功")
	return nil
}

// Unsubscribe 取消订阅WebSocket频道
func (b *Binance) Unsubscribe(channels []string) error {
	if !b.wsConnected {
		return errors.New("WebSocket未连接")
	}

	// 创建取消订阅消息
	req := WsPayload{
		ID:     time.Now().Unix(),
		Method: wsUnsubscribeMethod,
		Params: channels,
	}

	return b.wsConn.WriteJSON(req)
}

// WsClose 关闭WebSocket连接
func (b *Binance) WsClose() error {
	b.wsConnected = false
	if b.wsConn != nil {
		return b.wsConn.Close()
	}
	return nil
}

// IsWsConnected 返回WebSocket是否已连接
func (b *Binance) IsWsConnected() bool {
	return b.wsConnected
}

// GetIPManagerStatus 获取IP管理器状态信息
func (b *Binance) GetIPManagerStatus() map[string]interface{} {
	if b.ipManager == nil {
		return map[string]interface{}{
			"running": false,
			"error":   "IP manager not initialized",
		}
	}

	return b.ipManager.GetStatus()
}
