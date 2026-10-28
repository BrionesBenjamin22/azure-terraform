package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/BrionesBenjamin22/laboratorio-azure/apps/movies-api/internal/movies"
)

func TestPopularEndpointUsesDefaultPageAndReturnsEnvelope(t *testing.T) {
	providerMovies := make([]movies.Movie, 12)
	for index := range providerMovies {
		providerMovies[index] = movies.Movie{ID: int64(index + 1), Title: "Movie"}
	}
	provider := &httpFakeProvider{page: movies.Page{Number: 1, TotalPages: 2, Movies: providerMovies}}
	handler := handlerWithProvider(provider)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/movies/popular", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if provider.calls != 1 || provider.requestedPage != 1 {
		t.Errorf("unexpected provider call: calls=%d page=%d", provider.calls, provider.requestedPage)
	}

	var body movieListEnvelope
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Data) != 9 || body.Meta.Page != 1 || body.Meta.Returned != 9 || !body.Meta.HasNext {
		t.Errorf("unexpected response: %+v", body)
	}
}

func TestPopularEndpointUsesRequestedProviderPage(t *testing.T) {
	provider := &httpFakeProvider{page: movies.Page{Number: 2, TotalPages: 2, Movies: []movies.Movie{}}}
	handler := handlerWithProvider(provider)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/movies/popular?page=2", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK || provider.requestedPage != 2 {
		t.Fatalf("expected page 2 and status 200, got page=%d status=%d", provider.requestedPage, response.Code)
	}
}

func TestPopularEndpointRejectsInvalidPagesWithoutProviderCall(t *testing.T) {
	paths := []string{
		"/api/v1/movies/popular?page=",
		"/api/v1/movies/popular?page=0",
		"/api/v1/movies/popular?page=-1",
		"/api/v1/movies/popular?page=text",
		"/api/v1/movies/popular?page=1&page=2",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			provider := &httpFakeProvider{}
			handler := handlerWithProvider(provider)
			request := httptest.NewRequest(http.MethodGet, path, nil)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d", response.Code)
			}
			if provider.calls != 0 {
				t.Fatalf("expected no provider calls, got %d", provider.calls)
			}
			var body errorEnvelope
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if body.Error.Code != "INVALID_PAGE" || body.Error.RequestID == "" {
				t.Errorf("unexpected error response: %+v", body)
			}
		})
	}
}

func TestPopularEndpointMapsProviderErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "unauthorized", err: movies.ErrProviderUnauthorized, wantStatus: http.StatusBadGateway, wantCode: "PROVIDER_AUTHENTICATION_ERROR"},
		{name: "rate limited", err: movies.ErrProviderRateLimited, wantStatus: http.StatusServiceUnavailable, wantCode: "PROVIDER_RATE_LIMITED"},
		{name: "timeout", err: movies.ErrProviderTimeout, wantStatus: http.StatusGatewayTimeout, wantCode: "PROVIDER_TIMEOUT"},
		{name: "unavailable", err: movies.ErrProviderUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: "PROVIDER_UNAVAILABLE"},
		{name: "invalid response", err: movies.ErrProviderInvalidResponse, wantStatus: http.StatusBadGateway, wantCode: "PROVIDER_INVALID_RESPONSE"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider := &httpFakeProvider{err: test.err}
			request := httptest.NewRequest(http.MethodGet, "/api/v1/movies/popular", nil)
			response := httptest.NewRecorder()

			handlerWithProvider(provider).ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("expected status %d, got %d", test.wantStatus, response.Code)
			}
			var body errorEnvelope
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body.Error.Code != test.wantCode || body.Error.RequestID == "" {
				t.Errorf("unexpected mapped error: %+v", body.Error)
			}
		})
	}
}

func handlerWithProvider(provider *httpFakeProvider) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewHandler(logger, movies.NewService(provider))
}
