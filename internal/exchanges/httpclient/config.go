package httpclient

import (
	"time"

	"github.com/mooyang-code/data-miner/internal/ipmanager"
)

// DefaultConfig 返回默认配置
func DefaultConfig(name string) *Config {
	return &Config{
		Name:      name,
		UserAgent: "crypto-data-miner/1.0.0",
		Timeout:   30 * time.Second,
		DynamicIP: DefaultDynamicIPConfig(),
		Retry:     DefaultRetryConfig(),
		RateLimit: DefaultRateLimitConfig(),
		Transport: DefaultTransportConfig(),
		Debug:     false,
	}
}

// DefaultDynamicIPConfig 返回默认动态IP配置
func DefaultDynamicIPConfig() *DynamicIPConfig {
	return &DynamicIPConfig{
		Enabled:   false,
		Hostname:  "",
		IPManager: ipmanager.DefaultConfig(""),
	}
}

// DefaultRetryConfig 返回默认重试配置
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		Enabled:       true,
		MaxAttempts:   5,
		InitialDelay:  time.Second,
		MaxDelay:      8 * time.Second,
		BackoffFactor: 2.0,
	}
}

// DefaultRateLimitConfig 返回默认速率限制配置
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 1200, // 默认限制
	}
}

// DefaultTransportConfig 返回默认传输配置
func DefaultTransportConfig() *TransportConfig {
	return &TransportConfig{
		MaxIdleConns:          50,
		MaxIdleConnsPerHost:   10,
		MaxConnsPerHost:       15,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   15 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		DisableKeepAlives:     false,
		DisableCompression:    false,
	}
}

// DefaultRequestOptions 返回默认请求选项
func DefaultRequestOptions() *RequestOptions {
	return &RequestOptions{
		MaxRetries:      3,
		RetryDelay:      time.Second,
		MaxRetryDelay:   8 * time.Second,
		EnableDynamicIP: true,
		ForceIPSwitch:   false,
		SkipRateLimit:   false,
		Verbose:         false,
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Name == "" {
		c.Name = "httpclient"
	}

	if c.UserAgent == "" {
		c.UserAgent = "crypto-data-miner/1.0.0"
	}

	if c.Timeout <= 0 {
		c.Timeout = 30 * time.Second
	}

	if c.DynamicIP == nil {
		c.DynamicIP = DefaultDynamicIPConfig()
	}

	if c.Retry == nil {
		c.Retry = DefaultRetryConfig()
	}

	if c.RateLimit == nil {
		c.RateLimit = DefaultRateLimitConfig()
	}

	if c.Transport == nil {
		c.Transport = DefaultTransportConfig()
	}

	// 验证重试配置
	if c.Retry.MaxAttempts < 1 {
		c.Retry.MaxAttempts = 3
	}
	if c.Retry.InitialDelay <= 0 {
		c.Retry.InitialDelay = time.Second
	}
	if c.Retry.MaxDelay <= 0 {
		c.Retry.MaxDelay = 8 * time.Second
	}
	if c.Retry.BackoffFactor <= 0 {
		c.Retry.BackoffFactor = 2.0
	}

	// 验证速率限制配置
	if c.RateLimit.RequestsPerMinute < 1 {
		c.RateLimit.RequestsPerMinute = 60
	}

	// 验证传输配置
	if c.Transport.MaxIdleConns < 1 {
		c.Transport.MaxIdleConns = 50
	}
	if c.Transport.MaxIdleConnsPerHost < 1 {
		c.Transport.MaxIdleConnsPerHost = 10
	}
	if c.Transport.MaxConnsPerHost < 1 {
		c.Transport.MaxConnsPerHost = 15
	}
	if c.Transport.IdleConnTimeout <= 0 {
		c.Transport.IdleConnTimeout = 60 * time.Second
	}
	if c.Transport.TLSHandshakeTimeout <= 0 {
		c.Transport.TLSHandshakeTimeout = 15 * time.Second
	}
	if c.Transport.ResponseHeaderTimeout <= 0 {
		c.Transport.ResponseHeaderTimeout = 15 * time.Second
	}
	return nil
}

// Merge 合并配置
func (c *Config) Merge(other *Config) *Config {
	if other == nil {
		return c
	}

	result := *c // 复制当前配置
	if other.Name != "" {
		result.Name = other.Name
	}
	if other.UserAgent != "" {
		result.UserAgent = other.UserAgent
	}
	if other.Timeout > 0 {
		result.Timeout = other.Timeout
	}

	// 合并动态IP配置
	if other.DynamicIP != nil {
		if result.DynamicIP == nil {
			result.DynamicIP = &DynamicIPConfig{}
		}
		if other.DynamicIP.Hostname != "" {
			result.DynamicIP.Hostname = other.DynamicIP.Hostname
		}
		result.DynamicIP.Enabled = other.DynamicIP.Enabled
		if other.DynamicIP.IPManager != nil {
			result.DynamicIP.IPManager = other.DynamicIP.IPManager
		}
	}

	// 合并重试配置
	if other.Retry != nil {
		if result.Retry == nil {
			result.Retry = &RetryConfig{}
		}
		result.Retry.Enabled = other.Retry.Enabled
		if other.Retry.MaxAttempts > 0 {
			result.Retry.MaxAttempts = other.Retry.MaxAttempts
		}
		if other.Retry.InitialDelay > 0 {
			result.Retry.InitialDelay = other.Retry.InitialDelay
		}
		if other.Retry.MaxDelay > 0 {
			result.Retry.MaxDelay = other.Retry.MaxDelay
		}
		if other.Retry.BackoffFactor > 0 {
			result.Retry.BackoffFactor = other.Retry.BackoffFactor
		}
	}

	// 合并速率限制配置
	if other.RateLimit != nil {
		if result.RateLimit == nil {
			result.RateLimit = &RateLimitConfig{}
		}
		result.RateLimit.Enabled = other.RateLimit.Enabled
		if other.RateLimit.RequestsPerMinute > 0 {
			result.RateLimit.RequestsPerMinute = other.RateLimit.RequestsPerMinute
		}
	}

	// 合并传输配置
	if other.Transport != nil {
		if result.Transport == nil {
			result.Transport = &TransportConfig{}
		}
		if other.Transport.MaxIdleConns > 0 {
			result.Transport.MaxIdleConns = other.Transport.MaxIdleConns
		}
		if other.Transport.MaxIdleConnsPerHost > 0 {
			result.Transport.MaxIdleConnsPerHost = other.Transport.MaxIdleConnsPerHost
		}
		if other.Transport.MaxConnsPerHost > 0 {
			result.Transport.MaxConnsPerHost = other.Transport.MaxConnsPerHost
		}
		if other.Transport.IdleConnTimeout > 0 {
			result.Transport.IdleConnTimeout = other.Transport.IdleConnTimeout
		}
		if other.Transport.TLSHandshakeTimeout > 0 {
			result.Transport.TLSHandshakeTimeout = other.Transport.TLSHandshakeTimeout
		}
		if other.Transport.ResponseHeaderTimeout > 0 {
			result.Transport.ResponseHeaderTimeout = other.Transport.ResponseHeaderTimeout
		}
		result.Transport.DisableKeepAlives = other.Transport.DisableKeepAlives
		result.Transport.DisableCompression = other.Transport.DisableCompression
	}
	return &result
}
