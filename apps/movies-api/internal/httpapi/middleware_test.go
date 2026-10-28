package httpapi

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/BrionesBenjamin22/laboratorio-azure/apps/movies-api/internal/movies"
)

func TestRequestIDMiddlewarePropagatesValidIdentifier(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	request.Header.Set(requestIDHeader, "client-request_123")
	response := httptest.NewRecorder()

	testHandler().ServeHTTP(response, request)

	if requestID := response.Header().Get(requestIDHeader); requestID != "client-request_123" {
		t.Errorf("expected propagated request ID, got %q", requestID)
	}

	var body errorEnvelope
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error.RequestID != "client-request_123" {
		t.Errorf("expected correlated request ID, got %q", body.Error.RequestID)
	}
}

func TestRequestIDMiddlewareReplacesInvalidIdentifier(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	request.Header.Set(requestIDHeader, "invalid identifier with spaces")
	response := httptest.NewRecorder()

	testHandler().ServeHTTP(response, request)

	requestID := response.Header().Get(requestIDHeader)
	if requestID == "" || requestID == "invalid identifier with spaces" {
		t.Errorf("expected a generated request ID, got %q", requestID)
	}
	if !validRequestID(requestID) {
		t.Errorf("generated request ID is invalid: %q", requestID)
	}
}

func TestLoggingMiddlewareRecordsSafeRequestFields(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	handler := NewHandler(logger, movies.NewService(&httpFakeProvider{
		page: movies.Page{Number: 1, TotalPages: 1, Movies: []movies.Movie{}},
	}))
	request := httptest.NewRequest(http.MethodGet, "/unknown?token=super-secret", nil)
	request.Header.Set("Authorization", "Bearer hidden-secret")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	var record map[string]any
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatalf("decode log record: %v", err)
	}

	if record["request_id"] != response.Header().Get(requestIDHeader) {
		t.Errorf("log request ID does not match response header")
	}
	if record["method"] != http.MethodGet {
		t.Errorf("expected method GET, got %v", record["method"])
	}
	if record["path"] != "/unknown" {
		t.Errorf("expected path without query, got %v", record["path"])
	}
	if record["status"] != float64(http.StatusNotFound) {
		t.Errorf("expected status 404, got %v", record["status"])
	}
	if _, found := record["duration_ms"]; !found {
		t.Error("expected duration_ms in log record")
	}

	logged := output.String()
	if strings.Contains(logged, "super-secret") || strings.Contains(logged, "hidden-secret") {
		t.Errorf("log contains sensitive request data: %s", logged)
	}
}

func TestStatusRecorderSupportsResponseController(t *testing.T) {
	underlying := httptest.NewRecorder()
	recorder := &statusRecorder{ResponseWriter: underlying}

	if err := http.NewResponseController(recorder).Flush(); err != nil {
		t.Fatalf("flush through wrapped response writer: %v", err)
	}
}
