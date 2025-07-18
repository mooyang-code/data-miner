package httpclient

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mooyang-code/data-miner/internal/ipmanager"
	"github.com/mooyang-code/data-miner/pkg/cryptotrader/log"
)

// HTTPClient HTTP客户端实现
type HTTPClient struct {
	config       *Config
	httpClient   *http.Client
	ipManager    *ipmanager.Manager
	retryHandler *RetryHandler

	// 状态管理
	mu             sync.RWMutex
	running        bool
	defaultHeaders map[string]string

	// 统计信息
	stats struct {
		totalRequests   int64
		successRequests int64
		failedRequests  int64
		retryCount      int64
		lastRequest     time.Time
		lastError       string
	}

	// 速率限制
	rateLimit struct {
		mu           sync.Mutex
		enabled      bool
		requestCount int64
		lastReset    time.Time
		limit        int
	}
}

// New 创建新的HTTP客户端
func New(config *Config) (Client, error) {
	if config == nil {
		config = DefaultConfig("httpclient")
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	client := &HTTPClient{
		config:         config,
		defaultHeaders: make(map[string]string),
		running:        true,
	}

	// 初始化HTTP客户端
	if err := client.initHTTPClient(); err != nil {
		return nil, fmt.Errorf("failed to init HTTP client: %w", err)
	}

	// 初始化IP管理器
	if err := client.initIPManager(); err != nil {
		return nil, fmt.Errorf("failed to init IP manager: %w", err)
	}

	// 初始化重试处理器
	client.retryHandler = NewRetryHandler(config.Retry, config.Name)

	// 初始化速率限制
	client.initRateLimit()

	log.Infof(log.ExchangeSys, "HTTP client '%s' initialized successfully", config.Name)
	return client, nil
}

// initHTTPClient 初始化HTTP客户端
func (c *HTTPClient) initHTTPClient() error {
	transport := &http.Transport{
		DialContext:           c.customDialContext,
		MaxIdleConns:          c.config.Transport.MaxIdleConns,
		MaxIdleConnsPerHost:   c.config.Transport.MaxIdleConnsPerHost,
		MaxConnsPerHost:       c.config.Transport.MaxConnsPerHost,
		IdleConnTimeout:       c.config.Transport.IdleConnTimeout,
		TLSHandshakeTimeout:   c.config.Transport.TLSHandshakeTimeout,
		ResponseHeaderTimeout: c.config.Transport.ResponseHeaderTimeout,
		DisableKeepAlives:     c.config.Transport.DisableKeepAlives,
		DisableCompression:    c.config.Transport.DisableCompression,
		ForceAttemptHTTP2:     false, // 使用HTTP/1.1更稳定
	}

	c.httpClient = &http.Client{
		Transport: transport,
		Timeout:   c.config.Timeout,
	}
	return nil
}

// initIPManager 初始化IP管理器
func (c *HTTPClient) initIPManager() error {
	if !c.config.DynamicIP.Enabled || c.config.DynamicIP.Hostname == "" {
		log.Debugf(log.ExchangeSys, "Dynamic IP disabled for client '%s'", c.config.Name)
		return nil
	}

	// 创建IP管理器
	c.ipManager = ipmanager.New(c.config.DynamicIP.IPManager)

	// 启动IP管理器
	ctx := context.Background()
	if err := c.ipManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start IP manager: %w", err)
	}

	log.Infof(log.ExchangeSys, "IP manager started for client '%s' with hostname '%s'",
		c.config.Name, c.config.DynamicIP.Hostname)
	return nil
}

// initRateLimit 初始化速率限制
func (c *HTTPClient) initRateLimit() {
	c.rateLimit.enabled = c.config.RateLimit.Enabled
	c.rateLimit.limit = c.config.RateLimit.RequestsPerMinute
	c.rateLimit.lastReset = time.Now()
}

// customDialContext 自定义拨号器，用于IP替换
func (c *HTTPClient) customDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %w", err)
	}

	originalAddr := addr

	// 如果启用了动态IP且匹配目标主机名，使用IP管理器获取IP
	if c.config.DynamicIP.Enabled &&
		c.ipManager != nil &&
		host == c.config.DynamicIP.Hostname &&
		c.ipManager.IsRunning() {

		ip, err := c.ipManager.GetCurrentIP()
		if err != nil {
			log.Warnf(log.ExchangeSys, "Failed to get IP from manager for %s, using original address: %v",
				c.config.Name, err)
		} else {
			// 使用IP替换域名
			addr = net.JoinHostPort(ip, port)
			if c.config.Debug {
				log.Debugf(log.ExchangeSys, "Client '%s': Using IP %s instead of %s for %s",
					c.config.Name, ip, host, originalAddr)
			}
		}
	}

	// 使用默认拨号器
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	return dialer.DialContext(ctx, network, addr)
}

// Get 发送GET请求
func (c *HTTPClient) Get(ctx context.Context, url string, result interface{}) error {
	req := &Request{
		Method: http.MethodGet,
		URL:    url,
		Result: result,
	}
	_, err := c.DoRequest(ctx, req)
	return err
}

// Post 发送POST请求
func (c *HTTPClient) Post(ctx context.Context, url string, body interface{}, result interface{}) error {
	req := &Request{
		Method: http.MethodPost,
		URL:    url,
		Body:   body,
		Result: result,
	}
	_, err := c.DoRequest(ctx, req)
	return err
}

// Put 发送PUT请求
func (c *HTTPClient) Put(ctx context.Context, url string, body interface{}, result interface{}) error {
	req := &Request{
		Method: http.MethodPut,
		URL:    url,
		Body:   body,
		Result: result,
	}
	_, err := c.DoRequest(ctx, req)
	return err
}

// Delete 发送DELETE请求
func (c *HTTPClient) Delete(ctx context.Context, url string, result interface{}) error {
	req := &Request{
		Method: http.MethodDelete,
		URL:    url,
		Result: result,
	}
	_, err := c.DoRequest(ctx, req)
	return err
}

// SetHeaders 设置默认请求头
func (c *HTTPClient) SetHeaders(headers map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, value := range headers {
		c.defaultHeaders[key] = value
	}
}

// GetStatus 获取客户端状态
func (c *HTTPClient) GetStatus() *Status {
	c.mu.RLock()
	defer c.mu.RUnlock()

	status := &Status{
		Name:            c.config.Name,
		Running:         c.running,
		LastRequest:     c.stats.lastRequest,
		TotalRequests:   atomic.LoadInt64(&c.stats.totalRequests),
		SuccessRequests: atomic.LoadInt64(&c.stats.successRequests),
		FailedRequests:  atomic.LoadInt64(&c.stats.failedRequests),
		RetryCount:      atomic.LoadInt64(&c.stats.retryCount),
		LastError:       c.stats.lastError,
	}

	// 速率限制状态
	c.rateLimit.mu.Lock()
	remaining := c.rateLimit.limit - int(c.rateLimit.requestCount)
	if remaining < 0 {
		remaining = 0
	}
	status.RateLimit = &RateLimitStatus{
		Enabled:           c.rateLimit.enabled,
		RequestsPerMinute: c.rateLimit.limit,
		CurrentCount:      c.rateLimit.requestCount,
		ResetTime:         c.rateLimit.lastReset.Add(time.Minute),
		Remaining:         remaining,
	}
	c.rateLimit.mu.Unlock()

	// IP管理器状态
	if c.ipManager != nil {
		status.IPManager = c.ipManager.GetStatus()
	}
	return status
}

// Close 关闭客户端
func (c *HTTPClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.running = false

	// 停止IP管理器
	if c.ipManager != nil {
		c.ipManager.Stop()
		log.Infof(log.ExchangeSys, "IP manager stopped for client '%s'", c.config.Name)
	}
	log.Infof(log.ExchangeSys, "HTTP client '%s' closed", c.config.Name)
	return nil
}
