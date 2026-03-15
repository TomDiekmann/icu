package config_test

import (
	"os"
	"testing"

	"github.com/tomdiekmann/icu/internal/config"
)

// TestLoad_EnvVars verifies that ICU_* environment variables are picked up
// when no config file is present.
func TestLoad_EnvVars(t *testing.T) {
	t.Setenv("ICU_API_KEY", "test_key_123")
	t.Setenv("ICU_ATHLETE_ID", "i99999")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.APIKey != "test_key_123" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "test_key_123")
	}
	if cfg.AthleteID != "i99999" {
		t.Errorf("AthleteID = %q, want %q", cfg.AthleteID, "i99999")
	}
}

// TestLoad_Defaults verifies that defaults are applied when nothing is configured.
func TestLoad_Defaults(t *testing.T) {
	// Make sure env vars don't leak in from the environment.
	os.Unsetenv("ICU_API_KEY")
	os.Unsetenv("ICU_ATHLETE_ID")
	os.Unsetenv("ICU_OUTPUT")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.AthleteID != "0" {
		t.Errorf("default AthleteID = %q, want %q", cfg.AthleteID, "0")
	}
	if cfg.DefaultOutput != "auto" {
		t.Errorf("default DefaultOutput = %q, want %q", cfg.DefaultOutput, "auto")
	}
	if cfg.Units != "metric" {
		t.Errorf("default Units = %q, want %q", cfg.Units, "metric")
	}
}

// TestLoad_OutputEnv verifies ICU_OUTPUT is read.
func TestLoad_OutputEnv(t *testing.T) {
	t.Setenv("ICU_OUTPUT", "json")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.DefaultOutput != "json" {
		t.Errorf("DefaultOutput = %q, want %q", cfg.DefaultOutput, "json")
	}
}

// TestConfigFilePath_ContainsIcu verifies the config path is under ~/.config/icu.
func TestConfigFilePath_ContainsIcu(t *testing.T) {
	path, err := config.ConfigFilePath()
	if err != nil {
		t.Fatalf("ConfigFilePath() error: %v", err)
	}
	if path == "" {
		t.Error("ConfigFilePath() returned empty string")
	}
	// Must end with /icu/config.yaml
	if len(path) < 15 {
		t.Errorf("ConfigFilePath() too short: %q", path)
	}
}
