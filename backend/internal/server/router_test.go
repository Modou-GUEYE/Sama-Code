package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"samacode/backend/internal/auth"
	"samacode/backend/internal/testutil"
	"samacode/backend/internal/users"
)

type client struct {
	t     *testing.T
	h     http.Handler
	token string
}

func (c *client) do(method, path string, body any) (int, map[string]any) {
	c.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			c.t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	rec := httptest.NewRecorder()
	c.h.ServeHTTP(rec, req)
	var out map[string]any
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			c.t.Fatalf("%s %s : réponse non JSON %q", method, path, rec.Body.String())
		}
	}
	return rec.Code, out
}

func errorCode(body map[string]any) string {
	e, _ := body["error"].(map[string]any)
	code, _ := e["code"].(string)
	return code
}

func TestCandidateJourney(t *testing.T) {
	db := testutil.DB(t)
	sms := &testutil.SMS{}
	secret := []byte("test-secret-test-secret-test-secret!")
	tokens := auth.NewTokens(secret, 15*time.Minute)
	c := &client{t: t, h: NewRouter(Deps{
		DB: db, Tokens: tokens,
		Auth: auth.NewService(db, tokens, sms, secret, auth.DefaultConfig()),
	})}

	if status, _ := c.do("GET", "/healthz", nil); status != http.StatusOK {
		t.Fatalf("healthz = %d", status)
	}

	// Sans compte, aucun contenu n'est accessible (D19).
	if status, body := c.do("GET", "/api/v1/license-categories", nil); status != http.StatusUnauthorized || errorCode(body) != "unauthenticated" {
		t.Fatalf("accès anonyme = %d %v", status, body)
	}

	// Inscription : téléphone → code SMS → profil + PIN.
	status, body := c.do("POST", "/api/v1/auth/otp/request", map[string]string{"phone_number": "77 123 45 67", "purpose": "register"})
	if status != http.StatusAccepted {
		t.Fatalf("otp/request = %d %v", status, body)
	}
	if _, exposed := body["dev_code"]; exposed {
		t.Fatal("dev_code ne doit être renvoyé que si ExposeDevCodes est activé")
	}
	status, body = c.do("POST", "/api/v1/auth/otp/verify", map[string]string{
		"phone_number": "771234567", "purpose": "register", "code": sms.LastCode(t, "+221771234567"),
	})
	if status != http.StatusOK {
		t.Fatalf("otp/verify = %d %v", status, body)
	}
	register := map[string]string{
		"verification_token": body["verification_token"].(string),
		"display_name":       "Moussa Ndiaye", "preferred_language": "fr", "learning_mode": "text", "pin": "2580",
	}
	// Q13 : pas d'inscription sans accepter la version en vigueur des CGU.
	if status, body = c.do("POST", "/api/v1/auth/register", register); status != http.StatusBadRequest {
		t.Fatalf("inscription sans CGU = %d %v", status, body)
	}
	register["terms_version"] = "2000-01-01"
	if status, body = c.do("POST", "/api/v1/auth/register", register); status != http.StatusConflict || errorCode(body) != "terms_outdated" {
		t.Fatalf("inscription avec d'anciennes CGU = %d %v", status, body)
	}
	register["terms_version"] = users.TermsVersion
	status, body = c.do("POST", "/api/v1/auth/register", register)
	if status != http.StatusCreated {
		t.Fatalf("register = %d %v", status, body)
	}
	refresh := body["refresh_token"].(string)

	// Connexion par PIN.
	status, body = c.do("POST", "/api/v1/auth/login", map[string]string{"phone_number": "+221771234567", "pin": "2580"})
	if status != http.StatusOK {
		t.Fatalf("login = %d %v", status, body)
	}
	c.token = body["access_token"].(string)

	status, body = c.do("GET", "/api/v1/me", nil)
	if status != http.StatusOK || body["display_name"] != "Moussa Ndiaye" || body["phone_number"] != "+221771234567" ||
		body["terms_accepted"] != true || body["terms_version"] != users.TermsVersion {
		t.Fatalf("GET /me = %d %v", status, body)
	}
	if roles := body["roles"].([]any); len(roles) != 1 || roles[0] != "candidate" {
		t.Errorf("roles = %v", roles)
	}

	// Passage en wolof et en mode audio.
	status, body = c.do("PATCH", "/api/v1/me", map[string]string{"preferred_language": "wo", "learning_mode": "audio"})
	if status != http.StatusOK || body["preferred_language"] != "wo" || body["learning_mode"] != "audio" || body["display_name"] != "Moussa Ndiaye" {
		t.Fatalf("PATCH /me = %d %v", status, body)
	}
	if status, body = c.do("PATCH", "/api/v1/me", map[string]string{"preferred_language": "en"}); status != http.StatusBadRequest || errorCode(body) != "validation_error" {
		t.Errorf("langue non prise en charge = %d %v", status, body)
	}
	if status, body = c.do("PATCH", "/api/v1/me", map[string]any{"is_admin": true}); status != http.StatusBadRequest || errorCode(body) != "invalid_json" {
		t.Errorf("champ inconnu = %d %v", status, body)
	}

	// Catégories de permis B et C (CDC §5.1).
	status, body = c.do("GET", "/api/v1/license-categories", nil)
	if status != http.StatusOK {
		t.Fatalf("license-categories = %d %v", status, body)
	}
	items := body["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("catégories = %v", items)
	}
	b, cat := items[0].(map[string]any), items[1].(map[string]any)
	if b["code"] != "B" || b["series_count"] != 12.0 || b["questions_per_series"] != 25.0 || b["max_faults"] != 4.0 || b["exam_duration_minutes"] != 20.0 {
		t.Errorf("permis B = %v", b)
	}
	if cat["code"] != "C" || cat["series_count"] != 5.0 || cat["questions_per_series"] != 10.0 || cat["max_faults"] != 4.0 {
		t.Errorf("permis C = %v", cat)
	}

	// Déconnexion.
	if status, _ = c.do("POST", "/api/v1/auth/logout", map[string]string{"refresh_token": refresh}); status != http.StatusNoContent {
		t.Errorf("logout = %d", status)
	}
	if status, body = c.do("POST", "/api/v1/auth/refresh", map[string]string{"refresh_token": refresh}); status != http.StatusUnauthorized {
		t.Errorf("refresh après logout = %d %v", status, body)
	}

	// Jeton falsifié.
	c.token += "x"
	if status, _ = c.do("GET", "/api/v1/me", nil); status != http.StatusUnauthorized {
		t.Errorf("jeton falsifié = %d", status)
	}
}

func TestDevCodeExposure(t *testing.T) {
	db := testutil.DB(t)
	sms := &testutil.SMS{}
	secret := []byte("test-secret-test-secret-test-secret!")
	tokens := auth.NewTokens(secret, 15*time.Minute)
	svc := auth.NewService(db, tokens, sms, secret, auth.DefaultConfig())
	svc.ExposeDevCodes()
	c := &client{t: t, h: NewRouter(Deps{DB: db, Tokens: tokens, Auth: svc})}

	status, body := c.do("POST", "/api/v1/auth/otp/request", map[string]string{"phone_number": "78 000 00 02", "purpose": "register"})
	if status != http.StatusAccepted || body["dev_code"] != sms.LastCode(t, "+221780000002") {
		t.Fatalf("dev_code = %d %v", status, body)
	}
}
