package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	APIKey        string            `mapstructure:"api_key"`
	AthleteID     string            `mapstructure:"athlete_id"`
	DefaultOutput string            `mapstructure:"default_output"`
	Units         string            `mapstructure:"units"`
	SportColors   map[string]string `mapstructure:"sport_colors"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	cfgDir, err := configDir()
	if err != nil {
		return nil, err
	}
	viper.AddConfigPath(cfgDir)

	viper.SetEnvPrefix("ICU")
	viper.AutomaticEnv()

	// env var mappings
	_ = viper.BindEnv("api_key", "ICU_API_KEY")
	_ = viper.BindEnv("athlete_id", "ICU_ATHLETE_ID")
	_ = viper.BindEnv("default_output", "ICU_OUTPUT")

	// defaults
	viper.SetDefault("athlete_id", "0")
	viper.SetDefault("default_output", "auto")
	viper.SetDefault("units", "metric")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config: %w", err)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}

func Save(cfg *Config) error {
	dir, err := configDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	viper.Set("api_key", cfg.APIKey)
	viper.Set("athlete_id", cfg.AthleteID)
	viper.Set("default_output", cfg.DefaultOutput)
	viper.Set("units", cfg.Units)

	path := filepath.Join(dir, "config.yaml")
	return viper.WriteConfigAs(path)
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home dir: %w", err)
	}
	return filepath.Join(home, ".config", "icu"), nil
}

func ConfigFilePath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}
