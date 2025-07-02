package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	RabbitMQ RabbitMQConfig `mapstructure:"rabbitmq"`
	App      AppConfig      `mapstructure:"app"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port string `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

// RabbitMQConfig holds RabbitMQ configuration
type RabbitMQConfig struct {
	URL      string `mapstructure:"url"`
	Queue    string `mapstructure:"queue"`
	Exchange string `mapstructure:"exchange"`
}

// AppConfig holds application-specific configuration
type AppConfig struct {
	DefaultBatchSize int `mapstructure:"default_batch_size"`
	MaxIPsPerBatch   int `mapstructure:"max_ips_per_batch"`
}

// LoadConfig reads configuration from file or environment variables
func LoadConfig(path string) (*Config, error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Read environment variables
	viper.AutomaticEnv()

	// Set defaults
	setDefaults()

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// setDefaults sets default values for configuration
func setDefaults() {
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.host", "localhost")
	viper.SetDefault("rabbitmq.url", "amqp://guest:guest@localhost:5672/")
	viper.SetDefault("rabbitmq.queue", "ip-scan-queue")
	viper.SetDefault("rabbitmq.exchange", "")
	viper.SetDefault("app.default_batch_size", 100)
	viper.SetDefault("app.max_ips_per_batch", 1000)
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	if config.RabbitMQ.URL == "" {
		return fmt.Errorf("rabbitmq URL is required")
	}
	if config.RabbitMQ.Queue == "" {
		return fmt.Errorf("rabbitmq queue name is required")
	}
	if config.App.DefaultBatchSize <= 0 {
		return fmt.Errorf("default batch size must be greater than 0")
	}
	if config.App.MaxIPsPerBatch <= 0 {
		return fmt.Errorf("max IPs per batch must be greater than 0")
	}
	return nil
}

// GetEnvOrDefault returns environment variable value or default
func GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
