package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/BrionesBenjamin22/laboratorio-azure/apps/movies-api/internal/movies"
)

func NewHandler(logger *slog.Logger, movieService *movies.Service) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health/live", requireMethod(http.MethodGet, healthHandler("ok")))
	mux.HandleFunc("/health/ready", requireMethod(http.MethodGet, healthHandler("ready")))
	mux.HandleFunc("/api/v1/movies/popular", requireMethod(http.MethodGet, popularHandler(movieService)))
	mux.HandleFunc("/", notFoundHandler)

	return requestIDMiddleware(loggingMiddleware(logger)(mux))
}

func requireMethod(method string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			w.Header().Set("Allow", method)
			writeError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Lo sentimos. El método solicitado no está permitido. Verifique la solicitud e intente nuevamente.")
			return
		}

		next(w, r)
	}
}

func healthHandler(status string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, healthResponse{Status: status})
	}
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	writeError(w, r, http.StatusNotFound, "NOT_FOUND", "Lo sentimos. No pudimos encontrar el recurso solicitado. Verifique la dirección e intente nuevamente.")
}
