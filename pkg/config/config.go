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
	ApiKey               string   `toml:"api_key"`
	ServerUrl            string   `toml:"server_url"`
	AllowInvalidCerts    bool     `toml:"allow_invalid_certs"`
	NerdFont             bool     `toml:"nerd_font"`
	ColorScheme          string   `toml:"color_scheme"`
	StripLinkFeeds       []string `toml:"strip_link_feeds"`
	CacheExpiryDays      int      `toml:"cache_expiry_days"`
	RememberReadPosition bool     `toml:"remember_read_position"`
	CacheDir             string   `toml:"-"`
}

func DefaultConfig() Config {
	return Config{
		ApiKey:               "FIXME",
		ServerUrl:            "FIXME",
		AllowInvalidCerts:    false,
		NerdFont:             false,
		ColorScheme:          "default",
		CacheExpiryDays:      30,
		RememberReadPosition: true,
	}
}

func GetConfigFilepath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("could not find user config dir: %w", err)
	}
	path := filepath.Join(configDir, "anus", "config.toml")
	return path, nil
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

func Load() (Config, error) {
	path, err := GetConfigFilepath()
	if err != nil {
		return Config{}, err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return Config{}, fmt.Errorf("config file not found at %s. run with --init to create one", path)
	}

	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return Config{}, fmt.Errorf("error parsing config file: %w", err)
	}

	return validate(cfg, path)
}

// LoadFromEnv reads configuration from environment variables.
// Required: MINIFLUX_URL, MINIFLUX_API_KEY.
// Optional: ALLOW_INVALID_CERTS, CACHE_EXPIRY_DAYS, CACHE_DIR, REMEMBER_READ_POSITION.
func LoadFromEnv() (Config, error) {
	cfg := DefaultConfig()
	cfg.ApiKey = os.Getenv("MINIFLUX_API_KEY")
	cfg.ServerUrl = os.Getenv("MINIFLUX_URL")

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
	if v := os.Getenv("REMEMBER_READ_POSITION"); v != "" {
		cfg.RememberReadPosition, _ = strconv.ParseBool(v)
	}

	return validate(cfg, "environment")
}

func validate(cfg Config, source string) (Config, error) {
	if cfg.ApiKey == "" || cfg.ApiKey == "FIXME" {
		return Config{}, fmt.Errorf("api_key is not configured in %s", source)
	}

	cfg.ServerUrl = strings.TrimSpace(cfg.ServerUrl)
	if cfg.ServerUrl == "" || cfg.ServerUrl == "FIXME" {
		return Config{}, fmt.Errorf("server_url is not configured in %s", source)
	}
	cfg.ServerUrl = strings.TrimSuffix(cfg.ServerUrl, "/")

	if cfg.CacheExpiryDays <= 0 {
		cfg.CacheExpiryDays = 30
	}

	return cfg, nil
}
