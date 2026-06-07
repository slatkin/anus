package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	ApiKey                 string   `toml:"api_key"                  json:"api_key"`
	ServerUrl              string   `toml:"server_url"               json:"server_url"`
	AllowInvalidCerts      bool     `toml:"allow_invalid_certs"      json:"allow_invalid_certs"`
	PollingIntervalMinutes int      `toml:"polling_interval_minutes" json:"polling_interval_minutes"`
	CacheExpiryDays        int      `toml:"cache_expiry_days"        json:"cache_expiry_days"`
	CacheDir               string   `toml:"-"                        json:"-"`
}

func DefaultConfig() Config {
	return Config{
		ApiKey:               "FIXME",
		ServerUrl:            "FIXME",
		AllowInvalidCerts:    false,
		CacheExpiryDays:        30,
		PollingIntervalMinutes: 10,
	}
}

func GetConfigFilepath() (string, error) {
	if d := os.Getenv("DATA_DIR"); d != "" {
		return filepath.Join(d, "config.toml"), nil
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("could not find user config dir: %w", err)
	}
	return filepath.Join(configDir, "anus", "config.toml"), nil
}

func Init() (string, error) {
	path, err := GetConfigFilepath()
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("configuration file already exists at %s", path)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(DefaultConfig()); err != nil {
		return "", err
	}

	return path, nil
}

// Load builds config by starting with defaults, applying env vars, then
// overlaying config.toml on top (if it exists). Config file values win over
// env vars, so settings saved via the UI always take precedence.
func Load() (Config, error) {
	cfg := DefaultConfig()

	// Apply env vars.
	if v := os.Getenv("MINIFLUX_API_KEY"); v != "" {
		cfg.ApiKey = v
	}
	if v := os.Getenv("MINIFLUX_URL"); v != "" {
		cfg.ServerUrl = v
	}
	if v := os.Getenv("ALLOW_INVALID_CERTS"); v != "" {
		cfg.AllowInvalidCerts, _ = strconv.ParseBool(v)
	}
	if v := os.Getenv("CACHE_EXPIRY_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.CacheExpiryDays = n
		}
	}
	if v := os.Getenv("CACHE_DIR"); v != "" {
		cfg.CacheDir = v
	}

	// Overlay config file on top if it exists.
	if path, err := GetConfigFilepath(); err == nil {
		if _, err := os.Stat(path); err == nil {
			if _, err := toml.DecodeFile(path, &cfg); err != nil {
				return Config{}, fmt.Errorf("error parsing config file: %w", err)
			}
		}
	}

	return validate(cfg)
}

func validate(cfg Config) (Config, error) {
	if cfg.ApiKey == "" || cfg.ApiKey == "FIXME" {
		return Config{}, fmt.Errorf("api_key is not configured (set MINIFLUX_API_KEY or api_key in config.toml)")
	}

	cfg.ServerUrl = strings.TrimSpace(cfg.ServerUrl)
	if cfg.ServerUrl == "" || cfg.ServerUrl == "FIXME" {
		return Config{}, fmt.Errorf("server_url is not configured (set MINIFLUX_URL or server_url in config.toml)")
	}
	cfg.ServerUrl = strings.TrimSuffix(cfg.ServerUrl, "/")

	if cfg.CacheExpiryDays <= 0 {
		cfg.CacheExpiryDays = 30
	}

	return cfg, nil
}
