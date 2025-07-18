// Package ipmanager 提供通用的IP地址管理功能
// 支持动态DNS解析、故障转移和自动更新
package ipmanager

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/mooyang-code/data-miner/pkg/cryptotrader/log"
)

// Manager 管理域名对应的IP地址列表
type Manager struct {
	mu         sync.RWMutex
	ips        []string
	currentIdx int
	hostname   string
	updateChan chan struct{}
	stopChan   chan struct{}
	isRunning  bool

	// 配置选项
	updateInterval time.Duration
	dnsServers     []string
	dnsTimeout     time.Duration
}

// Config IP管理器配置
type Config struct {
	Hostname       string        // 要解析的域名
	UpdateInterval time.Duration // 更新间隔，默认5分钟
	DNSServers     []string      // DNS服务器列表
	DNSTimeout     time.Duration // DNS查询超时时间，默认5秒
}

// DefaultConfig 返回默认配置
func DefaultConfig(hostname string) *Config {
	return &Config{
		Hostname:       hostname,
		UpdateInterval: 5 * time.Minute,
		DNSServers: []string{
			"8.8.8.8:53",        // Google DNS
			"1.1.1.1:53",        // Cloudflare DNS
			"208.67.222.222:53", // OpenDNS
		},
		DNSTimeout: 5 * time.Second,
	}
}

// New 创建新的IP管理器
func New(config *Config) *Manager {
	if config == nil {
		config = DefaultConfig("")
	}

	// 设置默认值
	if config.UpdateInterval == 0 {
		config.UpdateInterval = 5 * time.Minute
	}
	if config.DNSTimeout == 0 {
		config.DNSTimeout = 5 * time.Second
	}
	if len(config.DNSServers) == 0 {
		config.DNSServers = DefaultConfig("").DNSServers
	}

	return &Manager{
		hostname:       config.Hostname,
		ips:            make([]string, 0),
		updateChan:     make(chan struct{}, 1),
		stopChan:       make(chan struct{}),
		updateInterval: config.UpdateInterval,
		dnsServers:     config.DNSServers,
		dnsTimeout:     config.DNSTimeout,
	}
}

// Start 启动IP管理器
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.isRunning {
		m.mu.Unlock()
		return fmt.Errorf("IP manager is already running")
	}
	m.isRunning = true
	m.mu.Unlock()

	// 立即获取一次IP列表
	if err := m.updateIPs(); err != nil {
		log.Errorf(log.WebsocketMgr, "Failed to get initial IP list for %s: %v", m.hostname, err)
		return err
	}

	// 启动定时更新协程
	go m.updateLoop(ctx)

	log.Infof(log.WebsocketMgr, "IP Manager started for hostname: %s", m.hostname)
	return nil
}

// Stop 停止IP管理器
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		return
	}

	close(m.stopChan)
	m.isRunning = false
	log.Infof(log.WebsocketMgr, "IP Manager stopped for hostname: %s", m.hostname)
}

// GetCurrentIP 获取当前可用的IP地址
func (m *Manager) GetCurrentIP() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.ips) == 0 {
		return "", fmt.Errorf("no available IPs for hostname: %s", m.hostname)
	}

	ip := m.ips[m.currentIdx]
	log.Debugf(log.WebsocketMgr, "Using IP: %s (index: %d/%d) for %s",
		ip, m.currentIdx, len(m.ips)-1, m.hostname)
	return ip, nil
}

// GetNextIP 获取下一个可用的IP地址（用于故障转移）
func (m *Manager) GetNextIP() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.ips) == 0 {
		return "", fmt.Errorf("no available IPs for hostname: %s", m.hostname)
	}

	// 移动到下一个IP
	m.currentIdx = (m.currentIdx + 1) % len(m.ips)
	ip := m.ips[m.currentIdx]

	log.Infof(log.WebsocketMgr, "Switched to next IP: %s (index: %d/%d) for %s",
		ip, m.currentIdx, len(m.ips)-1, m.hostname)
	return ip, nil
}

// GetAllIPs 获取所有可用的IP地址
func (m *Manager) GetAllIPs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 返回副本以避免并发修改
	result := make([]string, len(m.ips))
	copy(result, m.ips)
	return result
}

// GetHostname 获取管理的域名
func (m *Manager) GetHostname() string {
	return m.hostname
}

// ForceUpdate 强制更新IP列表
func (m *Manager) ForceUpdate() {
	select {
	case m.updateChan <- struct{}{}:
		log.Debugf(log.WebsocketMgr, "Forced IP update requested for %s", m.hostname)
	default:
		log.Debugf(log.WebsocketMgr, "IP update already pending for %s", m.hostname)
	}
}

// IsRunning 检查IP管理器是否正在运行
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isRunning
}

// GetStatus 获取IP管理器状态信息
func (m *Manager) GetStatus() map[string]interface{} {
	if !m.IsRunning() {
		return map[string]interface{}{
			"hostname": m.hostname,
			"running":  false,
			"error":    "IP manager not running",
		}
	}

	currentIP, err := m.GetCurrentIP()
	allIPs := m.GetAllIPs()

	status := map[string]interface{}{
		"hostname":        m.hostname,
		"running":         true,
		"current_ip":      currentIP,
		"current_index":   m.currentIdx,
		"all_ips":         allIPs,
		"ip_count":        len(allIPs),
		"update_interval": m.updateInterval.String(),
		"dns_servers":     m.dnsServers,
	}

	if err != nil {
		status["error"] = err.Error()
	}

	return status
}

// updateLoop 定时更新IP列表的主循环
func (m *Manager) updateLoop(ctx context.Context) {
	ticker := time.NewTicker(m.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Debugf(log.WebsocketMgr, "IP Manager update loop stopped due to context cancellation for %s", m.hostname)
			return
		case <-m.stopChan:
			log.Debugf(log.WebsocketMgr, "IP Manager update loop stopped for %s", m.hostname)
			return
		case <-ticker.C:
			log.Debugf(log.WebsocketMgr, "Scheduled IP update triggered for %s", m.hostname)
			if err := m.updateIPs(); err != nil {
				log.Errorf(log.WebsocketMgr, "Failed to update IP list for %s: %v", m.hostname, err)
			}
		case <-m.updateChan:
			log.Debugf(log.WebsocketMgr, "Manual IP update triggered for %s", m.hostname)
			if err := m.updateIPs(); err != nil {
				log.Errorf(log.WebsocketMgr, "Failed to update IP list for %s: %v", m.hostname, err)
			}
		}
	}
}

// updateIPs 更新IP列表
func (m *Manager) updateIPs() error {
	log.Debugf(log.WebsocketMgr, "Updating IP list for hostname: %s", m.hostname)

	var allIPs []string
	ipSet := make(map[string]bool) // 用于去重

	for _, dnsServer := range m.dnsServers {
		ips, err := m.resolveWithDNS(m.hostname, dnsServer)
		if err != nil {
			log.Warnf(log.WebsocketMgr, "Failed to resolve %s with DNS %s: %v", m.hostname, dnsServer, err)
			continue
		}

		// 去重并添加到列表
		for _, ip := range ips {
			if !ipSet[ip] {
				ipSet[ip] = true
				allIPs = append(allIPs, ip)
			}
		}
	}

	if len(allIPs) == 0 {
		return fmt.Errorf("failed to resolve any IPs for hostname: %s", m.hostname)
	}

	// 更新IP列表
	m.mu.Lock()
	oldIPs := m.ips
	m.ips = allIPs

	// 如果当前索引超出范围，重置为0
	if m.currentIdx >= len(m.ips) {
		m.currentIdx = 0
	}
	m.mu.Unlock()

	log.Infof(log.WebsocketMgr, "Updated IP list for %s: %v (previous: %v)",
		m.hostname, allIPs, oldIPs)

	return nil
}

// resolveWithDNS 使用指定的DNS服务器解析域名
func (m *Manager) resolveWithDNS(hostname, dnsServer string) ([]string, error) {
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: time.Second * 3,
			}
			return d.DialContext(ctx, network, dnsServer)
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), m.dnsTimeout)
	defer cancel()

	ips, err := resolver.LookupIPAddr(ctx, hostname)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, ip := range ips {
		// 只使用IPv4地址
		if ip.IP.To4() != nil {
			result = append(result, ip.IP.String())
		}
	}

	return result, nil
}
