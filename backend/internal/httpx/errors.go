// Package httpx regroupe les utilitaires HTTP communs : erreurs normalisées,
// réponses JSON, identité de l'appelant et limitation de débit.
package httpx

import (
	"maps"
	"net/http"
)

// Error est le format unique des erreurs de l'API :
//
//	{"error": {"code": "invalid_pin", "message": "…", "details": {…}}}
//
// Code est stable et destiné au client ; Message est en français, affichable tel quel.
type Error struct {
	Status  int            `json:"-"`
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func NewError(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

// With renvoie une copie enrichie d'un détail ; l'erreur d'origine (souvent une
// variable partagée) n'est pas modifiée.
func (e *Error) With(key string, value any) *Error {
	c := *e
	c.Details = maps.Clone(e.Details)
	if c.Details == nil {
		c.Details = map[string]any{}
	}
	c.Details[key] = value
	return &c
}

func Validation(field, message string) *Error {
	return NewError(http.StatusBadRequest, "validation_error", message).With("field", field)
}

var (
	ErrUnauthenticated = NewError(http.StatusUnauthorized, "unauthenticated", "Connexion requise.")
	ErrForbidden       = NewError(http.StatusForbidden, "forbidden", "Accès refusé.")
	ErrNotFound        = NewError(http.StatusNotFound, "not_found", "Ressource introuvable.")
	ErrInvalidJSON     = NewError(http.StatusBadRequest, "invalid_json", "Corps de requête JSON invalide.")
	ErrTooManyRequests = NewError(http.StatusTooManyRequests, "rate_limited", "Trop de requêtes. Réessayez dans un instant.")
	errInternal        = NewError(http.StatusInternalServerError, "internal_error", "Erreur interne du serveur.")
)
