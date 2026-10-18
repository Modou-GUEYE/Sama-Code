package httpx

import (
	"context"
	"net/http"
	"slices"
)

// Principal identifie l'utilisateur authentifié à l'origine d'une requête.
type Principal struct {
	UserID string
	Roles  []string
}

func (p Principal) HasRole(role string) bool { return slices.Contains(p.Roles, role) }

type principalKey struct{}

func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	return p, ok
}

// MustPrincipal est réservé aux handlers montés derrière le middleware d'authentification.
func MustPrincipal(ctx context.Context) Principal {
	p, ok := PrincipalFrom(ctx)
	if !ok {
		panic("httpx: handler protégé appelé sans principal")
	}
	return p
}

// RequireRole laisse passer un utilisateur possédant au moins un des rôles donnés.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := PrincipalFrom(r.Context())
			if !ok {
				WriteError(w, r, ErrUnauthenticated)
				return
			}
			if !slices.ContainsFunc(roles, p.HasRole) {
				WriteError(w, r, ErrForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
