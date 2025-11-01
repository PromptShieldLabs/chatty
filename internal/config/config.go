package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	envAPIKey = "GOAICHAT_API_KEY"
	envAPIURL = "GOAICHAT_API_URL"
)

// Config captures runtime configuration for the Goaichat application.
type Config struct {
	API     APIConfig     `yaml:"api"`
	Model   ModelConfig   `yaml:"model"`
	Logging LoggingConfig `yaml:"logging"`
	UI      UIConfig      `yaml:"ui"`
}

// APIConfig holds settings for connecting to the OpenAI-compatible API.
type APIConfig struct {
	URL string `yaml:"url"`
	Key string `yaml:"key"`
}

// ModelConfig controls default model behaviour.
type ModelConfig struct {
	Name        string  `yaml:"name"`
	Temperature float64 `yaml:"temperature"`
	Stream      bool    `yaml:"stream"`
}

// LoggingConfig encapsulates logging preferences.
type LoggingConfig struct {
	Level string `yaml:"level"`
}

// UIConfig defines terminal rendering preferences.
type UIConfig struct {
	ShowTimestamps bool `yaml:"show_timestamps"`
}

// Load reads configuration from the provided path, falling back to defaults and
// environment overrides.
func Load(path string) (*Config, error) {
	cfg := defaultConfig()

	if path != "" {
		if err := loadFile(path, &cfg); err != nil {
			return nil, err
		}
	} else {
		if err := loadFile("config.yaml", &cfg); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}

	applyEnvOverrides(&cfg)

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func loadFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return err
		}
		return fmt.Errorf("read config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	return nil
}

func applyEnvOverrides(cfg *Config) {
	if url := strings.TrimSpace(os.Getenv(envAPIURL)); url != "" {
		cfg.API.URL = url
	}
	if key := strings.TrimSpace(os.Getenv(envAPIKey)); key != "" {
		cfg.API.Key = key
	}
}

func (c *Config) validate() error {
	if strings.TrimSpace(c.API.URL) == "" {
		return errors.New("api.url must be set")
	}
	if strings.TrimSpace(c.API.Key) == "" {
		return errors.New("api.key must be set or GOAICHAT_API_KEY provided")
	}
	if c.Model.Temperature < 0 || c.Model.Temperature > 2 {
		return fmt.Errorf("model.temperature must be between 0 and 2, got %f", c.Model.Temperature)
	}
	return nil
}

func defaultConfig() Config {
	return Config{
		API: APIConfig{
			URL: "https://api.promptshield.io/v1",
		},
		Model: ModelConfig{
			Name:        "groq/moonshotai/kimi-k2-instruct-0905",
			Temperature: 0.7,
			Stream:      true,
		},
		Logging: LoggingConfig{
			Level: "info",
		},
		UI: UIConfig{
			ShowTimestamps: true,
		},
	}
}
