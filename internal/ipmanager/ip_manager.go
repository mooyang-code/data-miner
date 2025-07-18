// Package ipmanager 提供通用的IP地址管理功能
// 支持动态DNS解析、故障转移和自动更新
package ipmanager

import (
	"context"
	"fmt"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/mooyang-code/data-miner/pkg/cryptotrader/log"
)

// IPInfo 存储IP地址及其延迟信息
type IPInfo struct {
	IP        string        // IP地址
	Latency   time.Duration // 网络延迟
	LastPing  time.Time     // 最后一次ping时间
	Available bool          // 是否可用
}

// Manager 管理域名对应的IP地址列表
type Manager struct {
	mu         sync.RWMutex
	ips        []string  // 保持向后兼容
	ipInfos    []*IPInfo // 新的IP信息列表
	currentIdx int
	hostname   string
	updateChan chan struct{}
	stopChan   chan struct{}
	isRunning  bool

	// 配置选项
	updateInterval time.Duration
	dnsServers     []string
	dnsTimeout     time.Duration

	// 延迟检测配置
	enableLatencyCheck   bool          // 是否启用延迟检测
	latencyCheckInterval time.Duration // 延迟检测间隔
	latencyTimeout       time.Duration // 延迟检测超时
	latencyPort          string        // 用于延迟检测的端口
}

// Config IP管理器配置
type Config struct {
	Hostname       string        // 要解析的域名
	UpdateInterval time.Duration // 更新间隔，默认5分钟
	DNSServers     []string      // DNS服务器列表
	DNSTimeout     time.Duration // DNS查询超时时间，默认5秒

	// 延迟检测配置
	EnableLatencyCheck   bool          // 是否启用延迟检测，默认true
	LatencyCheckInterval time.Duration // 延迟检测间隔，默认30秒
	LatencyTimeout       time.Duration // 延迟检测超时，默认3秒
	LatencyPort          string        // 用于延迟检测的端口，默认443
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

		// 延迟检测默认配置
		EnableLatencyCheck:   true,
		LatencyCheckInterval: 60 * time.Second, // 增加检测间隔，减少干扰
		LatencyTimeout:       2 * time.Second,  // 减少超时时间
		LatencyPort:          "80",             // HTTP端口，避免与HTTPS请求冲突
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
	if config.LatencyCheckInterval == 0 {
		config.LatencyCheckInterval = 30 * time.Second
	}
	if config.LatencyTimeout == 0 {
		config.LatencyTimeout = 3 * time.Second
	}
	if config.LatencyPort == "" {
		config.LatencyPort = "80"
	}

	return &Manager{
		hostname:             config.Hostname,
		ips:                  make([]string, 0),
		ipInfos:              make([]*IPInfo, 0),
		updateChan:           make(chan struct{}, 1),
		stopChan:             make(chan struct{}),
		updateInterval:       config.UpdateInterval,
		dnsServers:           config.DNSServers,
		dnsTimeout:           config.DNSTimeout,
		enableLatencyCheck:   config.EnableLatencyCheck,
		latencyCheckInterval: config.LatencyCheckInterval,
		latencyTimeout:       config.LatencyTimeout,
		latencyPort:          config.LatencyPort,
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

	// 如果启用延迟检测，启动延迟检测协程
	if m.enableLatencyCheck {
		go m.latencyCheckLoop(ctx)
		log.Infof(log.WebsocketMgr, "Latency check enabled for hostname: %s", m.hostname)
	}

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

// GetCurrentIP 获取当前可用的IP地址（优先返回延迟最低的IP）
func (m *Manager) GetCurrentIP() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.ips) == 0 {
		return "", fmt.Errorf("no available IPs for hostname: %s", m.hostname)
	}

	// 如果启用了延迟检测且有延迟信息，返回延迟最低的可用IP
	if m.enableLatencyCheck && len(m.ipInfos) > 0 {
		for _, ipInfo := range m.ipInfos {
			if ipInfo.Available {
				log.Debugf(log.WebsocketMgr, "Using best latency IP: %s (latency: %v) for %s",
					ipInfo.IP, ipInfo.Latency, m.hostname)
				return ipInfo.IP, nil
			}
		}
	}

	// 回退到传统方式
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
		"hostname":              m.hostname,
		"running":               true,
		"current_ip":            currentIP,
		"current_index":         m.currentIdx,
		"all_ips":               allIPs,
		"ip_count":              len(allIPs),
		"update_interval":       m.updateInterval.String(),
		"dns_servers":           m.dnsServers,
		"latency_check_enabled": m.enableLatencyCheck,
	}

	// 添加延迟信息
	if m.enableLatencyCheck && len(m.ipInfos) > 0 {
		latencyInfo := make([]map[string]interface{}, 0, len(m.ipInfos))
		for _, ipInfo := range m.ipInfos {
			info := map[string]interface{}{
				"ip":        ipInfo.IP,
				"latency":   ipInfo.Latency.String(),
				"available": ipInfo.Available,
				"last_ping": ipInfo.LastPing.Format("2006-01-02 15:04:05"),
			}
			latencyInfo = append(latencyInfo, info)
		}
		status["latency_info"] = latencyInfo
		status["latency_check_interval"] = m.latencyCheckInterval.String()
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

		// 处理解析到的IP列表
		m.processResolvedIPs(ips, ipSet, &allIPs)
	}
	if len(allIPs) == 0 {
		log.Warnf(log.WebsocketMgr, "!!! Failed to resolve any valid IPs for %s, trying fallback IPs", m.hostname)

		// 使用已知的Binance API IP作为备用
		fallbackIPs := m.getFallbackIPs()
		if len(fallbackIPs) > 0 {
			allIPs = fallbackIPs
			log.Infof(log.WebsocketMgr, "Using fallback IPs for %s: %v", m.hostname, allIPs)
		} else {
			return fmt.Errorf("failed to resolve any IPs for hostname: %s", m.hostname)
		}
	}

	// 更新IP列表
	m.mu.Lock()
	oldIPs := m.ips
	m.ips = allIPs

	// 更新ipInfos列表
	m.updateIPInfos(allIPs)

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
	log.Debugf(log.WebsocketMgr, "Resolving %s using DNS server %s", hostname, dnsServer)

	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: time.Second * 5, // 增加超时时间
			}
			// 强制使用指定的DNS服务器
			return d.DialContext(ctx, network, dnsServer)
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), m.dnsTimeout)
	defer cancel()

	ips, err := resolver.LookupIPAddr(ctx, hostname)
	if err != nil {
		log.Warnf(log.WebsocketMgr, "DNS resolution failed for %s using %s: %v", hostname, dnsServer, err)
		return nil, err
	}

	var result []string
	for _, ip := range ips {
		// 只使用IPv4地址
		if ip.IP.To4() != nil {
			ipStr := ip.IP.String()
			result = append(result, ipStr)
			log.Debugf(log.WebsocketMgr, "Resolved %s to %s using DNS %s", hostname, ipStr, dnsServer)
		}
	}

	// 验证解析结果的合理性
	if len(result) == 0 {
		return nil, fmt.Errorf("no IPv4 addresses found for %s using DNS %s", hostname, dnsServer)
	}

	log.Infof(log.WebsocketMgr, "Successfully resolved %s to %v using DNS %s", hostname, result, dnsServer)
	return result, nil
}

// processResolvedIPs 处理解析到的IP列表，去重、验证并添加到结果列表
func (m *Manager) processResolvedIPs(ips []string, ipSet map[string]bool, allIPs *[]string) {
	for _, ip := range ips {
		if !ipSet[ip] && m.isValidBinanceIP(ip) {
			ipSet[ip] = true
			*allIPs = append(*allIPs, ip)
			log.Debugf(log.WebsocketMgr, "Added valid IP %s for %s", ip, m.hostname)
		} else if ipSet[ip] {
			log.Debugf(log.WebsocketMgr, "Skipping duplicate IP %s", ip)
		} else {
			log.Warnf(log.WebsocketMgr, "Skipping invalid IP %s for %s", ip, m.hostname)
		}
	}
}

// isValidBinanceIP 验证IP地址是否可能属于Binance
func (m *Manager) isValidBinanceIP(ip string) bool {
	// 已知的一些不应该属于Binance的IP段
	invalidRanges := []string{
		"199.59.148.0/22", // Twitter
		"31.13.0.0/16",    // Facebook
		"157.240.0.0/16",  // Facebook
		"173.252.0.0/16",  // Facebook
		"69.63.176.0/20",  // Facebook
		"69.171.224.0/19", // Facebook
	}

	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return false
	}

	for _, cidr := range invalidRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ipAddr) {
			log.Warnf(log.WebsocketMgr, "IP %s appears to be in invalid range %s, skipping", ip, cidr)
			return false
		}
	}

	return true
}

// getFallbackIPs 获取备用IP地址列表
func (m *Manager) getFallbackIPs() []string {
	// 根据域名返回已知的备用IP地址
	switch m.hostname {
	case "api.binance.com":
		// 这些是通过可信DNS服务器解析到的已知Binance IP
		return []string{
			"13.32.33.215",   // CloudFront
			"13.226.67.225",  // CloudFront
			"54.230.155.168", // CloudFront
			"54.230.155.169", // CloudFront
		}
	case "stream.binance.com":
		return []string{
			"13.32.33.215",
			"13.226.67.225",
		}
	default:
		return nil
	}
}

// updateIPInfos 更新IP信息列表（调用时需要持有锁）
func (m *Manager) updateIPInfos(newIPs []string) {
	// 创建新的IP信息映射
	newIPInfos := make([]*IPInfo, 0, len(newIPs))
	existingIPs := make(map[string]*IPInfo)

	// 保存现有的IP信息
	for _, ipInfo := range m.ipInfos {
		existingIPs[ipInfo.IP] = ipInfo
	}

	// 为新IP列表创建或更新IP信息
	for _, ip := range newIPs {
		if existing, found := existingIPs[ip]; found {
			// 保留现有的延迟信息
			newIPInfos = append(newIPInfos, existing)
		} else {
			// 创建新的IP信息
			newIPInfos = append(newIPInfos, &IPInfo{
				IP:        ip,
				Latency:   time.Duration(0),
				LastPing:  time.Time{},
				Available: true, // 默认可用，等待延迟检测
			})
		}
	}

	m.ipInfos = newIPInfos

	// 如果启用延迟检测，立即触发一次延迟检测
	if m.enableLatencyCheck {
		go m.checkLatencyForAllIPs()
	}
}

// latencyCheckLoop 延迟检测主循环
func (m *Manager) latencyCheckLoop(ctx context.Context) {
	// 初始延迟，避免启动时立即检测
	time.Sleep(5 * time.Second)

	ticker := time.NewTicker(m.latencyCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Debugf(log.WebsocketMgr, "Latency check loop stopped due to context cancellation for %s", m.hostname)
			return
		case <-m.stopChan:
			log.Debugf(log.WebsocketMgr, "Latency check loop stopped for %s", m.hostname)
			return
		case <-ticker.C:
			log.Debugf(log.WebsocketMgr, "Scheduled latency check triggered for %s", m.hostname)
			// 在单独的goroutine中执行延迟检测，避免阻塞
			go m.checkLatencyForAllIPs()
		}
	}
}

// checkLatencyForAllIPs 检测所有IP的延迟
func (m *Manager) checkLatencyForAllIPs() {
	m.mu.RLock()
	ipInfos := make([]*IPInfo, len(m.ipInfos))
	copy(ipInfos, m.ipInfos)
	m.mu.RUnlock()

	if len(ipInfos) == 0 {
		return
	}

	log.Debugf(log.WebsocketMgr, "Checking latency for %d IPs of %s", len(ipInfos), m.hostname)

	// 使用带缓冲的channel控制并发数，避免过多连接
	semaphore := make(chan struct{}, 3) // 最多3个并发连接
	var wg sync.WaitGroup

	for _, ipInfo := range ipInfos {
		wg.Add(1)
		go func(info *IPInfo) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			latency, err := m.measureLatency(info.IP)

			m.mu.Lock()
			info.LastPing = time.Now()
			if err != nil {
				info.Available = false
				info.Latency = time.Duration(0)
				log.Debugf(log.WebsocketMgr, "IP %s is unavailable: %v", info.IP, err)
			} else {
				info.Available = true
				info.Latency = latency
				log.Debugf(log.WebsocketMgr, "IP %s latency: %v", info.IP, latency)
			}
			m.mu.Unlock()
		}(ipInfo)
	}
	wg.Wait()

	// 按延迟排序IP列表
	m.sortIPsByLatency()
}

// measureLatency 测量到指定IP的网络延迟
func (m *Manager) measureLatency(ip string) (time.Duration, error) {
	start := time.Now()

	// 创建专用的拨号器，避免与HTTP客户端冲突
	dialer := &net.Dialer{
		Timeout:   m.latencyTimeout,
		KeepAlive: -1, // 禁用keep-alive，避免连接复用冲突
	}

	// 使用TCP连接测试延迟
	conn, err := dialer.Dial("tcp", net.JoinHostPort(ip, m.latencyPort))
	if err != nil {
		return 0, err
	}

	// 立即关闭连接，避免影响后续请求
	conn.Close()

	latency := time.Since(start)
	return latency, nil
}

// sortIPsByLatency 按延迟对IP进行排序（调用时需要持有锁）
func (m *Manager) sortIPsByLatency() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.enableLatencyCheck || len(m.ipInfos) == 0 {
		return
	}

	// 按延迟排序，可用的IP优先，然后按延迟从低到高排序
	sort.Slice(m.ipInfos, func(i, j int) bool {
		ipA, ipB := m.ipInfos[i], m.ipInfos[j]

		// 可用的IP优先
		if ipA.Available != ipB.Available {
			return ipA.Available
		}

		// 如果都可用，按延迟排序
		if ipA.Available && ipB.Available {
			return ipA.Latency < ipB.Latency
		}

		// 如果都不可用，保持原有顺序
		return false
	})

	// 更新传统的ips列表以保持兼容性
	newIPs := make([]string, 0, len(m.ipInfos))
	for _, ipInfo := range m.ipInfos {
		newIPs = append(newIPs, ipInfo.IP)
	}
	m.ips = newIPs

	// 重置当前索引
	m.currentIdx = 0

	if len(m.ipInfos) > 0 && m.ipInfos[0].Available {
		log.Infof(log.WebsocketMgr, "Best IP for %s: %s (latency: %v)",
			m.hostname, m.ipInfos[0].IP, m.ipInfos[0].Latency)
	}
}

// GetBestIP 获取延迟最低的可用IP
func (m *Manager) GetBestIP() (string, time.Duration, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.enableLatencyCheck || len(m.ipInfos) == 0 {
		// 回退到传统方式
		if len(m.ips) == 0 {
			return "", 0, fmt.Errorf("no available IPs for hostname: %s", m.hostname)
		}
		return m.ips[m.currentIdx], 0, nil
	}

	for _, ipInfo := range m.ipInfos {
		if ipInfo.Available {
			return ipInfo.IP, ipInfo.Latency, nil
		}
	}

	return "", 0, fmt.Errorf("no available IPs for hostname: %s", m.hostname)
}

// GetAllIPsWithLatency 获取所有IP及其延迟信息
func (m *Manager) GetAllIPsWithLatency() []*IPInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.enableLatencyCheck {
		return nil
	}

	// 返回副本以避免并发修改
	result := make([]*IPInfo, len(m.ipInfos))
	for i, ipInfo := range m.ipInfos {
		result[i] = &IPInfo{
			IP:        ipInfo.IP,
			Latency:   ipInfo.Latency,
			LastPing:  ipInfo.LastPing,
			Available: ipInfo.Available,
		}
	}
	return result
}

// ForceLatencyCheck 强制执行一次延迟检测
func (m *Manager) ForceLatencyCheck() {
	if !m.enableLatencyCheck {
		log.Warnf(log.WebsocketMgr, "Latency check is disabled for %s", m.hostname)
		return
	}

	log.Debugf(log.WebsocketMgr, "Forced latency check requested for %s", m.hostname)
	go m.checkLatencyForAllIPs()
}
