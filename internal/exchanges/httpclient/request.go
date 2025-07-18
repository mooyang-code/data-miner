package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/mooyang-code/data-miner/pkg/cryptotrader/log"
)

// DoRequest 发送自定义请求
func (c *HTTPClient) DoRequest(ctx context.Context, req *Request) (*Response, error) {
	if !c.running {
		return nil, fmt.Errorf("client '%s' is not running", c.config.Name)
	}

	// 检查速率限制
	if req.Options == nil || !req.Options.SkipRateLimit {
		if err := c.checkRateLimit(); err != nil {
			return nil, err
		}
	}

	// 更新统计信息
	atomic.AddInt64(&c.stats.totalRequests, 1)
	c.mu.Lock()
	c.stats.lastRequest = time.Now()
	c.mu.Unlock()

	var response *Response

	// 执行带重试的请求
	err := c.retryHandler.Execute(ctx, func() error {
		resp, err := c.doHTTPRequest(ctx, req)
		if err != nil {
			return err
		}
		response = resp
		return nil
	}, func(attempt int, err error) {
		// 重试回调：切换IP
		atomic.AddInt64(&c.stats.retryCount, 1)
		if c.ipManager != nil && c.config.DynamicIP.Enabled {
			nextIP, switchErr := c.ipManager.GetNextIP()
			if switchErr != nil {
				log.Errorf(log.ExchangeSys, "Client '%s': Failed to switch to next IP: %v", c.config.Name, switchErr)
			} else {
				log.Infof(log.ExchangeSys, "Client '%s': Switching to next IP: %s", c.config.Name, nextIP)
			}
		}
	})

	if err != nil {
		atomic.AddInt64(&c.stats.failedRequests, 1)
		c.mu.Lock()
		c.stats.lastError = err.Error()
		c.mu.Unlock()
		return nil, err
	}

	atomic.AddInt64(&c.stats.successRequests, 1)
	return response, nil
}

// doHTTPRequest 执行实际的HTTP请求
func (c *HTTPClient) doHTTPRequest(ctx context.Context, req *Request) (*Response, error) {
	startTime := time.Now()

	// 准备请求体
	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, NewHTTPError(ErrorTypeHTTP, 0, "failed to marshal request body", req.URL, "", false, err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, bodyReader)
	if err != nil {
		return nil, NewHTTPError(ErrorTypeHTTP, 0, "failed to create request", req.URL, "", false, err)
	}

	// 设置请求头
	c.setRequestHeaders(httpReq, req)

	// 获取当前使用的IP（用于日志）
	currentIP := c.getCurrentIP()

	if c.config.Debug {
		log.Debugf(log.ExchangeSys, "Client '%s': Making %s request to %s with IP %s",
			c.config.Name, req.Method, req.URL, currentIP)
	}

	// 发送请求
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		classifiedErr := ClassifyError(err)
		classifiedErr.URL = req.URL
		classifiedErr.IP = currentIP
		return nil, classifiedErr
	}
	defer httpResp.Body.Close()

	duration := time.Since(startTime)

	if c.config.Debug {
		log.Debugf(log.ExchangeSys, "Client '%s': Response status %d, duration %v",
			c.config.Name, httpResp.StatusCode, duration)
	}

	// 读取响应体
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, NewHTTPError(ErrorTypeNetwork, httpResp.StatusCode, "failed to read response body", req.URL, currentIP, true, err)
	}

	// 检查HTTP状态码
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		retryable := httpResp.StatusCode >= 500 || httpResp.StatusCode == 429
		return nil, NewHTTPError(ErrorTypeHTTP, httpResp.StatusCode,
			fmt.Sprintf("HTTP error %d", httpResp.StatusCode), req.URL, currentIP, retryable, nil)
	}

	// 解析响应到结果对象
	if req.Result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, req.Result); err != nil {
			return nil, NewHTTPError(ErrorTypeHTTP, httpResp.StatusCode, "failed to unmarshal response", req.URL, currentIP, false, err)
		}
	}

	// 构建响应对象
	response := &Response{
		StatusCode: httpResp.StatusCode,
		Headers:    make(map[string]string),
		Body:       respBody,
		Duration:   duration,
		IP:         currentIP,
	}

	// 复制响应头
	for key, values := range httpResp.Header {
		if len(values) > 0 {
			response.Headers[key] = values[0]
		}
	}
	return response, nil
}

// setRequestHeaders 设置请求头
func (c *HTTPClient) setRequestHeaders(httpReq *http.Request, req *Request) {
	// 设置默认请求头
	c.mu.RLock()
	for key, value := range c.defaultHeaders {
		httpReq.Header.Set(key, value)
	}
	c.mu.RUnlock()

	// 设置用户代理
	if c.config.UserAgent != "" {
		httpReq.Header.Set("User-Agent", c.config.UserAgent)
	}

	// 设置内容类型
	if req.Body != nil && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// 设置请求特定的头部
	if req.Headers != nil {
		for key, value := range req.Headers {
			httpReq.Header.Set(key, value)
		}
	}
}

// getCurrentIP 获取当前使用的IP地址
func (c *HTTPClient) getCurrentIP() string {
	if c.ipManager != nil && c.config.DynamicIP.Enabled && c.ipManager.IsRunning() {
		if ip, err := c.ipManager.GetCurrentIP(); err == nil {
			return ip
		}
	}
	return "unknown"
}

// checkRateLimit 检查速率限制
func (c *HTTPClient) checkRateLimit() error {
	if !c.rateLimit.enabled {
		return nil
	}

	c.rateLimit.mu.Lock()
	defer c.rateLimit.mu.Unlock()

	now := time.Now()

	// 每分钟重置计数器
	if now.Sub(c.rateLimit.lastReset) >= time.Minute {
		c.rateLimit.requestCount = 0
		c.rateLimit.lastReset = now
	}

	// 检查是否超过限制
	if c.rateLimit.requestCount >= int64(c.rateLimit.limit) {
		return NewHTTPError(ErrorTypeRateLimit, 429,
			fmt.Sprintf("rate limit exceeded: %d requests per minute", c.rateLimit.limit),
			"", "", true, nil)
	}

	c.rateLimit.requestCount++
	return nil
}

// NewCustomClient 创建自定义配置的HTTP客户端
func NewCustomClient(name, hostname string, enableDynamicIP bool) (Client, error) {
	config := DefaultConfig(name)

	if enableDynamicIP && hostname != "" {
		config.DynamicIP.Enabled = true
		config.DynamicIP.Hostname = hostname
		config.DynamicIP.IPManager.Hostname = hostname
	}
	return New(config)
}
