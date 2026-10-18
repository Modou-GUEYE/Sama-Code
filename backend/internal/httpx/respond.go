package httpx

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

// HandlerFunc est un handler qui renvoie une erreur au lieu de l'écrire lui-même.
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

func Handle(fn HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			WriteError(w, r, err)
		}
	}
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

// WriteError écrit une *Error telle quelle ; toute autre erreur devient une 500
// générique, journalisée sans être exposée au client.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		slog.ErrorContext(r.Context(), "erreur non gérée", "err", err, "method", r.Method, "path", r.URL.Path)
		apiErr = errInternal
	}
	WriteJSON(w, apiErr.Status, map[string]*Error{"error": apiErr})
}

const maxBodyBytes = 1 << 20

// DecodeJSON lit un corps JSON (1 Mo max) en refusant les champs inconnus.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return ErrInvalidJSON
	}
	return nil
}
