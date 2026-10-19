// Package users gère le profil et les préférences du candidat (GET/PATCH /me).
package users

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"samacode/backend/internal/httpx"
)

const maxDisplayNameRunes = 80

// TermsVersion : version en vigueur des conditions générales d'utilisation (date de
// mise à jour). À changer en même temps que le texte affiché par l'application
// (mobile/lib/features/settings/views/terms_view.dart) : les candidats devront alors
// les accepter de nouveau.
const TermsVersion = "2026-10-19"

var (
	errTermsRequired = httpx.Validation("terms_version", "Vous devez accepter les conditions d'utilisation.")
	errTermsOutdated = httpx.NewError(http.StatusConflict, "terms_outdated",
		"Les conditions d'utilisation ont changé : mettez à jour l'application.")
)

// CheckTerms vérifie que la version acceptée est celle en vigueur.
func CheckTerms(version string) error {
	switch strings.TrimSpace(version) {
	case "":
		return errTermsRequired
	case TermsVersion:
		return nil
	default:
		return errTermsOutdated
	}
}

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
