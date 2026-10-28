package config

import (
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestLoadUsesDefaults(t *testing.T) {
	cfg, err := load(environmentWithToken(nil))
	if err != nil {
		t.Fatalf("load defaults: %v", err)
	}

	if cfg.HTTPPort != 8080 {
		t.Errorf("expected HTTP port 8080, got %d", cfg.HTTPPort)
	}
	if cfg.HTTPReadTimeout != 5*time.Second {
		t.Errorf("expected read timeout 5s, got %s", cfg.HTTPReadTimeout)
	}
	if cfg.HTTPWriteTimeout != 10*time.Second {
		t.Errorf("expected write timeout 10s, got %s", cfg.HTTPWriteTimeout)
	}
	if cfg.HTTPIdleTimeout != 60*time.Second {
		t.Errorf("expected idle timeout 60s, got %s", cfg.HTTPIdleTimeout)
	}
	if cfg.ShutdownTimeout != 10*time.Second {
		t.Errorf("expected shutdown timeout 10s, got %s", cfg.ShutdownTimeout)
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Errorf("expected info log level, got %s", cfg.LogLevel)
	}
	if cfg.TMDBBaseURL != "https://api.themoviedb.org/3" {
		t.Errorf("unexpected TMDB base URL %q", cfg.TMDBBaseURL)
	}
	if cfg.TMDBAPIToken != "test-token" {
		t.Error("expected configured TMDB token")
	}
	if cfg.TMDBLanguage != "es-AR" || cfg.TMDBTimeout != 5*time.Second {
		t.Errorf("unexpected TMDB defaults: language=%q timeout=%s", cfg.TMDBLanguage, cfg.TMDBTimeout)
	}
}

func TestLoadUsesEnvironmentValues(t *testing.T) {
	values := map[string]string{
		"HTTP_PORT":                  "9090",
		"HTTP_READ_TIMEOUT_SECONDS":  "2",
		"HTTP_WRITE_TIMEOUT_SECONDS": "3",
		"HTTP_IDLE_TIMEOUT_SECONDS":  "4",
		"SHUTDOWN_TIMEOUT_SECONDS":   "5",
		"LOG_LEVEL":                  "debug",
		"TMDB_BASE_URL":              "http://localhost:9091/api/",
		"TMDB_API_TOKEN":             "custom-token",
		"TMDB_DEFAULT_LANGUAGE":      "pt-BR",
		"TMDB_TIMEOUT_SECONDS":       "6",
	}

	cfg, err := load(mapEnvironment(values))
	if err != nil {
		t.Fatalf("load environment: %v", err)
	}

	if cfg.HTTPAddress() != ":9090" {
		t.Errorf("expected address :9090, got %q", cfg.HTTPAddress())
	}
	if cfg.HTTPReadTimeout != 2*time.Second || cfg.HTTPWriteTimeout != 3*time.Second || cfg.HTTPIdleTimeout != 4*time.Second {
		t.Errorf("unexpected HTTP timeouts: read=%s write=%s idle=%s", cfg.HTTPReadTimeout, cfg.HTTPWriteTimeout, cfg.HTTPIdleTimeout)
	}
	if cfg.ShutdownTimeout != 5*time.Second {
		t.Errorf("expected shutdown timeout 5s, got %s", cfg.ShutdownTimeout)
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Errorf("expected debug log level, got %s", cfg.LogLevel)
	}
	if cfg.TMDBBaseURL != "http://localhost:9091/api" || cfg.TMDBLanguage != "pt-BR" || cfg.TMDBTimeout != 6*time.Second {
		t.Errorf("unexpected TMDB configuration: base=%q language=%q timeout=%s", cfg.TMDBBaseURL, cfg.TMDBLanguage, cfg.TMDBTimeout)
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]string
	}{
		{name: "non numeric port", values: map[string]string{"HTTP_PORT": "invalid"}},
		{name: "port above range", values: map[string]string{"HTTP_PORT": "65536"}},
		{name: "zero timeout", values: map[string]string{"HTTP_READ_TIMEOUT_SECONDS": "0"}},
		{name: "negative shutdown timeout", values: map[string]string{"SHUTDOWN_TIMEOUT_SECONDS": "-1"}},
		{name: "invalid log level", values: map[string]string{"LOG_LEVEL": "verbose"}},
		{name: "invalid base URL", values: map[string]string{"TMDB_BASE_URL": "relative/path"}},
		{name: "base URL with credentials", values: map[string]string{"TMDB_BASE_URL": "https://user:password@example.com/3"}},
		{name: "invalid language", values: map[string]string{"TMDB_DEFAULT_LANGUAGE": "es AR"}},
		{name: "zero TMDB timeout", values: map[string]string{"TMDB_TIMEOUT_SECONDS": "0"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := load(environmentWithToken(test.values)); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestLoadRequiresTMDBToken(t *testing.T) {
	_, err := load(mapEnvironment(map[string]string{"TMDB_API_TOKEN": "   "}))
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "TMDB_API_TOKEN") {
		t.Fatalf("expected token configuration error, got %v", err)
	}
}

func TestLoadErrorsDoNotLeakTMDBToken(t *testing.T) {
	const secret = "secret-that-must-not-appear"
	_, err := load(mapEnvironment(map[string]string{
		"TMDB_API_TOKEN":        secret,
		"TMDB_DEFAULT_LANGUAGE": "invalid language",
	}))
	if err == nil {
		t.Fatal("expected an error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("configuration error leaked token: %v", err)
	}
}

func mapEnvironment(values map[string]string) lookupEnv {
	return func(name string) (string, bool) {
		value, found := values[name]
		return value, found
	}
}

func environmentWithToken(values map[string]string) lookupEnv {
	return func(name string) (string, bool) {
		if value, found := values[name]; found {
			return value, true
		}
		if name == "TMDB_API_TOKEN" {
			return "test-token", true
		}
		return "", false
	}
}
