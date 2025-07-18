// Package httpclient 提供通用的HTTP请求客户端，支持动态IP管理和重试机制
package httpclient

import (
	"context"
	"time"

	"github.com/mooyang-code/data-miner/internal/ipmanager"
)

// Client HTTP客户端接口
type Client interface {
	// Get 发送GET请求
	Get(ctx context.Context, url string, result interface{}) error

	// Post 发送POST请求
	Post(ctx context.Context, url string, body interface{}, result interface{}) error

	// Put 发送PUT请求
	Put(ctx context.Context, url string, body interface{}, result interface{}) error

	// Delete 发送DELETE请求
	Delete(ctx context.Context, url string, result interface{}) error

	// DoRequest 发送自定义请求
	DoRequest(ctx context.Context, req *Request) (*Response, error)

	// SetHeaders 设置默认请求头
	SetHeaders(headers map[string]string)

	// GetStatus 获取客户端状态
	GetStatus() *Status

	// Close 关闭客户端
	Close() error
}

// Request HTTP请求结构
type Request struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    interface{}       `json:"body"`
	Result  interface{}       `json:"-"` // 不序列化
	Timeout time.Duration     `json:"timeout"`
	Options *RequestOptions   `json:"options"`
}

// RequestOptions 请求选项
type RequestOptions struct {
	// 重试相关
	MaxRetries    int           `json:"max_retries"`
	RetryDelay    time.Duration `json:"retry_delay"`
	MaxRetryDelay time.Duration `json:"max_retry_delay"`

	// IP管理相关
	EnableDynamicIP bool `json:"enable_dynamic_ip"`
	ForceIPSwitch   bool `json:"force_ip_switch"`

	// 其他选项
	SkipRateLimit bool `json:"skip_rate_limit"`
	Verbose       bool `json:"verbose"`
}

// Response HTTP响应结构
type Response struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       []byte            `json:"body"`
	Duration   time.Duration     `json:"duration"`
	IP         string            `json:"ip"` // 使用的IP地址
}

// Status 客户端状态
type Status struct {
	// 基本信息
	Name        string    `json:"name"`
	Running     bool      `json:"running"`
	LastRequest time.Time `json:"last_request"`

	// 统计信息
	TotalRequests   int64 `json:"total_requests"`
	SuccessRequests int64 `json:"success_requests"`
	FailedRequests  int64 `json:"failed_requests"`
	RetryCount      int64 `json:"retry_count"`

	// 速率限制
	RateLimit *RateLimitStatus `json:"rate_limit"`

	// IP管理器状态
	IPManager map[string]interface{} `json:"ip_manager"`

	// 错误信息
	LastError string `json:"last_error,omitempty"`
}

// RateLimitStatus 速率限制状态
type RateLimitStatus struct {
	Enabled           bool      `json:"enabled"`
	RequestsPerMinute int       `json:"requests_per_minute"`
	CurrentCount      int64     `json:"current_count"`
	ResetTime         time.Time `json:"reset_time"`
	Remaining         int       `json:"remaining"`
}

// Config HTTP客户端配置
type Config struct {
	// 基本配置
	Name      string        `yaml:"name" json:"name"`
	UserAgent string        `yaml:"user_agent" json:"user_agent"`
	Timeout   time.Duration `yaml:"timeout" json:"timeout"`

	// 动态IP配置
	DynamicIP *DynamicIPConfig `yaml:"dynamic_ip" json:"dynamic_ip"`

	// 重试配置
	Retry *RetryConfig `yaml:"retry" json:"retry"`

	// 速率限制配置
	RateLimit *RateLimitConfig `yaml:"rate_limit" json:"rate_limit"`

	// HTTP传输配置
	Transport *TransportConfig `yaml:"transport" json:"transport"`

	// 调试配置
	Debug bool `yaml:"debug" json:"debug"`
}

// DynamicIPConfig 动态IP配置
type DynamicIPConfig struct {
	Enabled   bool              `yaml:"enabled" json:"enabled"`
	Hostname  string            `yaml:"hostname" json:"hostname"`
	IPManager *ipmanager.Config `yaml:"ip_manager" json:"ip_manager"`
}

// RetryConfig 重试配置
type RetryConfig struct {
	Enabled       bool          `yaml:"enabled" json:"enabled"`
	MaxAttempts   int           `yaml:"max_attempts" json:"max_attempts"`
	InitialDelay  time.Duration `yaml:"initial_delay" json:"initial_delay"`
	MaxDelay      time.Duration `yaml:"max_delay" json:"max_delay"`
	BackoffFactor float64       `yaml:"backoff_factor" json:"backoff_factor"`
}

// RateLimitConfig 速率限制配置
type RateLimitConfig struct {
	Enabled           bool `yaml:"enabled" json:"enabled"`
	RequestsPerMinute int  `yaml:"requests_per_minute" json:"requests_per_minute"`
}

// TransportConfig HTTP传输配置
type TransportConfig struct {
	MaxIdleConns          int           `yaml:"max_idle_conns" json:"max_idle_conns"`
	MaxIdleConnsPerHost   int           `yaml:"max_idle_conns_per_host" json:"max_idle_conns_per_host"`
	MaxConnsPerHost       int           `yaml:"max_conns_per_host" json:"max_conns_per_host"`
	IdleConnTimeout       time.Duration `yaml:"idle_conn_timeout" json:"idle_conn_timeout"`
	TLSHandshakeTimeout   time.Duration `yaml:"tls_handshake_timeout" json:"tls_handshake_timeout"`
	ResponseHeaderTimeout time.Duration `yaml:"response_header_timeout" json:"response_header_timeout"`
	DisableKeepAlives     bool          `yaml:"disable_keep_alives" json:"disable_keep_alives"`
	DisableCompression    bool          `yaml:"disable_compression" json:"disable_compression"`
}

// ErrorType 错误类型
type ErrorType int

const (
	// ErrorTypeUnknown 未知错误
	ErrorTypeUnknown ErrorType = iota
	// ErrorTypeNetwork 网络错误
	ErrorTypeNetwork
	// ErrorTypeTimeout 超时错误
	ErrorTypeTimeout
	// ErrorTypeTLS TLS错误
	ErrorTypeTLS
	// ErrorTypeHTTP HTTP错误
	ErrorTypeHTTP
	// ErrorTypeRateLimit 速率限制错误
	ErrorTypeRateLimit
)

// HTTPError HTTP错误
type HTTPError struct {
	Type       ErrorType `json:"type"`
	StatusCode int       `json:"status_code"`
	Message    string    `json:"message"`
	URL        string    `json:"url"`
	IP         string    `json:"ip"`
	Retryable  bool      `json:"retryable"`
	Cause      error     `json:"-"`
}

// Error 实现error接口
func (e *HTTPError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// Unwrap 实现errors.Unwrap接口
func (e *HTTPError) Unwrap() error {
	return e.Cause
}

// IsRetryable 判断错误是否可重试
func (e *HTTPError) IsRetryable() bool {
	return e.Retryable
}
