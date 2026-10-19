package server

import (
	"net/http"
	"os"
	"strings"
	"testing"

	"samacode/backend/internal/users"
)

// Q13 : un compte créé avant les CGU (ou après leur mise à jour) doit les accepter.
func TestTermsAcceptance(t *testing.T) {
	e := newAdminEnv(t)
	candidate := e.user("candidate")

	status, body := candidate.do("GET", "/api/v1/me", nil)
	mustStatus(t, status, body, http.StatusOK, "profil")
	if body["terms_accepted"] != false || body["terms_version"] != users.TermsVersion {
		t.Fatalf("profil avant acceptation = %v", body)
	}

	status, body = candidate.do("POST", "/api/v1/me/terms-acceptance", map[string]string{"version": "2000-01-01"})
	mustStatus(t, status, body, http.StatusConflict, "anciennes CGU")
	status, body = candidate.do("POST", "/api/v1/me/terms-acceptance", map[string]string{})
	mustStatus(t, status, body, http.StatusBadRequest, "version absente")

	for range 2 { // idempotent
		status, body = candidate.do("POST", "/api/v1/me/terms-acceptance", map[string]string{"version": users.TermsVersion})
		mustStatus(t, status, body, http.StatusOK, "acceptation")
		if body["terms_accepted"] != true {
			t.Fatalf("profil après acceptation = %v", body)
		}
	}
}

// La version des CGU de l'application doit être celle de l'API, sinon l'inscription
// est refusée (terms_outdated) : les deux changent ensemble.
func TestTermsVersionMatchesMobile(t *testing.T) {
	src, err := os.ReadFile("../../../mobile/lib/core/config.dart")
	if err != nil {
		t.Skip("application mobile absente :", err)
	}
	if want := "termsVersion = '" + users.TermsVersion + "'"; !strings.Contains(string(src), want) {
		t.Fatalf("mobile/lib/core/config.dart doit contenir %q", want)
	}
}
