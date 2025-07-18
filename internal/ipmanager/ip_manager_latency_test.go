package ipmanager

import (
	"context"
	"testing"
	"time"
)

func TestLatencyCheckEnabled(t *testing.T) {
	config := &Config{
		Hostname:             "google.com",
		UpdateInterval:       1 * time.Minute,
		DNSServers:           []string{"8.8.8.8:53"},
		DNSTimeout:           5 * time.Second,
		EnableLatencyCheck:   true,
		LatencyCheckInterval: 5 * time.Second,
		LatencyTimeout:       2 * time.Second,
		LatencyPort:          "80", // HTTP端口，更容易连接
	}

	manager := New(config)
	if !manager.enableLatencyCheck {
		t.Error("延迟检测应该被启用")
	}

	ctx := context.Background()
	err := manager.Start(ctx)
	if err != nil {
		t.Fatalf("启动管理器失败: %v", err)
	}
	defer manager.Stop()

	// 等待初始化完成
	time.Sleep(3 * time.Second)

	// 检查是否有IP信息
	manager.mu.RLock()
	hasIPs := len(manager.ips) > 0
	hasIPInfos := len(manager.ipInfos) > 0
	manager.mu.RUnlock()

	if !hasIPs {
		t.Error("应该解析到IP地址")
	}

	if !hasIPInfos {
		t.Error("应该创建IP信息列表")
	}

	// 等待延迟检测完成
	time.Sleep(5 * time.Second)

	// 检查延迟信息
	ipInfos := manager.GetAllIPsWithLatency()
	if ipInfos == nil {
		t.Error("应该返回延迟信息")
	}

	if len(ipInfos) == 0 {
		t.Error("应该有IP延迟信息")
	}

	// 检查是否有可用的IP
	hasAvailable := false
	for _, info := range ipInfos {
		if info.Available && info.Latency > 0 {
			hasAvailable = true
			break
		}
	}

	if !hasAvailable {
		t.Error("应该至少有一个可用的IP")
	}
}

func TestLatencyCheckDisabled(t *testing.T) {
	config := &Config{
		Hostname:           "google.com",
		UpdateInterval:     1 * time.Minute,
		DNSServers:         []string{"8.8.8.8:53"},
		DNSTimeout:         5 * time.Second,
		EnableLatencyCheck: false,
	}

	manager := New(config)
	if manager.enableLatencyCheck {
		t.Error("延迟检测应该被禁用")
	}

	ctx := context.Background()
	err := manager.Start(ctx)
	if err != nil {
		t.Fatalf("启动管理器失败: %v", err)
	}
	defer manager.Stop()

	// 等待初始化完成
	time.Sleep(2 * time.Second)

	// 检查延迟信息应该为空
	ipInfos := manager.GetAllIPsWithLatency()
	if ipInfos != nil {
		t.Error("禁用延迟检测时应该返回nil")
	}

	// GetBestIP应该回退到传统方式
	bestIP, latency, err := manager.GetBestIP()
	if err != nil {
		t.Errorf("获取最佳IP失败: %v", err)
	}

	if latency != 0 {
		t.Error("禁用延迟检测时延迟应该为0")
	}

	if bestIP == "" {
		t.Error("应该返回IP地址")
	}
}

func TestMeasureLatency(t *testing.T) {
	config := DefaultConfig("google.com")
	config.LatencyPort = "80"
	config.LatencyTimeout = 2 * time.Second

	manager := New(config)

	// 测试可达的IP
	latency, err := manager.measureLatency("8.8.8.8")
	if err != nil {
		t.Errorf("测量延迟失败: %v", err)
	}

	if latency <= 0 {
		t.Error("延迟应该大于0")
	}

	if latency > 5*time.Second {
		t.Error("延迟不应该超过5秒")
	}

	// 测试不可达的IP（使用一个肯定不存在的私有地址）
	_, err = manager.measureLatency("10.255.255.254") // 私有地址，通常不可达
	if err == nil {
		t.Log("警告: 测试IP可能可达，这在某些网络环境下是正常的")
	}
}

func TestSortIPsByLatency(t *testing.T) {
	manager := &Manager{
		enableLatencyCheck: true,
		ipInfos: []*IPInfo{
			{IP: "1.1.1.1", Latency: 100 * time.Millisecond, Available: true},
			{IP: "2.2.2.2", Latency: 50 * time.Millisecond, Available: true},
			{IP: "3.3.3.3", Latency: 200 * time.Millisecond, Available: false},
			{IP: "4.4.4.4", Latency: 30 * time.Millisecond, Available: true},
		},
	}

	manager.sortIPsByLatency()

	// 检查排序结果：可用的IP应该在前面，按延迟从低到高排序
	expected := []string{"4.4.4.4", "2.2.2.2", "1.1.1.1", "3.3.3.3"}
	
	if len(manager.ips) != len(expected) {
		t.Errorf("IP数量不匹配，期望 %d，实际 %d", len(expected), len(manager.ips))
	}

	for i, expectedIP := range expected {
		if i >= len(manager.ips) || manager.ips[i] != expectedIP {
			t.Errorf("位置 %d 的IP不匹配，期望 %s，实际 %s", i, expectedIP, manager.ips[i])
		}
	}

	// 检查ipInfos也应该按相同顺序排序
	for i, expectedIP := range expected {
		if i >= len(manager.ipInfos) || manager.ipInfos[i].IP != expectedIP {
			t.Errorf("位置 %d 的IPInfo不匹配，期望 %s，实际 %s", i, expectedIP, manager.ipInfos[i].IP)
		}
	}
}

func TestGetBestIP(t *testing.T) {
	manager := &Manager{
		enableLatencyCheck: true,
		ipInfos: []*IPInfo{
			{IP: "1.1.1.1", Latency: 100 * time.Millisecond, Available: true},
			{IP: "2.2.2.2", Latency: 50 * time.Millisecond, Available: true},
			{IP: "3.3.3.3", Latency: 200 * time.Millisecond, Available: false},
		},
	}

	bestIP, latency, err := manager.GetBestIP()
	if err != nil {
		t.Errorf("获取最佳IP失败: %v", err)
	}

	if bestIP != "1.1.1.1" {
		t.Errorf("最佳IP不正确，期望 1.1.1.1，实际 %s", bestIP)
	}

	if latency != 100*time.Millisecond {
		t.Errorf("延迟不正确，期望 100ms，实际 %v", latency)
	}
}

func TestForceLatencyCheck(t *testing.T) {
	config := DefaultConfig("google.com")
	config.EnableLatencyCheck = true
	config.LatencyPort = "80"

	manager := New(config)

	ctx := context.Background()
	err := manager.Start(ctx)
	if err != nil {
		t.Fatalf("启动管理器失败: %v", err)
	}
	defer manager.Stop()

	// 等待初始化
	time.Sleep(2 * time.Second)

	// 强制延迟检测
	manager.ForceLatencyCheck()

	// 等待检测完成
	time.Sleep(3 * time.Second)

	// 检查是否有延迟信息
	ipInfos := manager.GetAllIPsWithLatency()
	if len(ipInfos) == 0 {
		t.Error("强制延迟检测后应该有IP信息")
	}

	hasValidLatency := false
	for _, info := range ipInfos {
		if info.Available && info.Latency > 0 {
			hasValidLatency = true
			break
		}
	}

	if !hasValidLatency {
		t.Error("应该至少有一个IP有有效的延迟信息")
	}
}
