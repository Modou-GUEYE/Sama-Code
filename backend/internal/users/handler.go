package users

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"samacode/backend/internal/httpx"
)

type Handler struct {
	db *pgxpool.Pool
}

func NewHandler(db *pgxpool.Pool) *Handler { return &Handler{db: db} }

// Routes doit être monté derrière le middleware d'authentification.
func (h *Handler) Routes(r chi.Router) {
	r.Get("/me", httpx.Handle(h.getMe))
	r.Patch("/me", httpx.Handle(h.patchMe))
}

type Me struct {
	ID                string    `json:"id"`
	DisplayName       string    `json:"display_name"`
	PhoneNumber       string    `json:"phone_number"`
	PreferredLanguage string    `json:"preferred_language"`
	LearningMode      string    `json:"learning_mode"`
	Roles             []string  `json:"roles"`
	CreatedAt         time.Time `json:"created_at"`
}

func (h *Handler) getMe(w http.ResponseWriter, r *http.Request) error {
	me, err := h.load(r.Context(), httpx.MustPrincipal(r.Context()).UserID)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, me)
	return nil
}

type patchMeRequest struct {
	DisplayName       *string `json:"display_name"`
	PreferredLanguage *string `json:"preferred_language"`
	LearningMode      *string `json:"learning_mode"`
}

func (h *Handler) patchMe(w http.ResponseWriter, r *http.Request) error {
	var req patchMeRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	if req.DisplayName != nil {
		name, err := NormalizeDisplayName(*req.DisplayName)
		if err != nil {
			return err
		}
		req.DisplayName = &name
	}
	if req.PreferredLanguage != nil {
		if err := ValidateLanguage(*req.PreferredLanguage); err != nil {
			return err
		}
	}
	if req.LearningMode != nil {
		if err := ValidateLearningMode(*req.LearningMode); err != nil {
			return err
		}
	}

	userID := httpx.MustPrincipal(r.Context()).UserID
	tag, err := h.db.Exec(r.Context(),
		`UPDATE users SET
			display_name       = COALESCE($2, display_name),
			preferred_language = COALESCE($3, preferred_language),
			learning_mode      = COALESCE($4, learning_mode)
		 WHERE id = $1 AND status = 'active'`,
		userID, req.DisplayName, req.PreferredLanguage, req.LearningMode)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return httpx.ErrNotFound
	}

	me, err := h.load(r.Context(), userID)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, me)
	return nil
}

func (h *Handler) load(ctx context.Context, userID string) (Me, error) {
	var me Me
	err := h.db.QueryRow(ctx,
		`SELECT u.id, u.display_name, u.phone_number, u.preferred_language, u.learning_mode, u.created_at,
		        COALESCE(array_agg(r.code ORDER BY r.code) FILTER (WHERE r.code IS NOT NULL), '{}')
		   FROM users u
		   LEFT JOIN user_roles ur ON ur.user_id = u.id
		   LEFT JOIN roles r ON r.id = ur.role_id
		  WHERE u.id = $1 AND u.status = 'active'
		  GROUP BY u.id`,
		userID).Scan(&me.ID, &me.DisplayName, &me.PhoneNumber, &me.PreferredLanguage, &me.LearningMode, &me.CreatedAt, &me.Roles)
	if errors.Is(err, pgx.ErrNoRows) {
		return Me{}, httpx.ErrNotFound
	}
	return me, err
}
