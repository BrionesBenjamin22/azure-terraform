package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/BrionesBenjamin22/laboratorio-azure/apps/movies-api/internal/movies"
)

func TestHealthEndpoints(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantStatus string
	}{
		{name: "live", path: "/health/live", wantStatus: "ok"},
		{name: "ready", path: "/health/ready", wantStatus: "ready"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			response := httptest.NewRecorder()

			testHandler().ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
			}
			if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
				t.Errorf("expected Content-Type application/json, got %q", contentType)
			}

			var body healthResponse
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body.Status != test.wantStatus {
				t.Errorf("expected status body %q, got %q", test.wantStatus, body.Status)
			}
		})
	}
}

func TestHealthEndpointsRejectUnsupportedMethods(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/health/ready", nil)
	response := httptest.NewRecorder()

	testHandler().ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, response.Code)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", contentType)
	}
	if allow := response.Header().Get("Allow"); allow != http.MethodGet {
		t.Errorf("expected Allow header %q, got %q", http.MethodGet, allow)
	}

	var body errorEnvelope
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body.Error.Code != "METHOD_NOT_ALLOWED" {
		t.Errorf("expected METHOD_NOT_ALLOWED code, got %q", body.Error.Code)
	}
	if body.Error.RequestID == "" {
		t.Error("expected a request ID in the error response")
	}
}

func TestUnknownRouteUsesErrorEnvelope(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	response := httptest.NewRecorder()

	testHandler().ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, response.Code)
	}

	var body errorEnvelope
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body.Error.Code != "NOT_FOUND" {
		t.Errorf("expected NOT_FOUND code, got %q", body.Error.Code)
	}
	if body.Error.Message == "" || body.Error.RequestID == "" {
		t.Error("expected a safe message and request ID")
	}
}

func testHandler() http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewHandler(logger, movies.NewService(&httpFakeProvider{
		page: movies.Page{Number: 1, TotalPages: 1, Movies: []movies.Movie{}},
	}))
}

type httpFakeProvider struct {
	page          movies.Page
	err           error
	calls         int
	requestedPage int
}

func (provider *httpFakeProvider) Popular(_ context.Context, page int) (movies.Page, error) {
	provider.calls++
	provider.requestedPage = page
	return provider.page, provider.err
}
