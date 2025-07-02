package config

import (
	"fmt"
	"time"

	"port-scanner/internal/domain"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	RabbitMQ RabbitMQConfig `mapstructure:"rabbitmq"`
	Scan     ScanConfig     `mapstructure:"scan"`
	MongoDB  MongoDBConfig  `mapstructure:"mongodb"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port string `mapstructure:"port"`
}

// RabbitMQConfig represents RabbitMQ configuration
type RabbitMQConfig struct {
	URL                  string `mapstructure:"url"`
	IPQueue              string `mapstructure:"ip_queue"`
	ScanResultQueue      string `mapstructure:"scan_result_queue"`
	EnrichmentQueue      string `mapstructure:"enrichment_queue"`
	ServiceAnalysisQueue string `mapstructure:"service_analysis_queue"`
}

// MongoDBConfig represents MongoDB configuration
type MongoDBConfig struct {
	ConnectionString string `mapstructure:"connection_string"`
	DatabaseName     string `mapstructure:"database_name"`
	CollectionName   string `mapstructure:"collection_name"`
	EnableDatabase   bool   `mapstructure:"enable_database"`
}

// ScanConfig represents scan configuration
type ScanConfig struct {
	PingTimeout      string `mapstructure:"ping_timeout"`
	ConnectTimeout   string `mapstructure:"connect_timeout"`
	BannerTimeout    string `mapstructure:"banner_timeout"`
	MaxRetries       int    `mapstructure:"max_retries"`
	RetryDelay       string `mapstructure:"retry_delay"`
	Concurrency      int    `mapstructure:"concurrency"`
	ZGrabConcurrency int    `mapstructure:"zgrab_concurrency"`
	EnableBanner     bool   `mapstructure:"enable_banner"`
	EnablePing       bool   `mapstructure:"enable_ping"`
	PriorityPorts    []int  `mapstructure:"priority_ports"`
}

// LoadConfig loads configuration from file and environment
func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configPath)

	// Set defaults
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", "8080")

	viper.SetDefault("rabbitmq.url", "amqp://guest:guest@localhost:5672/")
	viper.SetDefault("rabbitmq.ip_queue", "ip_queue")
	viper.SetDefault("rabbitmq.scan_result_queue", "scan_result_queue")
	viper.SetDefault("rabbitmq.enrichment_queue", "enrichment_queue")
	viper.SetDefault("rabbitmq.service_analysis_queue", "service_analysis_queue")

	viper.SetDefault("mongodb.connection_string", "mongodb://localhost:27017")
	viper.SetDefault("mongodb.database_name", "solomon")
	viper.SetDefault("mongodb.collection_name", "scan_results")
	viper.SetDefault("mongodb.enable_database", true)

	viper.SetDefault("scan.ping_timeout", "5s")
	viper.SetDefault("scan.connect_timeout", "3s")
	viper.SetDefault("scan.banner_timeout", "2s")
	viper.SetDefault("scan.max_retries", 3)
	viper.SetDefault("scan.retry_delay", "1s")
	viper.SetDefault("scan.concurrency", 100)
	viper.SetDefault("scan.zgrab_concurrency", 20)
	viper.SetDefault("scan.enable_banner", true)
	viper.SetDefault("scan.enable_ping", true)
	viper.SetDefault("scan.priority_ports", []int{80, 443, 22, 21, 25, 3306, 5432})

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// ToDomainScanConfig converts Config to domain.ScanConfig
func (c *Config) ToDomainScanConfig() *domain.ScanConfig {
	pingTimeout, _ := time.ParseDuration(c.Scan.PingTimeout)
	connectTimeout, _ := time.ParseDuration(c.Scan.ConnectTimeout)
	bannerTimeout, _ := time.ParseDuration(c.Scan.BannerTimeout)
	retryDelay, _ := time.ParseDuration(c.Scan.RetryDelay)

	return &domain.ScanConfig{
		PingTimeout:      pingTimeout,
		ConnectTimeout:   connectTimeout,
		BannerTimeout:    bannerTimeout,
		MaxRetries:       c.Scan.MaxRetries,
		RetryDelay:       retryDelay,
		Concurrency:      c.Scan.Concurrency,
		ZGrabConcurrency: c.Scan.ZGrabConcurrency,
		DefaultPorts:     []int{21, 22, 23, 25, 53, 80, 110, 143, 443, 993, 995, 3306, 3389, 5432, 8080, 8443},
		PriorityPorts:    c.Scan.PriorityPorts,
		EnableBanner:     c.Scan.EnableBanner,
		EnablePing:       c.Scan.EnablePing,
	}
}
