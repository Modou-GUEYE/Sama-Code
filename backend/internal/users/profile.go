// Package users gère le profil et les préférences du candidat (GET/PATCH /me).
package users

import (
	"strings"
	"unicode/utf8"

	"samacode/backend/internal/httpx"
)

const maxDisplayNameRunes = 80

// Langues du MVP (D10).
var supportedLanguages = map[string]bool{"fr": true, "wo": true}

var learningModes = map[string]bool{"text": true, "audio": true}

func NormalizeDisplayName(s string) (string, error) {
	s = strings.Join(strings.Fields(s), " ")
	if s == "" || utf8.RuneCountInString(s) > maxDisplayNameRunes {
		return "", httpx.Validation("display_name", "Le nom doit contenir entre 1 et 80 caractères.")
	}
	return s, nil
}

func ValidateLanguage(code string) error {
	if !supportedLanguages[code] {
		return httpx.Validation("preferred_language", "Langue non prise en charge (fr ou wo).")
	}
	return nil
}

func ValidateLearningMode(mode string) error {
	if !learningModes[mode] {
		return httpx.Validation("learning_mode", "Mode d'apprentissage invalide (text ou audio).")
	}
	return nil
}
