// Package main 演示IP管理器的网络延迟检测功能
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mooyang-code/data-miner/internal/ipmanager"
)

func main() {
	fmt.Println("=== IP管理器网络延迟检测示例 ===")

	// 创建启用延迟检测的配置
	config := &ipmanager.Config{
		Hostname:             "api.binance.com",
		UpdateInterval:       5 * time.Minute,
		DNSServers:           []string{"8.8.8.8:53", "1.1.1.1:53"},
		DNSTimeout:           5 * time.Second,
		EnableLatencyCheck:   true,
		LatencyCheckInterval: 10 * time.Second, // 更频繁的检测用于演示
		LatencyTimeout:       2 * time.Second,  // 减少超时时间
		LatencyPort:          "80",             // HTTP端口，避免与HTTPS冲突
	}

	// 创建IP管理器
	manager := ipmanager.New(config)

	// 启动管理器
	ctx := context.Background()
	err := manager.Start(ctx)
	if err != nil {
		log.Fatalf("启动IP管理器失败: %v", err)
	}
	defer manager.Stop()

	fmt.Println("IP管理器已启动，等待初始化...")
	time.Sleep(3 * time.Second)

	// 显示初始状态
	fmt.Println("\n=== 初始状态 ===")
	showStatus(manager)

	// 等待第一次延迟检测完成
	fmt.Println("\n等待延迟检测完成...")
	time.Sleep(5 * time.Second)

	// 显示延迟检测后的状态
	fmt.Println("\n=== 延迟检测后状态 ===")
	showStatus(manager)

	// 演示获取最佳IP
	fmt.Println("\n=== 获取最佳IP ===")
	bestIP, latency, err := manager.GetBestIP()
	if err != nil {
		fmt.Printf("获取最佳IP失败: %v\n", err)
	} else {
		fmt.Printf("最佳IP: %s (延迟: %v)\n", bestIP, latency)
	}

	// 演示获取当前IP（会自动选择延迟最低的）
	fmt.Println("\n=== 获取当前IP ===")
	currentIP, err := manager.GetCurrentIP()
	if err != nil {
		fmt.Printf("获取当前IP失败: %v\n", err)
	} else {
		fmt.Printf("当前IP: %s\n", currentIP)
	}

	// 显示所有IP的延迟信息
	fmt.Println("\n=== 所有IP延迟信息 ===")
	ipInfos := manager.GetAllIPsWithLatency()
	if ipInfos == nil {
		fmt.Println("延迟检测未启用")
	} else {
		for i, info := range ipInfos {
			status := "可用"
			if !info.Available {
				status = "不可用"
			}
			fmt.Printf("%d. IP: %s, 延迟: %v, 状态: %s, 最后检测: %s\n",
				i+1, info.IP, info.Latency, status, info.LastPing.Format("15:04:05"))
		}
	}

	// 强制执行一次延迟检测
	fmt.Println("\n=== 强制延迟检测 ===")
	manager.ForceLatencyCheck()
	time.Sleep(3 * time.Second)

	// 再次显示状态
	fmt.Println("\n=== 强制检测后状态 ===")
	showStatus(manager)

	// 持续监控一段时间
	fmt.Println("\n=== 持续监控 (30秒) ===")
	fmt.Println("监控IP变化和延迟更新...")
	
	for i := 0; i < 6; i++ {
		time.Sleep(5 * time.Second)
		
		currentIP, err := manager.GetCurrentIP()
		if err != nil {
			fmt.Printf("[%ds] 获取IP失败: %v\n", (i+1)*5, err)
			continue
		}
		
		bestIP, latency, err := manager.GetBestIP()
		if err != nil {
			fmt.Printf("[%ds] 当前IP: %s (无延迟信息)\n", (i+1)*5, currentIP)
		} else {
			fmt.Printf("[%ds] 当前IP: %s, 最佳IP: %s (延迟: %v)\n", 
				(i+1)*5, currentIP, bestIP, latency)
		}
	}

	fmt.Println("\n=== 示例完成 ===")
}

// showStatus 显示IP管理器状态
func showStatus(manager *ipmanager.Manager) {
	status := manager.GetStatus()
	
	fmt.Printf("域名: %s\n", status["hostname"])
	fmt.Printf("运行状态: %v\n", status["running"])
	fmt.Printf("当前IP: %s\n", status["current_ip"])
	fmt.Printf("IP数量: %v\n", status["ip_count"])
	fmt.Printf("延迟检测: %v\n", status["latency_check_enabled"])
	
	if latencyInfo, ok := status["latency_info"]; ok {
		fmt.Println("延迟信息:")
		if infos, ok := latencyInfo.([]map[string]interface{}); ok {
			for i, info := range infos {
				fmt.Printf("  %d. IP: %s, 延迟: %s, 可用: %v, 最后检测: %s\n",
					i+1, info["ip"], info["latency"], info["available"], info["last_ping"])
			}
		}
	}
	
	if allIPs, ok := status["all_ips"]; ok {
		fmt.Printf("所有IP: %v\n", allIPs)
	}
}
