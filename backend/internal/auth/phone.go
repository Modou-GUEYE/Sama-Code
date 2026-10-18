package auth

import (
	"net/http"
	"strings"

	"samacode/backend/internal/httpx"
)

var ErrInvalidPhone = httpx.NewError(http.StatusBadRequest, "invalid_phone",
	"Numéro de téléphone invalide. Exemple : 77 123 45 67.")

// Préfixes des numéros mobiles sénégalais (Expresso, Promobile, Free, Orange).
var mobilePrefixes = []string{"70", "75", "76", "77", "78"}

// NormalizePhone convertit un numéro mobile sénégalais au format E.164 (+221XXXXXXXXX).
// Accepte « 77 123 45 67 », « 771234567 », « +221 77… », « 00221 77… ».
func NormalizePhone(raw string) (string, error) {
	var b strings.Builder
	for i, r := range strings.TrimSpace(raw) {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '+' && i == 0:
		case r == ' ' || r == '-' || r == '.' || r == '(' || r == ')':
		default:
			return "", ErrInvalidPhone
		}
	}

	d := b.String()
	switch {
	case strings.HasPrefix(d, "00221"):
		d = d[5:]
	case strings.HasPrefix(d, "221") && len(d) == 12:
		d = d[3:]
	}
	if len(d) != 9 || !hasMobilePrefix(d) {
		return "", ErrInvalidPhone
	}
	return "+221" + d, nil
}

func hasMobilePrefix(d string) bool {
	for _, p := range mobilePrefixes {
		if strings.HasPrefix(d, p) {
			return true
		}
	}
	return false
}
