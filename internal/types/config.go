// Package types 定义数据挖掘器的配置类型
package types

import "time"

// Config 主配置结构
type Config struct {
	App        AppConfig        `yaml:"app"`        // 应用配置
	Database   DatabaseConfig   `yaml:"database"`   // 数据库配置
	Exchanges  ExchangesConfig  `yaml:"exchanges"`  // 交易所配置
	Scheduler  SchedulerConfig  `yaml:"scheduler"`  // 调度器配置
	Storage    StorageConfig    `yaml:"storage"`    // 存储配置
	Monitoring MonitoringConfig `yaml:"monitoring"` // 监控配置
}

// AppConfig 应用配置
type AppConfig struct {
	Name     string `yaml:"name"`      // 应用名称
	Version  string `yaml:"version"`   // 应用版本
	LogLevel string `yaml:"log_level"` // 日志级别
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Enabled  bool   `yaml:"enabled"`  // 是否启用数据库
	Driver   string `yaml:"driver"`   // 数据库驱动
	Host     string `yaml:"host"`     // 数据库主机
	Port     int    `yaml:"port"`     // 数据库端口
	Username string `yaml:"username"` // 用户名
	Password string `yaml:"password"` // 密码
	Database string `yaml:"database"` // 数据库名
}

// ExchangesConfig 交易所配置
type ExchangesConfig struct {
	Binance BinanceConfig `yaml:"binance"` // Binance交易所配置
}

// BinanceConfig Binance交易所配置
type BinanceConfig struct {
	Enabled       bool             `yaml:"enabled"`        // 是否启用
	APIURL        string           `yaml:"api_url"`        // API地址
	WebsocketURL  string           `yaml:"websocket_url"`  // WebSocket地址
	APIKey        string           `yaml:"api_key"`        // API密钥
	APISecret     string           `yaml:"api_secret"`     // API密钥
	UseWebsocket  bool             `yaml:"use_websocket"`  // 是否使用websocket模式
	DataTypes     BinanceDataTypes `yaml:"data_types"`     // 数据类型配置
	TradablePairs TradablePairsConfig `yaml:"tradable_pairs"` // 可交易交易对配置
}

// BinanceDataTypes Binance数据类型配置
type BinanceDataTypes struct {
	Ticker    TickerConfig    `yaml:"ticker"`    // 行情配置
	Orderbook OrderbookConfig `yaml:"orderbook"` // 订单簿配置
	Trades    TradesConfig    `yaml:"trades"`    // 交易配置
	Klines    KlinesConfig    `yaml:"klines"`    // K线配置
}

// TickerConfig 行情配置
type TickerConfig struct {
	Enabled  bool     `yaml:"enabled"`  // 是否启用
	Symbols  []string `yaml:"symbols"`  // 交易对列表
	Interval string   `yaml:"interval"` // 更新间隔
}

// OrderbookConfig 订单簿配置
type OrderbookConfig struct {
	Enabled  bool     `yaml:"enabled"`  // 是否启用
	Symbols  []string `yaml:"symbols"`  // 交易对列表
	Depth    int      `yaml:"depth"`    // 深度
	Interval string   `yaml:"interval"` // 更新间隔
}

// TradesConfig 交易数据配置
type TradesConfig struct {
	Enabled  bool     `yaml:"enabled"`  // 是否启用
	Symbols  []string `yaml:"symbols"`  // 交易对列表
	Interval string   `yaml:"interval"` // 更新间隔
}

// KlinesConfig K线数据配置
type KlinesConfig struct {
	Enabled   bool     `yaml:"enabled"`   // 是否启用
	Symbols   []string `yaml:"symbols"`   // 交易对列表
	Intervals []string `yaml:"intervals"` // 时间间隔列表
	Interval  string   `yaml:"interval"`  // 更新间隔
}

// TradablePairsConfig 可交易交易对配置
type TradablePairsConfig struct {
	FetchFromAPI       bool          `yaml:"fetch_from_api"`        // 是否从API获取交易对列表
	UpdateInterval     time.Duration `yaml:"update_interval"`       // 更新间隔
	CacheEnabled       bool          `yaml:"cache_enabled"`         // 是否启用缓存
	CacheTTL           time.Duration `yaml:"cache_ttl"`             // 缓存生存时间
	SupportedAssets    []string      `yaml:"supported_assets"`      // 支持的资产类型 ["spot", "margin"]
	AutoUpdate         bool          `yaml:"auto_update"`           // 是否自动更新
	SkipOnNetworkError bool          `yaml:"skip_on_network_error"` // 网络错误时是否跳过初始化
}

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	Enabled           bool        `yaml:"enabled"`             // 是否启用
	MaxConcurrentJobs int         `yaml:"max_concurrent_jobs"` // 最大并发任务数
	Jobs              []JobConfig `yaml:"jobs"`                // 任务列表
}

// JobConfig 任务配置
type JobConfig struct {
	Name     string `yaml:"name"`      // 任务名称
	Exchange string `yaml:"exchange"`  // 交易所名称
	DataType string `yaml:"data_type"` // 数据类型
	Cron     string `yaml:"cron"`      // Cron表达式
}

// StorageConfig 存储配置
type StorageConfig struct {
	File  FileStorageConfig  `yaml:"file"`  // 文件存储配置
	Cache CacheStorageConfig `yaml:"cache"` // 缓存存储配置
}

// FileStorageConfig 文件存储配置
type FileStorageConfig struct {
	Enabled  bool   `yaml:"enabled"`   // 是否启用
	BasePath string `yaml:"base_path"` // 基础路径
	Format   string `yaml:"format"`    // 文件格式
}

// CacheStorageConfig 缓存存储配置
type CacheStorageConfig struct {
	Enabled bool          `yaml:"enabled"`  // 是否启用
	MaxSize int           `yaml:"max_size"` // 最大大小
	TTL     time.Duration `yaml:"ttl"`      // 生存时间
}

// MonitoringConfig 监控配置
type MonitoringConfig struct {
	Enabled         bool `yaml:"enabled"`           // 是否启用
	MetricsPort     int  `yaml:"metrics_port"`      // 指标端口
	HealthCheckPort int  `yaml:"health_check_port"` // 健康检查端口
}
