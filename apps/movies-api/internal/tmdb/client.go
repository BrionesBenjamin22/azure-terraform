package tmdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/BrionesBenjamin22/laboratorio-azure/apps/movies-api/internal/movies"
)

const maxResponseBodyBytes = 2 << 20

var (
	ErrNotFound        = movies.ErrNotFound
	ErrUnauthorized    = movies.ErrProviderUnauthorized
	ErrRateLimited     = movies.ErrProviderRateLimited
	ErrTimeout         = movies.ErrProviderTimeout
	ErrUnavailable     = movies.ErrProviderUnavailable
	ErrInvalidResponse = movies.ErrProviderInvalidResponse
)

type Config struct {
	BaseURL  string
	APIToken string
	Language string
	Timeout  time.Duration
}

type Client struct {
	baseURL    *url.URL
	apiToken   string
	language   string
	httpClient *http.Client
	logger     *slog.Logger
}

func New(cfg Config, logger *slog.Logger) (*Client, error) {
	baseURL, err := url.Parse(cfg.BaseURL)
	if err != nil || baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, fmt.Errorf("create TMDB client: invalid base URL")
	}
	if strings.TrimSpace(cfg.APIToken) == "" {
		return nil, fmt.Errorf("create TMDB client: API token is required")
	}
	if cfg.Language == "" {
		return nil, fmt.Errorf("create TMDB client: language is required")
	}
	if cfg.Timeout <= 0 {
		return nil, fmt.Errorf("create TMDB client: timeout must be positive")
	}
	if logger == nil {
		return nil, fmt.Errorf("create TMDB client: logger is required")
	}

	return &Client{
		baseURL:  baseURL,
		apiToken: cfg.APIToken,
		language: cfg.Language,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		logger: logger,
	}, nil
}

func (client *Client) Popular(ctx context.Context, page int) (movies.Page, error) {
	query := url.Values{
		"language": {client.language},
		"page":     {strconv.Itoa(page)},
	}

	var response listResponse
	if err := client.get(ctx, "popular", "movie/popular", query, &response); err != nil {
		return movies.Page{}, err
	}
	return mapPage(response)
}

func (client *Client) Search(ctx context.Context, queryText string, page int) (movies.Page, error) {
	query := url.Values{
		"query":         {queryText},
		"include_adult": {"false"},
		"language":      {client.language},
		"page":          {strconv.Itoa(page)},
	}

	var response listResponse
	if err := client.get(ctx, "search", "search/movie", query, &response); err != nil {
		return movies.Page{}, err
	}
	return mapPage(response)
}

func (client *Client) Detail(ctx context.Context, movieID int64) (movies.Movie, error) {
	query := url.Values{"language": {client.language}}

	var response movieDTO
	if err := client.get(ctx, "detail", "movie/"+strconv.FormatInt(movieID, 10), query, &response); err != nil {
		return movies.Movie{}, err
	}
	if response.ID <= 0 {
		return movies.Movie{}, ErrInvalidResponse
	}
	return mapMovie(response), nil
}

func (client *Client) get(ctx context.Context, operation, path string, query url.Values, target any) error {
	startedAt := time.Now()
	status := 0
	defer func() {
		client.logger.Info("provider request",
			"provider", "tmdb",
			"operation", operation,
			"status", status,
			"duration_ms", time.Since(startedAt).Milliseconds(),
		)
	}()

	endpoint := *client.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/" + strings.TrimLeft(path, "/")
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return ErrInvalidResponse
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+client.apiToken)

	response, err := client.httpClient.Do(request)
	if err != nil {
		return classifyTransportError(ctx, err)
	}
	defer response.Body.Close()
	status = response.StatusCode

	if response.StatusCode != http.StatusOK {
		return classifyStatus(response.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBodyBytes+1))
	if err != nil || len(body) > maxResponseBodyBytes {
		return ErrInvalidResponse
	}
	if err := json.Unmarshal(body, target); err != nil {
		return ErrInvalidResponse
	}
	return nil
}

func classifyTransportError(ctx context.Context, err error) error {
	if errors.Is(ctx.Err(), context.Canceled) {
		return context.Canceled
	}
	var networkError net.Error
	if errors.Is(err, context.DeadlineExceeded) || errors.As(err, &networkError) && networkError.Timeout() {
		return ErrTimeout
	}
	return ErrUnavailable
}

func classifyStatus(status int) error {
	switch {
	case status == http.StatusNotFound:
		return ErrNotFound
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return ErrUnauthorized
	case status == http.StatusTooManyRequests:
		return ErrRateLimited
	case status >= http.StatusInternalServerError:
		return ErrUnavailable
	default:
		return ErrInvalidResponse
	}
}
