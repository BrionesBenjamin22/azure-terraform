package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/BrionesBenjamin22/laboratorio-azure/apps/movies-api/internal/movies"
)

type healthResponse struct {
	Status string `json:"status"`
}

type errorEnvelope struct {
	Error publicError `json:"error"`
}

type publicError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeJSON(w, status, errorEnvelope{Error: publicError{
		Code:      code,
		Message:   message,
		RequestID: requestIDFromContext(r.Context()),
	}})
}

func writeMappedError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, movies.ErrInvalidPage):
		writeError(w, r, http.StatusBadRequest, "INVALID_PAGE", "Lo sentimos. La página debe ser un número entero mayor o igual que 1. Corrija el valor e intente nuevamente.")
	case errors.Is(err, movies.ErrNotFound):
		writeError(w, r, http.StatusNotFound, "MOVIE_NOT_FOUND", "Lo sentimos. No pudimos encontrar la película solicitada. Verifique el identificador e intente nuevamente.")
	case errors.Is(err, movies.ErrProviderUnauthorized):
		writeError(w, r, http.StatusBadGateway, "PROVIDER_AUTHENTICATION_ERROR", "Lo sentimos. No pudimos recuperar la información. Intente nuevamente más tarde.")
	case errors.Is(err, movies.ErrProviderRateLimited):
		writeError(w, r, http.StatusServiceUnavailable, "PROVIDER_RATE_LIMITED", "Lo sentimos. El servicio está temporalmente ocupado. Intente nuevamente más tarde.")
	case errors.Is(err, movies.ErrProviderTimeout):
		writeError(w, r, http.StatusGatewayTimeout, "PROVIDER_TIMEOUT", "Lo sentimos. La consulta demoró demasiado. Intente nuevamente.")
	case errors.Is(err, movies.ErrProviderUnavailable):
		writeError(w, r, http.StatusServiceUnavailable, "PROVIDER_UNAVAILABLE", "Lo sentimos. No pudimos recuperar la información. Intente nuevamente.")
	case errors.Is(err, movies.ErrProviderInvalidResponse):
		writeError(w, r, http.StatusBadGateway, "PROVIDER_INVALID_RESPONSE", "Lo sentimos. Recibimos una respuesta inesperada. Intente nuevamente más tarde.")
	default:
		writeError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Lo sentimos. No pudimos procesar la solicitud. Intente nuevamente.")
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(append(encoded, '\n'))
}
