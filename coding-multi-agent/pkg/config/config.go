package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type ModelConfig struct {
	ModelID    string `yaml:"model_id" json:"model_id"`
	Location   string `yaml:"location" json:"location"`
	IsThinking bool   `yaml:"is_thinking" json:"is_thinking"`
}

type RedisConfig struct {
	Host string `yaml:"host" json:"host"`
	Port int    `yaml:"port" json:"port"`
}

type MCPClientConfig struct {
	Endpoint            string `yaml:"endpoint" json:"endpoint"`
	AuthType            string `yaml:"auth_type" json:"auth_type"`
	HeaderKey           string `yaml:"header_key,omitempty" json:"header_key,omitempty"`
	UserProjectOverride string `yaml:"user_project_override,omitempty" json:"user_project_override,omitempty"`
}

type Config struct {
	Models     map[string]ModelConfig     `yaml:"models" json:"models"`
	Storage    struct {
		Redis RedisConfig `yaml:"redis" json:"redis"`
	} `yaml:"storage" json:"storage"`
	MCPClients map[string]MCPClientConfig `yaml:"mcp_clients" json:"mcp_clients"`
}

func Load(path string) (*Config, error) {
	config := &Config{}

	// 1. Load from file if exists
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			// If file is specified but fails to read, return error
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// 2. Override with Environment Variables (Basic overrides)
	if redisHost := os.Getenv("REDIS_HOST"); redisHost != "" {
		config.Storage.Redis.Host = redisHost
	}

	// Initialize maps if nil (in case file wasn't loaded)
	if config.Models == nil {
		config.Models = make(map[string]ModelConfig)
	}
	if config.MCPClients == nil {
		config.MCPClients = make(map[string]MCPClientConfig)
	}

	// Set Defaults if not present (Optional, can be expanded)
	// Example: Default Redis Port
	if config.Storage.Redis.Port == 0 {
		config.Storage.Redis.Port = 6379
	}

	return config, nil
}
