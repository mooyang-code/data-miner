package httpclient

import (
	"context"
	"testing"
)

// TestNewHTTPClient 测试创建HTTP客户端
func TestNewHTTPClient(t *testing.T) {
	config := DefaultConfig("test")
	client, err := New(config)
	if err != nil {
		t.Fatalf("创建HTTP客户端失败: %v", err)
	}
	defer client.Close()

	status := client.GetStatus()
	if status.Name != "test" {
		t.Errorf("期望客户端名称为 'test'，实际为 '%s'", status.Name)
	}

	if !status.Running {
		t.Error("期望客户端处于运行状态")
	}
}

// TestNewCustomClient 测试创建自定义客户端
func TestNewCustomClient(t *testing.T) {
	client, err := NewCustomClient("test", "httpbin.org", false)
	if err != nil {
		t.Fatalf("创建自定义客户端失败: %v", err)
	}
	defer client.Close()
	
	status := client.GetStatus()
	if status.Name != "test" {
		t.Errorf("期望客户端名称为 'test'，实际为 '%s'", status.Name)
	}
}

// TestHTTPGetRequest 测试GET请求
func TestHTTPGetRequest(t *testing.T) {
	client, err := NewCustomClient("test", "", false)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()
	
	ctx := context.Background()
	var result map[string]interface{}
	
	err = client.Get(ctx, "https://httpbin.org/get", &result)
	if err != nil {
		t.Fatalf("GET请求失败: %v", err)
	}
	
	if result["url"] == nil {
		t.Error("响应中缺少url字段")
	}
	
	// 检查统计信息
	status := client.GetStatus()
	if status.TotalRequests != 1 {
		t.Errorf("期望总请求数为1，实际为%d", status.TotalRequests)
	}
	
	if status.SuccessRequests != 1 {
		t.Errorf("期望成功请求数为1，实际为%d", status.SuccessRequests)
	}
}

// TestHTTPPostRequest 测试POST请求
func TestHTTPPostRequest(t *testing.T) {
	client, err := NewCustomClient("test", "", false)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()
	
	ctx := context.Background()
	postData := map[string]interface{}{
		"message": "test",
		"number":  123,
	}
	
	var result map[string]interface{}
	err = client.Post(ctx, "https://httpbin.org/post", postData, &result)
	if err != nil {
		t.Fatalf("POST请求失败: %v", err)
	}
	
	if result["url"] == nil {
		t.Error("响应中缺少url字段")
	}
	
	// 检查是否包含发送的数据
	if result["json"] == nil {
		t.Error("响应中缺少json字段")
	}
}

// TestCustomRequest 测试自定义请求
func TestCustomRequest(t *testing.T) {
	client, err := NewCustomClient("test", "", false)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()
	
	ctx := context.Background()
	req := &Request{
		Method: "PUT",
		URL:    "https://httpbin.org/put",
		Headers: map[string]string{
			"X-Test-Header": "test-value",
		},
		Body: map[string]string{
			"action": "update",
		},
		Options: &RequestOptions{
			MaxRetries:      2,
			EnableDynamicIP: false,
			Verbose:         true,
		},
	}
	
	var result map[string]interface{}
	req.Result = &result
	
	resp, err := client.DoRequest(ctx, req)
	if err != nil {
		t.Fatalf("自定义请求失败: %v", err)
	}
	
	if resp.StatusCode != 200 {
		t.Errorf("期望状态码200，实际为%d", resp.StatusCode)
	}
	
	if resp.IP == "" {
		t.Error("响应中缺少IP信息")
	}
	
	if resp.Duration <= 0 {
		t.Error("响应时间应该大于0")
	}
}

// TestRateLimit 测试速率限制
func TestRateLimit(t *testing.T) {
	config := DefaultConfig("test")
	config.RateLimit.Enabled = true
	config.RateLimit.RequestsPerMinute = 2 // 设置很低的限制用于测试
	
	client, err := New(config)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()
	
	ctx := context.Background()
	
	// 发送第一个请求（应该成功）
	var result1 map[string]interface{}
	err = client.Get(ctx, "https://httpbin.org/get", &result1)
	if err != nil {
		t.Fatalf("第一个请求失败: %v", err)
	}
	
	// 发送第二个请求（应该成功）
	var result2 map[string]interface{}
	err = client.Get(ctx, "https://httpbin.org/get", &result2)
	if err != nil {
		t.Fatalf("第二个请求失败: %v", err)
	}
	
	// 发送第三个请求（应该被速率限制阻止）
	var result3 map[string]interface{}
	err = client.Get(ctx, "https://httpbin.org/get", &result3)
	if err == nil {
		t.Error("第三个请求应该被速率限制阻止")
	}
	
	// 检查错误类型
	if httpErr, ok := err.(*HTTPError); ok {
		if httpErr.Type != ErrorTypeRateLimit {
			t.Errorf("期望速率限制错误，实际为%v", httpErr.Type)
		}
	} else {
		t.Error("期望HTTPError类型的错误")
	}
}

// TestErrorClassification 测试错误分类
func TestErrorClassification(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected ErrorType
	}{
		{
			name:     "404错误",
			url:      "https://httpbin.org/status/404",
			expected: ErrorTypeHTTP,
		},
		{
			name:     "500错误",
			url:      "https://httpbin.org/status/500",
			expected: ErrorTypeHTTP,
		},
		{
			name:     "429错误",
			url:      "https://httpbin.org/status/429",
			expected: ErrorTypeHTTP,
		},
	}
	
	client, err := NewCustomClient("test", "", false)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()
	
	ctx := context.Background()
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result map[string]interface{}
			err := client.Get(ctx, tt.url, &result)
			
			if err == nil {
				t.Error("期望请求失败")
				return
			}
			
			if httpErr, ok := err.(*HTTPError); ok {
				if httpErr.Type != tt.expected {
					t.Errorf("期望错误类型%v，实际为%v", tt.expected, httpErr.Type)
				}
			} else {
				t.Error("期望HTTPError类型的错误")
			}
		})
	}
}

// TestClientStatus 测试客户端状态
func TestClientStatus(t *testing.T) {
	client, err := NewCustomClient("test", "", false)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()
	
	// 初始状态检查
	status := client.GetStatus()
	if status.TotalRequests != 0 {
		t.Errorf("期望初始总请求数为0，实际为%d", status.TotalRequests)
	}
	
	// 发送一个成功请求
	ctx := context.Background()
	var result map[string]interface{}
	err = client.Get(ctx, "https://httpbin.org/get", &result)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	
	// 检查更新后的状态
	status = client.GetStatus()
	if status.TotalRequests != 1 {
		t.Errorf("期望总请求数为1，实际为%d", status.TotalRequests)
	}
	
	if status.SuccessRequests != 1 {
		t.Errorf("期望成功请求数为1，实际为%d", status.SuccessRequests)
	}
	
	if status.FailedRequests != 0 {
		t.Errorf("期望失败请求数为0，实际为%d", status.FailedRequests)
	}
	
	// 检查速率限制状态
	if status.RateLimit == nil {
		t.Error("期望速率限制状态不为nil")
	} else {
		if !status.RateLimit.Enabled {
			t.Error("期望速率限制启用")
		}
	}
}

// TestConfigValidation 测试配置验证
func TestConfigValidation(t *testing.T) {
	// 测试空配置
	config := &Config{}
	err := config.Validate()
	if err != nil {
		t.Fatalf("配置验证失败: %v", err)
	}
	
	// 检查默认值是否被设置
	if config.Name == "" {
		t.Error("期望设置默认名称")
	}
	
	if config.UserAgent == "" {
		t.Error("期望设置默认用户代理")
	}
	
	if config.Timeout <= 0 {
		t.Error("期望设置默认超时时间")
	}
	
	if config.Retry == nil {
		t.Error("期望设置默认重试配置")
	}
	
	if config.RateLimit == nil {
		t.Error("期望设置默认速率限制配置")
	}
	
	if config.Transport == nil {
		t.Error("期望设置默认传输配置")
	}
}

// BenchmarkHTTPGet 性能测试GET请求
func BenchmarkHTTPGet(b *testing.B) {
	client, err := NewCustomClient("bench", "", false)
	if err != nil {
		b.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result map[string]interface{}
		err := client.Get(ctx, "https://httpbin.org/get", &result)
		if err != nil {
			b.Fatalf("请求失败: %v", err)
		}
	}
}
