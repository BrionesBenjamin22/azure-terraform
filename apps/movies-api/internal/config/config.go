package config

import (
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultHTTPPort                = 8080
	defaultHTTPReadTimeoutSeconds  = 5
	defaultHTTPWriteTimeoutSeconds = 10
	defaultHTTPIdleTimeoutSeconds  = 60
	defaultShutdownTimeoutSeconds  = 10
	defaultLogLevel                = "info"
	defaultTMDBBaseURL             = "https://api.themoviedb.org/3"
	defaultTMDBLanguage            = "es-AR"
	defaultTMDBTimeoutSeconds      = 5
)

type Config struct {
	HTTPPort         int
	HTTPReadTimeout  time.Duration
	HTTPWriteTimeout time.Duration
	HTTPIdleTimeout  time.Duration
	ShutdownTimeout  time.Duration
	LogLevel         slog.Level
	TMDBBaseURL      string
	TMDBAPIToken     string
	TMDBLanguage     string
	TMDBTimeout      time.Duration
}

func Load() (Config, error) {
	return load(os.LookupEnv)
}

func (cfg Config) HTTPAddress() string {
	return net.JoinHostPort("", strconv.Itoa(cfg.HTTPPort))
}

type lookupEnv func(string) (string, bool)

func load(lookup lookupEnv) (Config, error) {
	httpPort, err := positiveInt(lookup, "HTTP_PORT", defaultHTTPPort)
	if err != nil {
		return Config{}, err
	}
	if httpPort > 65535 {
		return Config{}, fmt.Errorf("HTTP_PORT must be between 1 and 65535")
	}

	httpReadTimeout, err := positiveSeconds(lookup, "HTTP_READ_TIMEOUT_SECONDS", defaultHTTPReadTimeoutSeconds)
	if err != nil {
		return Config{}, err
	}
	httpWriteTimeout, err := positiveSeconds(lookup, "HTTP_WRITE_TIMEOUT_SECONDS", defaultHTTPWriteTimeoutSeconds)
	if err != nil {
		return Config{}, err
	}
	httpIdleTimeout, err := positiveSeconds(lookup, "HTTP_IDLE_TIMEOUT_SECONDS", defaultHTTPIdleTimeoutSeconds)
	if err != nil {
		return Config{}, err
	}
	shutdownTimeout, err := positiveSeconds(lookup, "SHUTDOWN_TIMEOUT_SECONDS", defaultShutdownTimeoutSeconds)
	if err != nil {
		return Config{}, err
	}

	logLevelText := valueOrDefault(lookup, "LOG_LEVEL", defaultLogLevel)
	var logLevel slog.Level
	if err := logLevel.UnmarshalText([]byte(logLevelText)); err != nil {
		return Config{}, fmt.Errorf("LOG_LEVEL must be debug, info, warn, or error")
	}

	tmdbBaseURL, err := baseURL(lookup)
	if err != nil {
		return Config{}, err
	}
	tmdbAPIToken := strings.TrimSpace(valueOrDefault(lookup, "TMDB_API_TOKEN", ""))
	if tmdbAPIToken == "" {
		return Config{}, fmt.Errorf("TMDB_API_TOKEN is required")
	}
	tmdbLanguage := valueOrDefault(lookup, "TMDB_DEFAULT_LANGUAGE", defaultTMDBLanguage)
	if !validLanguage(tmdbLanguage) {
		return Config{}, fmt.Errorf("TMDB_DEFAULT_LANGUAGE must use a language or language-country code")
	}
	tmdbTimeout, err := positiveSeconds(lookup, "TMDB_TIMEOUT_SECONDS", defaultTMDBTimeoutSeconds)
	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTPPort:         httpPort,
		HTTPReadTimeout:  httpReadTimeout,
		HTTPWriteTimeout: httpWriteTimeout,
		HTTPIdleTimeout:  httpIdleTimeout,
		ShutdownTimeout:  shutdownTimeout,
		LogLevel:         logLevel,
		TMDBBaseURL:      tmdbBaseURL,
		TMDBAPIToken:     tmdbAPIToken,
		TMDBLanguage:     tmdbLanguage,
		TMDBTimeout:      tmdbTimeout,
	}, nil
}

func baseURL(lookup lookupEnv) (string, error) {
	value := strings.TrimRight(valueOrDefault(lookup, "TMDB_BASE_URL", defaultTMDBBaseURL), "/")
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("TMDB_BASE_URL must be an absolute HTTP or HTTPS URL without credentials, query, or fragment")
	}
	return value, nil
}

func validLanguage(value string) bool {
	if len(value) < 2 || len(value) > 8 {
		return false
	}
	for index, character := range value {
		if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character == '-' && index > 0 && index < len(value)-1 {
			continue
		}
		return false
	}
	return true
}

func positiveSeconds(lookup lookupEnv, name string, fallback int) (time.Duration, error) {
	value, err := positiveInt(lookup, name, fallback)
	if err != nil {
		return 0, err
	}
	return time.Duration(value) * time.Second, nil
}

func positiveInt(lookup lookupEnv, name string, fallback int) (int, error) {
	text := valueOrDefault(lookup, name, strconv.Itoa(fallback))
	value, err := strconv.Atoi(text)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return value, nil
}

func valueOrDefault(lookup lookupEnv, name, fallback string) string {
	if value, found := lookup(name); found && value != "" {
		return value
	}
	return fallback
}
