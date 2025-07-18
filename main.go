package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/mooyang-code/data-miner/internal/app"
)

var (
	configPath = flag.String("config", "./config.yaml", "配置文件路径")
	version    = flag.Bool("version", false, "显示版本信息")
	help       = flag.Bool("help", false, "显示帮助信息")
)

func main() {
	// 解析命令行参数
	if shouldExit := parseFlags(); shouldExit {
		return
	}

	// 创建应用程序管理器
	manager := app.New()

	// 初始化应用程序
	if err := manager.Initialize(*configPath); err != nil {
		fmt.Printf("初始化应用程序失败: %v\n", err)
		os.Exit(1)
	}
	defer manager.Sync()

	// 启动应用程序
	if err := manager.Start(); err != nil {
		fmt.Printf("启动应用程序失败: %v\n", err)
		os.Exit(1)
	}
}

// parseFlags 解析命令行参数
func parseFlags() bool {
	flag.Parse()

	if *help {
		showHelp()
		return true
	}

	if *version {
		showVersion()
		return true
	}
	return false
}

// showHelp 显示帮助信息
func showHelp() {
	fmt.Println("加密货币数据采集器")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  data-miner [选项]")
	fmt.Println()
	fmt.Println("选项:")
	fmt.Println("  -config string")
	fmt.Println("        配置文件路径 (默认 \"config/config.yaml\")")
	fmt.Println("  -version")
	fmt.Println("        显示版本信息")
	fmt.Println("  -help")
	fmt.Println("        显示此帮助信息")
}

// showVersion 显示版本信息
func showVersion() {
	fmt.Println("加密货币数据采集器 v1.0.0")
	fmt.Println("构建时间: ", time.Now().Format("2006-01-02 15:04:05"))
}
