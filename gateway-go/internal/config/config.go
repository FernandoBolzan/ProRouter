package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Dashboard DashboardConfig `yaml:"dashboard"`
	Providers ProvidersConfig `yaml:"providers"`
	LogLevel string `yaml:"log_level"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type DatabaseConfig struct {
	Path    string `yaml:"path"`
	WALMode bool   `yaml:"wal_mode"`
}

type DashboardConfig struct {
	Enabled bool   `yaml:"enabled"`
	Theme   string `yaml:"theme"`
}

type ProvidersConfig struct {
	OpenAI    ProviderAuth `yaml:"openai"`
	Anthropic ProviderAuth `yaml:"anthropic"`
	Google    ProviderAuth `yaml:"google"`
	DeepSeek  ProviderAuth `yaml:"deepseek"`
	Local     LocalConfig  `yaml:"local"`
}

type ProviderAuth struct {
	APIKey string `yaml:"api_key"`
}

type LocalConfig struct {
	ScanPorts bool   `yaml:"scan_ports"`
	Ports     []int  `yaml:"ports"`
}

func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		Database: DatabaseConfig{
			Path:    filepath.Join(home, ".prorouter", "data", "prorouter.db"),
			WALMode: true,
		},
		Dashboard: DashboardConfig{
			Enabled: true,
			Theme:   "dark",
		},
		Providers: ProvidersConfig{
			Local: LocalConfig{
				ScanPorts: true,
				Ports:     []int{11434, 8000, 1234},
			},
		},
		LogLevel: "info",
	}
}

func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Expand env vars in provider keys
	resolveEnv(&cfg.Providers.OpenAI.APIKey)
	resolveEnv(&cfg.Providers.Anthropic.APIKey)
	resolveEnv(&cfg.Providers.Google.APIKey)
	resolveEnv(&cfg.Providers.DeepSeek.APIKey)

	return cfg, nil
}

func resolveEnv(val *string) {
	if len(*val) > 2 && (*val)[0] == '$' && (*val)[1] == '{' {
		envName := (*val)[2 : len(*val)-1]
		if envVal := os.Getenv(envName); envVal != "" {
			*val = envVal
		}
	}
}
