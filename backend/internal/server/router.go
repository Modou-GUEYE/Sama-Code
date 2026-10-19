// Package server assemble les modules en un routeur HTTP.
package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"samacode/backend/api"
	"samacode/backend/internal/admin"
	"samacode/backend/internal/auth"
	"samacode/backend/internal/courses"
	"samacode/backend/internal/examengine"
	"samacode/backend/internal/examseries"
	"samacode/backend/internal/httpx"
	"samacode/backend/internal/licenses"
	"samacode/backend/internal/media"
	"samacode/backend/internal/questions"
	"samacode/backend/internal/users"
	"samacode/backend/internal/workflow"
)

type Deps struct {
	DB                *pgxpool.Pool
	Tokens            *auth.Tokens
	Auth              *auth.Service
	Media             media.Storage
	TrustProxyHeaders bool
	// Docs sert la documentation Swagger sur /docs (désactivée en production par défaut).
	Docs bool
}

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	if d.TrustProxyHeaders {
		r.Use(middleware.RealIP)
	}
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := d.DB.Ping(r.Context()); err != nil {
			httpx.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	if d.Docs {
		r.Route("/docs", api.DocsRoutes)
	}

	r.Route("/api/v1", func(r chi.Router) {
		// 10 requêtes d'affilée par IP, puis 1 toutes les 6 s : en complément des quotas par numéro.
		authLimiter := httpx.NewRateLimiter(6*time.Second, 10)
		r.With(authLimiter.Middleware).Route("/auth", auth.NewHandler(d.Auth).Routes)

		mediaHandler := media.NewHandler(d.DB, d.Media)
		mediaHandler.PublicRoutes(r)

		r.Group(func(r chi.Router) {
			r.Use(d.Tokens.Middleware)
			users.NewHandler(d.DB).Routes(r)
			licenses.NewHandler(d.DB).Routes(r)
			examengine.NewHandler(examengine.NewService(d.DB)).Routes(r)
			courses.NewCandidateHandler(d.DB).Routes(r)

			// Administration des contenus : éditeurs, validateurs et administrateurs.
			// Les droits fins (publier, gérer les comptes) sont vérifiés plus bas ou dans les services.
			r.Route("/admin", func(r chi.Router) {
				r.Use(httpx.RequireRole(workflow.ContentRoles...))
				adminHandler := admin.NewHandler(d.DB)
				adminHandler.StatsRoutes(r)
				mediaHandler.AdminRoutes(r)
				questions.NewHandler(questions.NewService(d.DB)).AdminRoutes(r)
				examseries.NewHandler(examseries.NewService(d.DB)).AdminRoutes(r)
				courses.NewAdminHandler(d.DB).Routes(r)

				r.Group(func(r chi.Router) {
					r.Use(httpx.RequireRole(workflow.RoleAdmin))
					adminHandler.AdminOnlyRoutes(r)
				})
			})
		})
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) { httpx.WriteError(w, r, httpx.ErrNotFound) })
	return r
}
