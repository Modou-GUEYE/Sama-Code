package httpx

import (
	"net/http"
	"regexp"
	"strconv"

	"github.com/go-chi/chi/v5"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func IsUUID(s string) bool { return uuidPattern.MatchString(s) }

// UUIDParam lit un identifiant dans l'URL ; un identifiant mal formé vaut « introuvable ».
func UUIDParam(r *http.Request, name string) (string, error) {
	v := chi.URLParam(r, name)
	if !IsUUID(v) {
		return "", ErrNotFound
	}
	return v, nil
}

// Page lit ?page=&page_size= (page à partir de 1, 20 éléments par défaut, 100 max).
type Page struct {
	Number int
	Size   int
}

func (p Page) Offset() int { return (p.Number - 1) * p.Size }

func PageParams(r *http.Request) Page {
	p := Page{Number: 1, Size: 20}
	if n, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && n > 0 {
		p.Number = n
	}
	if n, err := strconv.Atoi(r.URL.Query().Get("page_size")); err == nil && n > 0 {
		p.Size = min(n, 100)
	}
	return p
}
