package tmdb

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPopularSendsExpectedRequestAndMapsResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/3/movie/popular" {
			t.Errorf("expected popular path, got %q", r.URL.Path)
		}
		if r.URL.Query().Get("language") != "es-AR" || r.URL.Query().Get("page") != "2" {
			t.Errorf("unexpected query: %s", r.URL.RawQuery)
		}
		assertAuthenticationHeaders(t, r)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"page":2,"total_pages":4,"results":[{"id":11,"title":"Película","original_title":"Movie","overview":"Resumen","release_date":"2026-01-02","poster_path":"/poster.jpg","backdrop_path":null,"vote_average":8.2,"genre_ids":[12,18]}]}`)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+"/3", time.Second, testLogger())
	page, err := client.Popular(context.Background(), 2)
	if err != nil {
		t.Fatalf("popular: %v", err)
	}

	if page.Number != 2 || page.TotalPages != 4 || len(page.Movies) != 1 {
		t.Fatalf("unexpected page: %+v", page)
	}
	movie := page.Movies[0]
	if movie.ID != 11 || movie.Title != "Película" || movie.OriginalTitle != "Movie" || movie.PosterPath == nil {
		t.Errorf("unexpected movie mapping: %+v", movie)
	}
	if len(movie.GenreIDs) != 2 || movie.GenreIDs[0] != 12 || movie.BackdropPath != nil {
		t.Errorf("unexpected optional fields: %+v", movie)
	}
}

func TestSearchSendsEncodedQueryWithoutAdultResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/3/search/movie" {
			t.Errorf("expected search path, got %q", r.URL.Path)
		}
		query := r.URL.Query()
		if query.Get("query") != "El viaje 日本" || query.Get("include_adult") != "false" || query.Get("page") != "3" {
			t.Errorf("unexpected search query: %s", r.URL.RawQuery)
		}
		fmt.Fprint(w, `{"page":3,"total_pages":3,"results":[]}`)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+"/3", time.Second, testLogger())
	page, err := client.Search(context.Background(), "El viaje 日本", 3)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if page.Number != 3 || len(page.Movies) != 0 {
		t.Errorf("unexpected search page: %+v", page)
	}
}

func TestDetailMapsGenresAndOptionalFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/3/movie/550" {
			t.Errorf("expected detail path, got %q", r.URL.Path)
		}
		if r.URL.Query().Get("language") != "es-AR" {
			t.Errorf("unexpected language: %s", r.URL.RawQuery)
		}
		fmt.Fprint(w, `{"id":550,"title":"El club de la pelea","original_title":"Fight Club","overview":"Resumen","release_date":"1999-10-15","poster_path":null,"backdrop_path":"/backdrop.jpg","vote_average":8.4,"genres":[{"id":18,"name":"Drama"}]}`)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+"/3", time.Second, testLogger())
	movie, err := client.Detail(context.Background(), 550)
	if err != nil {
		t.Fatalf("detail: %v", err)
	}
	if movie.ID != 550 || movie.PosterPath != nil || movie.BackdropPath == nil {
		t.Errorf("unexpected detail mapping: %+v", movie)
	}
	if len(movie.Genres) != 1 || movie.Genres[0].ID != 18 || movie.Genres[0].Name != "Drama" {
		t.Errorf("unexpected genres: %+v", movie.Genres)
	}
}

func TestClientClassifiesProviderStatuses(t *testing.T) {
	tests := []struct {
		status int
		want   error
	}{
		{status: http.StatusNotFound, want: ErrNotFound},
		{status: http.StatusUnauthorized, want: ErrUnauthorized},
		{status: http.StatusForbidden, want: ErrUnauthorized},
		{status: http.StatusTooManyRequests, want: ErrRateLimited},
		{status: http.StatusInternalServerError, want: ErrUnavailable},
		{status: http.StatusBadGateway, want: ErrUnavailable},
		{status: http.StatusUnprocessableEntity, want: ErrInvalidResponse},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("status_%d", test.status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.status)
				fmt.Fprint(w, `{"status_message":"must not leak"}`)
			}))
			defer server.Close()

			client := newTestClient(t, server.URL, time.Second, testLogger())
			_, err := client.Popular(context.Background(), 1)
			if !errors.Is(err, test.want) {
				t.Fatalf("expected %v, got %v", test.want, err)
			}
			if strings.Contains(err.Error(), "must not leak") {
				t.Errorf("provider body leaked in error: %v", err)
			}
		})
	}
}

func TestClientClassifiesTimeoutAndUnavailableNetwork(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		fmt.Fprint(w, `{"page":1,"total_pages":0,"results":[]}`)
	}))

	client := newTestClient(t, server.URL, 10*time.Millisecond, testLogger())
	if _, err := client.Popular(context.Background(), 1); !errors.Is(err, ErrTimeout) {
		t.Fatalf("expected timeout, got %v", err)
	}

	server.Close()
	if _, err := client.Popular(context.Background(), 1); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected unavailable error, got %v", err)
	}
}

func TestClientClassifiesInvalidPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"page":`)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, time.Second, testLogger())
	if _, err := client.Popular(context.Background(), 1); !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("expected invalid response, got %v", err)
	}
}

func TestClientPreservesCallerCancellation(t *testing.T) {
	client := newTestClient(t, "http://127.0.0.1:1", time.Second, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := client.Popular(ctx, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}

func TestProviderLogDoesNotContainTokenQueryOrBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"page":1,"total_pages":0,"results":[]}`)
	}))
	defer server.Close()

	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	client := newTestClient(t, server.URL, time.Second, logger)
	if _, err := client.Search(context.Background(), "private query", 1); err != nil {
		t.Fatalf("search: %v", err)
	}

	logged := output.String()
	for _, sensitive := range []string{"test-token", "private query", "results"} {
		if strings.Contains(logged, sensitive) {
			t.Errorf("provider log contains sensitive value %q: %s", sensitive, logged)
		}
	}
	for _, field := range []string{"provider", "operation", "status", "duration_ms"} {
		if !strings.Contains(logged, `"`+field+`"`) {
			t.Errorf("provider log is missing field %q: %s", field, logged)
		}
	}
}

func TestNewValidatesConfiguration(t *testing.T) {
	tests := []Config{
		{BaseURL: "://invalid", APIToken: "token", Language: "es-AR", Timeout: time.Second},
		{BaseURL: "https://example.com", APIToken: "", Language: "es-AR", Timeout: time.Second},
		{BaseURL: "https://example.com", APIToken: "token", Language: "", Timeout: time.Second},
		{BaseURL: "https://example.com", APIToken: "token", Language: "es-AR", Timeout: 0},
	}

	for index, cfg := range tests {
		if _, err := New(cfg, testLogger()); err == nil {
			t.Errorf("case %d: expected an error", index)
		}
	}
}

func newTestClient(t *testing.T, baseURL string, timeout time.Duration, logger *slog.Logger) *Client {
	t.Helper()
	client, err := New(Config{
		BaseURL:  baseURL,
		APIToken: "test-token",
		Language: "es-AR",
		Timeout:  timeout,
	}, logger)
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	return client
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func assertAuthenticationHeaders(t *testing.T, r *http.Request) {
	t.Helper()
	if authorization := r.Header.Get("Authorization"); authorization != "Bearer test-token" {
		t.Errorf("unexpected Authorization header %q", authorization)
	}
	if accept := r.Header.Get("Accept"); accept != "application/json" {
		t.Errorf("unexpected Accept header %q", accept)
	}
}
