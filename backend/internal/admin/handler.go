// Package admin regroupe les opérations réservées à l'administrateur : comptes et
// rôles, paramètres des catégories de permis, et statistiques du tableau de bord.
package admin

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"samacode/backend/internal/audit"
	"samacode/backend/internal/database"
	"samacode/backend/internal/httpx"
	"samacode/backend/internal/workflow"
)

var (
	allRoles        = []string{"candidate", workflow.RoleEditor, workflow.RoleValidator, workflow.RoleAdmin}
	errInvalidRole  = httpx.Validation("roles", "Rôle inconnu (candidate, editor, validator, admin).")
	errSelfDemotion = httpx.NewError(http.StatusConflict, "self_demotion", "Vous ne pouvez pas retirer votre propre rôle d'administrateur.")
)

type Handler struct {
	db *pgxpool.Pool
}

func NewHandler(db *pgxpool.Pool) *Handler { return &Handler{db: db} }

// StatsRoutes : tableau de bord, visible par tous les rôles de contenu.
func (h *Handler) StatsRoutes(r chi.Router) {
	r.Get("/statistics", httpx.Handle(h.statistics))
	r.Get("/license-categories", httpx.Handle(h.listLicenses))
}

// AdminOnlyRoutes : réservées au rôle admin.
func (h *Handler) AdminOnlyRoutes(r chi.Router) {
	r.Get("/users", httpx.Handle(h.listUsers))
	r.Put("/users/{id}/roles", httpx.Handle(h.setRoles))
	r.Patch("/license-categories/{id}", httpx.Handle(h.updateLicense))
}

// ---------------------------------------------------------------------------
// Statistiques
// ---------------------------------------------------------------------------

func (h *Handler) statistics(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	out := map[string]any{}
	var candidates, staff, sessions int
	if err := h.db.QueryRow(ctx,
		`SELECT (SELECT count(*) FROM user_roles ur JOIN roles ro ON ro.id = ur.role_id WHERE ro.code = 'candidate'),
		        (SELECT count(DISTINCT ur.user_id) FROM user_roles ur JOIN roles ro ON ro.id = ur.role_id WHERE ro.code <> 'candidate'),
		        (SELECT count(*) FROM exam_sessions)`).Scan(&candidates, &staff, &sessions); err != nil {
		return err
	}
	out["candidates"] = candidates
	out["staff"] = staff
	out["exam_sessions"] = sessions

	for key, table := range map[string]string{"questions": "questions", "courses": "courses", "lessons": "lessons", "series": "exam_series"} {
		byStatus, err := countByStatus(ctx, h.db, table)
		if err != nil {
			return err
		}
		out[key] = byStatus
	}

	rows, err := h.db.Query(ctx,
		`SELECT lc.code, lc.series_count,
		        (SELECT count(*) FROM exam_series s WHERE s.license_category_id = lc.id AND s.status = 'published')
		   FROM license_categories lc ORDER BY lc.sort_order`)
	if err != nil {
		return err
	}
	type licenseStat struct {
		Code      string `json:"code"`
		Expected  int    `json:"series_expected"`
		Published int    `json:"series_published"`
	}
	licenses, err := pgx.CollectRows(rows, pgx.RowToStructByPos[licenseStat])
	if err != nil {
		return err
	}
	out["licenses"] = licenses
	httpx.WriteJSON(w, http.StatusOK, out)
	return nil
}

// countByStatus : {"draft": n, "in_review": n, "published": n, "archived": n}.
func countByStatus(ctx context.Context, db *pgxpool.Pool, table string) (map[string]int, error) {
	rows, err := db.Query(ctx, `SELECT status, count(*) FROM `+table+` GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{workflow.Draft: 0, workflow.InReview: 0, workflow.Published: 0, workflow.Archived: 0}
	for rows.Next() {
		var status string
		var n int
		if err := rows.Scan(&status, &n); err != nil {
			return nil, err
		}
		out[status] = n
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// Utilisateurs et rôles
// ---------------------------------------------------------------------------

type User struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"display_name"`
	PhoneNumber string    `json:"phone_number"`
	Status      string    `json:"status"`
	Roles       []string  `json:"roles"`
	CreatedAt   time.Time `json:"created_at"`
}

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) error {
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	page := httpx.PageParams(r)
	rows, err := h.db.Query(r.Context(),
		`SELECT u.id, u.display_name, u.phone_number, u.status,
		        COALESCE(array_agg(ro.code ORDER BY ro.code) FILTER (WHERE ro.code IS NOT NULL), '{}'), u.created_at
		   FROM users u
		   LEFT JOIN user_roles ur ON ur.user_id = u.id
		   LEFT JOIN roles ro ON ro.id = ur.role_id
		  WHERE $1 = '' OR u.display_name ILIKE '%' || $1 || '%' OR u.phone_number LIKE '%' || $1 || '%'
		  GROUP BY u.id ORDER BY u.created_at DESC LIMIT $2 OFFSET $3`,
		search, page.Size, page.Offset())
	if err != nil {
		return err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[User])
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
	return nil
}

// setRoles remplace les rôles d'un utilisateur. Prend effet au prochain
// renouvellement de son jeton d'accès (15 min au plus).
func (h *Handler) setRoles(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.UUIDParam(r, "id")
	if err != nil {
		return err
	}
	var req struct {
		Roles []string `json:"roles"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	slices.Sort(req.Roles)
	req.Roles = slices.Compact(req.Roles)
	for _, role := range req.Roles {
		if !slices.Contains(allRoles, role) {
			return errInvalidRole
		}
	}
	p := httpx.MustPrincipal(r.Context())
	if id == p.UserID && !slices.Contains(req.Roles, workflow.RoleAdmin) {
		return errSelfDemotion
	}
	err = database.WithTx(r.Context(), h.db, func(tx pgx.Tx) error {
		var exists bool
		if err := tx.QueryRow(r.Context(), `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, id).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return httpx.ErrNotFound
		}
		if _, err := tx.Exec(r.Context(), `DELETE FROM user_roles WHERE user_id = $1`, id); err != nil {
			return err
		}
		if _, err := tx.Exec(r.Context(),
			`INSERT INTO user_roles (user_id, role_id) SELECT $1, id FROM roles WHERE code = ANY($2)`, id, req.Roles); err != nil {
			return err
		}
		return audit.Log(r.Context(), tx, audit.Entry{
			ActorUserID: &p.UserID, Action: "user.roles_set", EntityType: "user", EntityID: &id,
			Metadata: map[string]any{"roles": req.Roles},
		})
	})
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"id": id, "roles": req.Roles})
	return nil
}

// ---------------------------------------------------------------------------
// Catégories de permis (barème, durée d'examen)
// ---------------------------------------------------------------------------

type License struct {
	ID                  string `json:"id"`
	Code                string `json:"code"`
	Name                string `json:"name"`
	SeriesCount         int    `json:"series_count"`
	QuestionsPerSeries  int    `json:"questions_per_series"`
	MaxFaults           int    `json:"max_faults"`
	ExamDurationMinutes int    `json:"exam_duration_minutes"`
	IsActive            bool   `json:"is_active"`
}

func (h *Handler) listLicenses(w http.ResponseWriter, r *http.Request) error {
	rows, err := h.db.Query(r.Context(),
		`SELECT id, code, name, series_count, questions_per_series, max_faults, exam_duration_minutes, is_active
		   FROM license_categories ORDER BY sort_order, code`)
	if err != nil {
		return err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[License])
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
	return nil
}

type licensePatch struct {
	Name                *string `json:"name"`
	SeriesCount         *int    `json:"series_count"`
	QuestionsPerSeries  *int    `json:"questions_per_series"`
	MaxFaults           *int    `json:"max_faults"`
	ExamDurationMinutes *int    `json:"exam_duration_minutes"`
	IsActive            *bool   `json:"is_active"`
}

func (h *Handler) updateLicense(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.UUIDParam(r, "id")
	if err != nil {
		return err
	}
	var in licensePatch
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	switch {
	case in.Name != nil && strings.TrimSpace(*in.Name) == "":
		return httpx.Validation("name", "Le nom est obligatoire.")
	case in.SeriesCount != nil && (*in.SeriesCount < 1 || *in.SeriesCount > 100):
		return httpx.Validation("series_count", "Entre 1 et 100 séries.")
	case in.QuestionsPerSeries != nil && (*in.QuestionsPerSeries < 1 || *in.QuestionsPerSeries > 100):
		return httpx.Validation("questions_per_series", "Entre 1 et 100 questions.")
	case in.MaxFaults != nil && *in.MaxFaults < 0:
		return httpx.Validation("max_faults", "Le nombre de fautes autorisées ne peut pas être négatif.")
	case in.ExamDurationMinutes != nil && (*in.ExamDurationMinutes < 1 || *in.ExamDurationMinutes > 180):
		return httpx.Validation("exam_duration_minutes", "Entre 1 et 180 minutes.")
	}
	p := httpx.MustPrincipal(r.Context())
	var out License
	err = database.WithTx(r.Context(), h.db, func(tx pgx.Tx) error {
		err := tx.QueryRow(r.Context(),
			`UPDATE license_categories SET
			    name = COALESCE($2, name),
			    series_count = COALESCE($3, series_count),
			    questions_per_series = COALESCE($4, questions_per_series),
			    max_faults = COALESCE($5, max_faults),
			    exam_duration_minutes = COALESCE($6, exam_duration_minutes),
			    is_active = COALESCE($7, is_active)
			  WHERE id = $1
			  RETURNING id, code, name, series_count, questions_per_series, max_faults, exam_duration_minutes, is_active`,
			id, in.Name, in.SeriesCount, in.QuestionsPerSeries, in.MaxFaults, in.ExamDurationMinutes, in.IsActive,
		).Scan(&out.ID, &out.Code, &out.Name, &out.SeriesCount, &out.QuestionsPerSeries, &out.MaxFaults, &out.ExamDurationMinutes, &out.IsActive)
		if err == pgx.ErrNoRows {
			return httpx.ErrNotFound
		}
		if err != nil {
			return err
		}
		return audit.Log(r.Context(), tx, audit.Entry{
			ActorUserID: &p.UserID, Action: "license_category.updated", EntityType: "license_category", EntityID: &id,
			Metadata: map[string]any{"changes": in},
		})
	})
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, out)
	return nil
}
