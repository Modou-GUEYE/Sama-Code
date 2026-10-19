package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"

	"samacode/backend/internal/auth"
)

// Le contrat backend/api/openapi.yaml décrit exactement les routes de l'API : aucune
// route oubliée, aucune route documentée qui n'existe plus.
func TestOpenAPICoversAllRoutes(t *testing.T) {
	raw, err := os.ReadFile("../../api/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		Paths map[string]map[string]any `yaml:"paths"`
	}
	if err := yaml.Unmarshal(raw, &spec); err != nil {
		t.Fatalf("openapi.yaml invalide : %v", err)
	}
	documented := map[string]bool{}
	for path, ops := range spec.Paths {
		for method := range ops {
			if slices.Contains([]string{"get", "post", "put", "patch", "delete"}, method) {
				documented[strings.ToUpper(method)+" "+path] = true
			}
		}
	}

	router := NewRouter(Deps{Tokens: auth.NewTokens([]byte("x"), time.Minute)}).(chi.Routes)
	served := map[string]bool{}
	_ = chi.Walk(router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		served[method+" "+strings.TrimPrefix(route, "/api/v1")] = true
		return nil
	})

	var missing, stale []string
	for r := range served {
		if !documented[r] {
			missing = append(missing, r)
		}
	}
	for r := range documented {
		if !served[r] {
			stale = append(stale, r)
		}
	}
	slices.Sort(missing)
	slices.Sort(stale)
	if len(missing) > 0 {
		t.Errorf("routes absentes de openapi.yaml :\n  %s", strings.Join(missing, "\n  "))
	}
	if len(stale) > 0 {
		t.Errorf("routes documentées qui n'existent pas :\n  %s", strings.Join(stale, "\n  "))
	}
}

func TestDocsServed(t *testing.T) {
	router := NewRouter(Deps{Tokens: auth.NewTokens([]byte("x"), time.Minute), Docs: true})
	get := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		return w
	}
	if w := get("/docs"); w.Code != http.StatusMovedPermanently || w.Header().Get("Location") != "/docs/" {
		t.Errorf("/docs = %d %q", w.Code, w.Header().Get("Location"))
	}
	if w := get("/docs/"); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "SwaggerUIBundle") {
		t.Errorf("/docs/ = %d", w.Code)
	}
	if w := get("/docs/openapi.yaml"); w.Code != http.StatusOK || !strings.HasPrefix(w.Body.String(), "openapi:") {
		t.Errorf("/docs/openapi.yaml = %d", w.Code)
	}
	// Désactivée : introuvable.
	off := NewRouter(Deps{Tokens: auth.NewTokens([]byte("x"), time.Minute)})
	w := httptest.NewRecorder()
	off.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/docs/", nil))
	if w.Code != http.StatusNotFound {
		t.Errorf("/docs/ désactivée = %d", w.Code)
	}
}
