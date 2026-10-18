// Package licenses expose les catégories de permis et leurs paramètres d'évaluation.
package licenses

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"samacode/backend/internal/httpx"
)

type Category struct {
	ID                  string `json:"id"`
	Code                string `json:"code"`
	Name                string `json:"name"`
	SeriesCount         int    `json:"series_count"`
	QuestionsPerSeries  int    `json:"questions_per_series"`
	MaxFaults           int    `json:"max_faults"`
	ExamDurationMinutes int    `json:"exam_duration_minutes"`
}

type Handler struct {
	db *pgxpool.Pool
}

func NewHandler(db *pgxpool.Pool) *Handler { return &Handler{db: db} }

func (h *Handler) Routes(r chi.Router) {
	r.Get("/license-categories", httpx.Handle(h.list))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) error {
	rows, err := h.db.Query(r.Context(),
		`SELECT id, code, name, series_count, questions_per_series, max_faults, exam_duration_minutes
		   FROM license_categories WHERE is_active ORDER BY sort_order, code`)
	if err != nil {
		return err
	}
	cats, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Category])
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": cats})
	return nil
}
