package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mooyang-code/data-miner/internal/types"
	"gopkg.in/yaml.v3"
)

// LoadConfig 从YAML文件加载配置
func LoadConfig(configPath string) (*types.Config, error) {
	// 如果未指定配置文件路径，使用默认路径
	if configPath == "" {
		configPath = "config/config.yaml"
	}

	// 检查文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("配置文件不存在: %s", configPath)
	}

	// 读取文件内容
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	// 解析YAML
	var config types.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	// 验证配置
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("配置验证失败: %v", err)
	}

	return &config, nil
}

// validateConfig 验证配置的有效性
func validateConfig(config *types.Config) error {
	// 验证应用配置
	if config.App.Name == "" {
		return fmt.Errorf("应用名称不能为空")
	}

	// 验证交易所配置
	if config.Exchanges.Binance.Enabled {
		if config.Exchanges.Binance.APIURL == "" {
			return fmt.Errorf("Binance API URL不能为空")
		}
		if config.Exchanges.Binance.WebsocketURL == "" {
			return fmt.Errorf("Binance WebSocket URL不能为空")
		}
	}

	// 验证存储配置
	if config.Storage.File.Enabled && config.Storage.File.BasePath == "" {
		return fmt.Errorf("文件存储路径不能为空")
	}

	// 验证调度器配置
	if config.Scheduler.Enabled {
		if config.Scheduler.MaxConcurrentJobs <= 0 {
			return fmt.Errorf("最大并发任务数必须大于0")
		}

		// 验证任务配置
		for i, job := range config.Scheduler.Jobs {
			if job.Name == "" {
				return fmt.Errorf("第%d个任务名称不能为空", i+1)
			}
			if job.Exchange == "" {
				return fmt.Errorf("第%d个任务的交易所不能为空", i+1)
			}
			if job.DataType == "" {
				return fmt.Errorf("第%d个任务的数据类型不能为空", i+1)
			}
			if job.Cron == "" {
				return fmt.Errorf("第%d个任务的Cron表达式不能为空", i+1)
			}
		}
	}

	return nil
}

// SaveConfig 保存配置到文件
func SaveConfig(config *types.Config, configPath string) error {
	// 确保目录存在
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %v", err)
	}

	// 序列化为YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	// 写入文件
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}

	return nil
}

// GetDefaultConfig 获取默认配置
func GetDefaultConfig() *types.Config {
	return &types.Config{
		App: types.AppConfig{
			Name:     "crypto-data-miner",
			Version:  "1.0.0",
			LogLevel: "info",
		},
		Exchanges: types.ExchangesConfig{
			Binance: types.BinanceConfig{
				Enabled:      true,
				APIURL:       "https://api.binance.com",
				WebsocketURL: "wss://stream.binance.com:9443",
			},
		},
		Scheduler: types.SchedulerConfig{
			Enabled:           true,
			MaxConcurrentJobs: 10,
		},
		Storage: types.StorageConfig{
			File: types.FileStorageConfig{
				Enabled:  true,
				BasePath: "./data",
				Format:   "json",
			},
		},
		Monitoring: types.MonitoringConfig{
			Enabled:         true,
			MetricsPort:     8080,
			HealthCheckPort: 8081,
		},
	}
}
