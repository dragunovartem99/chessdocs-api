package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

// statusError is implemented by errors that already know how they should reach
// the client — see docs.Error and github.UpstreamError. Anything else is a bug
// on our side and becomes a 500 with its details kept to the log.
type statusError interface {
	error
	HTTPStatus() int
	PublicMessage() string
}

// fail answers with the status the error asks for, logging server-side and
// upstream faults on the way out.
func (s *server) fail(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusInternalServerError
	message := "Internal server error"

	var known statusError
	if errors.As(err, &known) {
		status, message = known.HTTPStatus(), known.PublicMessage()
	}
	if status >= 500 {
		s.log.ErrorContext(r.Context(), "request failed",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", status),
			slog.Any("error", err),
		)
	}
	writeError(w, status, message)
}

// writeError renders the { "error": string } body the OpenAPI document
// promises for every failure.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	w.Write(body)
}
