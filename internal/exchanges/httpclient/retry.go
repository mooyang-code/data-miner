package httpclient

import (
	"context"
	"github.com/avast/retry-go/v4"
	"github.com/mooyang-code/data-miner/pkg/cryptotrader/log"
	"net"
	"strings"
)

// RetryHandler 重试处理器
type RetryHandler struct {
	config *RetryConfig
	name   string
}

// NewRetryHandler 创建重试处理器
func NewRetryHandler(config *RetryConfig, name string) *RetryHandler {
	if config == nil {
		config = DefaultRetryConfig()
	}
	return &RetryHandler{
		config: config,
		name:   name,
	}
}

// Execute 执行带重试的操作
func (r *RetryHandler) Execute(ctx context.Context, operation func() error, onRetry func(attempt int, err error)) error {
	if !r.config.Enabled {
		return operation()
	}

	return retry.Do(
		operation,
		retry.Context(ctx),
		retry.RetryIf(func(err error) bool {
			if !r.isRetryableError(err) {
				log.Warnf(log.ExchangeSys, "%s: Non-retryable error: %v", r.name, err)
				return false
			}
			return true
		}),
		retry.Attempts(uint(r.config.MaxAttempts)),
		retry.LastErrorOnly(true),
		retry.DelayType(retry.BackOffDelay),
		retry.Delay(r.config.InitialDelay),
		retry.MaxDelay(r.config.MaxDelay),
		retry.OnRetry(func(n uint, err error) {
			log.Warnf(log.ExchangeSys, "%s: Attempt %d failed, retrying: %v", r.name, n+1, err)

			// 调用重试回调
			if onRetry != nil {
				onRetry(int(n+1), err)
			}
		}),
	)
}

// isRetryableError 判断错误是否可重试
func (r *RetryHandler) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// 检查自定义HTTP错误
	if httpErr, ok := err.(*HTTPError); ok {
		return httpErr.IsRetryable()
	}

	errStr := strings.ToLower(err.Error())

	// 网络连接错误 - 可重试
	if strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "connection timeout") ||
		strings.Contains(errStr, "network is unreachable") ||
		strings.Contains(errStr, "no route to host") {
		return true
	}

	// 超时错误 - 可重试
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "deadline exceeded") ||
		strings.Contains(errStr, "context deadline exceeded") {
		return true
	}

	// EOF错误 - 通常是连接被过早关闭，可重试
	if strings.Contains(errStr, "eof") ||
		strings.Contains(errStr, "unexpected eof") {
		return true
	}

	// TLS相关错误 - 可重试
	if strings.Contains(errStr, "tls") ||
		strings.Contains(errStr, "handshake") ||
		strings.Contains(errStr, "certificate") {
		return true
	}

	// DNS解析错误 - 可重试
	if strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "dns") {
		return true
	}

	// HTTP 5xx错误 - 服务器错误，可重试
	if strings.Contains(errStr, "500") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "504") ||
		strings.Contains(errStr, "internal server error") ||
		strings.Contains(errStr, "bad gateway") ||
		strings.Contains(errStr, "service unavailable") ||
		strings.Contains(errStr, "gateway timeout") {
		return true
	}

	// HTTP 429错误 - 速率限制，可重试
	if strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "too many requests") {
		return true
	}

	// 检查网络错误类型
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout() || netErr.Temporary()
	}

	// 其他错误默认不重试
	return false
}

// ClassifyError 分类错误类型
func ClassifyError(err error) *HTTPError {
	if err == nil {
		return nil
	}

	// 如果已经是HTTPError，直接返回
	if httpErr, ok := err.(*HTTPError); ok {
		return httpErr
	}

	errStr := strings.ToLower(err.Error())
	httpErr := &HTTPError{
		Type:      ErrorTypeUnknown,
		Message:   err.Error(),
		Retryable: false,
		Cause:     err,
	}

	// 分类网络错误
	if strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "connection timeout") ||
		strings.Contains(errStr, "network is unreachable") ||
		strings.Contains(errStr, "no route to host") ||
		strings.Contains(errStr, "eof") {
		httpErr.Type = ErrorTypeNetwork
		httpErr.Retryable = true
		return httpErr
	}

	// 分类超时错误
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "deadline exceeded") {
		httpErr.Type = ErrorTypeTimeout
		httpErr.Retryable = true
		return httpErr
	}

	// 分类TLS错误
	if strings.Contains(errStr, "tls") ||
		strings.Contains(errStr, "handshake") ||
		strings.Contains(errStr, "certificate") {
		httpErr.Type = ErrorTypeTLS
		httpErr.Retryable = true
		return httpErr
	}

	// 分类HTTP错误
	if strings.Contains(errStr, "500") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "504") {
		httpErr.Type = ErrorTypeHTTP
		httpErr.Retryable = true
		return httpErr
	}

	if strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "too many requests") {
		httpErr.Type = ErrorTypeRateLimit
		httpErr.Retryable = true
		return httpErr
	}

	// 检查网络错误类型
	if netErr, ok := err.(net.Error); ok {
		if netErr.Timeout() {
			httpErr.Type = ErrorTypeTimeout
			httpErr.Retryable = true
		} else if netErr.Temporary() {
			httpErr.Type = ErrorTypeNetwork
			httpErr.Retryable = true
		}
		return httpErr
	}

	return httpErr
}

// NewHTTPError 创建HTTP错误
func NewHTTPError(errorType ErrorType, statusCode int, message, url, ip string, retryable bool, cause error) *HTTPError {
	return &HTTPError{
		Type:       errorType,
		StatusCode: statusCode,
		Message:    message,
		URL:        url,
		IP:         ip,
		Retryable:  retryable,
		Cause:      cause,
	}
}

// IsNetworkError 判断是否为网络错误
func IsNetworkError(err error) bool {
	if httpErr, ok := err.(*HTTPError); ok {
		return httpErr.Type == ErrorTypeNetwork
	}
	return false
}

// IsTimeoutError 判断是否为超时错误
func IsTimeoutError(err error) bool {
	if httpErr, ok := err.(*HTTPError); ok {
		return httpErr.Type == ErrorTypeTimeout
	}
	return false
}

// IsTLSError 判断是否为TLS错误
func IsTLSError(err error) bool {
	if httpErr, ok := err.(*HTTPError); ok {
		return httpErr.Type == ErrorTypeTLS
	}
	return false
}

// IsRateLimitError 判断是否为速率限制错误
func IsRateLimitError(err error) bool {
	if httpErr, ok := err.(*HTTPError); ok {
		return httpErr.Type == ErrorTypeRateLimit
	}
	return false
}
