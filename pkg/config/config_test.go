package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/slatkin/anus/pkg/config"
)

// isolate redirects XDG_CONFIG_HOME to a temp dir with no config file.
func isolate(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("MINIFLUX_API_KEY", "")
	t.Setenv("MINIFLUX_URL", "")
	t.Setenv("ALLOW_INVALID_CERTS", "")
	t.Setenv("CACHE_EXPIRY_DAYS", "")
	t.Setenv("CACHE_DIR", "")
}

func writeConfig(t *testing.T, content string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfgDir := filepath.Join(dir, "anus")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	if cfg.CacheExpiryDays != 30 {
		t.Errorf("CacheExpiryDays: got %d, want 30", cfg.CacheExpiryDays)
	}
	if cfg.AllowInvalidCerts {
		t.Error("AllowInvalidCerts should default to false")
	}
	if cfg.ApiKey != "FIXME" {
		t.Errorf("ApiKey: got %q, want \"FIXME\"", cfg.ApiKey)
	}
}

func TestLoad_MissingBoth(t *testing.T) {
	isolate(t)
	_, err := config.Load()
	if err == nil {
		t.Error("expected error when no config file and no env vars")
	}
}

func TestLoad_FixmeApiKey(t *testing.T) {
	isolate(t)
	writeConfig(t, "api_key = \"FIXME\"\nserver_url = \"http://example.com\"\n")
	_, err := config.Load()
	if err == nil {
		t.Error("expected error for api_key = FIXME")
	}
}

func TestLoad_EmptyApiKey(t *testing.T) {
	isolate(t)
	writeConfig(t, "api_key = \"\"\nserver_url = \"http://example.com\"\n")
	_, err := config.Load()
	if err == nil {
		t.Error("expected error for empty api_key")
	}
}

func TestLoad_FixmeServerUrl(t *testing.T) {
	isolate(t)
	writeConfig(t, "api_key = \"mykey\"\nserver_url = \"FIXME\"\n")
	_, err := config.Load()
	if err == nil {
		t.Error("expected error for server_url = FIXME")
	}
}

func TestLoad_EmptyServerUrl(t *testing.T) {
	isolate(t)
	writeConfig(t, "api_key = \"mykey\"\nserver_url = \"\"\n")
	_, err := config.Load()
	if err == nil {
		t.Error("expected error for empty server_url")
	}
}

func TestLoad_TrailingSlashStripped(t *testing.T) {
	isolate(t)
	writeConfig(t, "api_key = \"mykey\"\nserver_url = \"http://example.com/\"\n")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ServerUrl != "http://example.com" {
		t.Errorf("ServerUrl: got %q, want trailing slash stripped", cfg.ServerUrl)
	}
}

func TestLoad_CacheExpiryDaysZeroDefaults(t *testing.T) {
	isolate(t)
	writeConfig(t, "api_key = \"mykey\"\nserver_url = \"http://example.com\"\ncache_expiry_days = 0\n")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CacheExpiryDays != 30 {
		t.Errorf("CacheExpiryDays: got %d, want 30 when set to 0", cfg.CacheExpiryDays)
	}
}

func TestLoad_NegativeCacheExpiryDaysDefaults(t *testing.T) {
	isolate(t)
	writeConfig(t, "api_key = \"mykey\"\nserver_url = \"http://example.com\"\ncache_expiry_days = -5\n")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CacheExpiryDays != 30 {
		t.Errorf("CacheExpiryDays: got %d, want 30 when negative", cfg.CacheExpiryDays)
	}
}

func TestLoad_ValidFile(t *testing.T) {
	isolate(t)
	writeConfig(t, "api_key = \"mykey\"\nserver_url = \"http://example.com\"\ncache_expiry_days = 14\n")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ApiKey != "mykey" {
		t.Errorf("ApiKey: got %q, want \"mykey\"", cfg.ApiKey)
	}
	if cfg.ServerUrl != "http://example.com" {
		t.Errorf("ServerUrl: got %q, want \"http://example.com\"", cfg.ServerUrl)
	}
	if cfg.CacheExpiryDays != 14 {
		t.Errorf("CacheExpiryDays: got %d, want 14", cfg.CacheExpiryDays)
	}
}

func TestLoad_EnvVarsOnly(t *testing.T) {
	isolate(t)
	t.Setenv("MINIFLUX_API_KEY", "mykey")
	t.Setenv("MINIFLUX_URL", "http://example.com")
	t.Setenv("CACHE_EXPIRY_DAYS", "14")
	t.Setenv("ALLOW_INVALID_CERTS", "true")
	t.Setenv("CACHE_DIR", "/tmp/anus-cache")

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ApiKey != "mykey" {
		t.Errorf("ApiKey: got %q", cfg.ApiKey)
	}
	if cfg.ServerUrl != "http://example.com" {
		t.Errorf("ServerUrl: got %q", cfg.ServerUrl)
	}
	if cfg.CacheExpiryDays != 14 {
		t.Errorf("CacheExpiryDays: got %d, want 14", cfg.CacheExpiryDays)
	}
	if !cfg.AllowInvalidCerts {
		t.Error("AllowInvalidCerts: want true")
	}
	if cfg.CacheDir != "/tmp/anus-cache" {
		t.Errorf("CacheDir: got %q", cfg.CacheDir)
	}
}

func TestLoad_FileOverridesEnv(t *testing.T) {
	isolate(t)
	t.Setenv("MINIFLUX_API_KEY", "env-key")
	t.Setenv("MINIFLUX_URL", "http://env.example.com")
	t.Setenv("CACHE_EXPIRY_DAYS", "7")
	writeConfig(t, "api_key = \"file-key\"\nserver_url = \"http://file.example.com\"\ncache_expiry_days = 60\n")

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ApiKey != "file-key" {
		t.Errorf("ApiKey: got %q, want file to win over env", cfg.ApiKey)
	}
	if cfg.ServerUrl != "http://file.example.com" {
		t.Errorf("ServerUrl: got %q, want file to win over env", cfg.ServerUrl)
	}
	if cfg.CacheExpiryDays != 60 {
		t.Errorf("CacheExpiryDays: got %d, want 60 from file", cfg.CacheExpiryDays)
	}
}

func TestLoad_EnvTrailingSlashStripped(t *testing.T) {
	isolate(t)
	t.Setenv("MINIFLUX_API_KEY", "mykey")
	t.Setenv("MINIFLUX_URL", "http://example.com/")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ServerUrl != "http://example.com" {
		t.Errorf("ServerUrl: got %q, want trailing slash stripped", cfg.ServerUrl)
	}
}

func TestLoad_DefaultCacheExpiryFromEnv(t *testing.T) {
	isolate(t)
	t.Setenv("MINIFLUX_API_KEY", "mykey")
	t.Setenv("MINIFLUX_URL", "http://example.com")
	t.Setenv("CACHE_EXPIRY_DAYS", "0")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CacheExpiryDays != 30 {
		t.Errorf("CacheExpiryDays: got %d, want 30 as default when 0", cfg.CacheExpiryDays)
	}
}
