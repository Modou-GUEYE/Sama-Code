package courses

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"samacode/backend/internal/database"
	"samacode/backend/internal/httpx"
)

// Côté candidat : uniquement les cours et leçons publiés (CDC §6.2), avec la
// progression du candidat (leçons terminées).

// Label : titre d'un thème, d'un cours ou d'une leçon dans une langue.
type Label struct {
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	AudioURL    *string `json:"audio_url"`
}

type CategoryView struct {
	ID      string           `json:"id"`
	Code    string           `json:"code"`
	Labels  map[string]Label `json:"labels"`
	Courses []CourseView     `json:"courses"`
}

type CourseView struct {
	ID               string           `json:"id"`
	LicenseCode      *string          `json:"license_code"`
	Labels           map[string]Label `json:"labels"`
	LessonCount      int              `json:"lesson_count"`
	CompletedLessons int              `json:"completed_lessons"`
	Lessons          []LessonSummary  `json:"lessons,omitempty"`
}

type LessonSummary struct {
	ID        string           `json:"id"`
	Labels    map[string]Label `json:"labels"`
	Completed bool             `json:"completed"`
}

// BlockView : bloc de leçon tel que vu par le candidat.
type BlockView struct {
	Type     string  `json:"type"` // text | audio | image
	Text     *string `json:"text"`
	MediaURL *string `json:"media_url"`
}

type LessonView struct {
	ID           string                 `json:"id"`
	CourseID     string                 `json:"course_id"`
	Labels       map[string]Label       `json:"labels"`
	Blocks       map[string][]BlockView `json:"blocks"`
	Completed    bool                   `json:"completed"`
	NextLessonID *string                `json:"next_lesson_id"`
}

func mediaURL(id *string) *string {
	if id == nil {
		return nil
	}
	u := "/api/v1/media/" + *id + "/file"
	return &u
}

func labels(tr Translations) map[string]Label {
	out := map[string]Label{}
	for lang, t := range tr {
		out[lang] = Label{Title: t.Title, Description: t.Description, AudioURL: mediaURL(t.AudioMediaID)}
	}
	return out
}

type CandidateHandler struct {
	db *pgxpool.Pool
}

func NewCandidateHandler(db *pgxpool.Pool) *CandidateHandler { return &CandidateHandler{db: db} }

// Routes : candidat connecté, sous /api/v1.
func (h *CandidateHandler) Routes(r chi.Router) {
	r.Get("/course-categories", httpx.Handle(h.catalog))
	r.Get("/courses/{id}", httpx.Handle(h.course))
	r.Get("/lessons/{id}", httpx.Handle(h.lesson))
	r.Post("/lessons/{id}/completion", httpx.Handle(h.complete))
}

func userID(r *http.Request) string { return httpx.MustPrincipal(r.Context()).UserID }

// catalog : thèmes contenant au moins un cours publié, avec leurs cours et la progression.
func (h *CandidateHandler) catalog(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	rows, err := h.db.Query(ctx,
		`SELECT c.id, c.category_id, lc.code,
		        (SELECT count(*) FROM lessons l WHERE l.course_id = c.id AND l.status = 'published'),
		        (SELECT count(*) FROM lessons l JOIN lesson_progress p ON p.lesson_id = l.id
		          WHERE l.course_id = c.id AND l.status = 'published' AND p.user_id = $1 AND p.status = 'completed')
		   FROM courses c LEFT JOIN license_categories lc ON lc.id = c.license_category_id
		  WHERE c.status = 'published'
		  ORDER BY c.sort_order, c.created_at`, userID(r))
	if err != nil {
		return err
	}
	type courseRow struct {
		CourseView
		categoryID string
	}
	courses, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (courseRow, error) {
		var c courseRow
		return c, row.Scan(&c.ID, &c.categoryID, &c.LicenseCode, &c.LessonCount, &c.CompletedLessons)
	})
	if err != nil {
		return err
	}

	byCategory := map[string][]CourseView{}
	for _, c := range courses {
		if c.LessonCount == 0 {
			continue // un cours publié sans leçon publiée n'a rien à montrer
		}
		tr, err := loadTranslations(ctx, h.db, "course_translations", "course_id", "title", c.ID)
		if err != nil {
			return err
		}
		c.Labels = labels(tr)
		byCategory[c.categoryID] = append(byCategory[c.categoryID], c.CourseView)
	}

	rows, err = h.db.Query(ctx, `SELECT id, code FROM course_categories ORDER BY sort_order, code`)
	if err != nil {
		return err
	}
	cats, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (CategoryView, error) {
		var c CategoryView
		return c, row.Scan(&c.ID, &c.Code)
	})
	if err != nil {
		return err
	}
	out := []CategoryView{}
	for _, c := range cats {
		if len(byCategory[c.ID]) == 0 {
			continue
		}
		tr, err := loadTranslations(ctx, h.db, "course_category_translations", "category_id", "name", c.ID)
		if err != nil {
			return err
		}
		c.Labels = labels(tr)
		c.Courses = byCategory[c.ID]
		out = append(out, c)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": out})
	return nil
}

func (h *CandidateHandler) course(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.UUIDParam(r, "id")
	if err != nil {
		return err
	}
	ctx := r.Context()
	var c CourseView
	err = h.db.QueryRow(ctx,
		`SELECT c.id, lc.code FROM courses c LEFT JOIN license_categories lc ON lc.id = c.license_category_id
		  WHERE c.id = $1 AND c.status = 'published'`, id).Scan(&c.ID, &c.LicenseCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	tr, err := loadTranslations(ctx, h.db, "course_translations", "course_id", "title", id)
	if err != nil {
		return err
	}
	c.Labels = labels(tr)

	rows, err := h.db.Query(ctx,
		`SELECT l.id, COALESCE(p.status = 'completed', false)
		   FROM lessons l LEFT JOIN lesson_progress p ON p.lesson_id = l.id AND p.user_id = $2
		  WHERE l.course_id = $1 AND l.status = 'published'
		  ORDER BY l.sort_order, l.created_at`, id, userID(r))
	if err != nil {
		return err
	}
	c.Lessons, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (LessonSummary, error) {
		var l LessonSummary
		return l, row.Scan(&l.ID, &l.Completed)
	})
	if err != nil {
		return err
	}
	for i := range c.Lessons {
		ltr, err := loadTranslations(ctx, h.db, "lesson_translations", "lesson_id", "title", c.Lessons[i].ID)
		if err != nil {
			return err
		}
		c.Lessons[i].Labels = labels(ltr)
		if c.Lessons[i].Completed {
			c.CompletedLessons++
		}
	}
	c.LessonCount = len(c.Lessons)
	httpx.WriteJSON(w, http.StatusOK, c)
	return nil
}

// publishedLesson vérifie qu'une leçon et son cours sont publiés, et renvoie le cours.
func publishedLesson(ctx context.Context, q database.Querier, id string) (courseID string, err error) {
	err = q.QueryRow(ctx,
		`SELECT l.course_id FROM lessons l JOIN courses c ON c.id = l.course_id
		  WHERE l.id = $1 AND l.status = 'published' AND c.status = 'published'`, id).Scan(&courseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", httpx.ErrNotFound
	}
	return courseID, err
}

func (h *CandidateHandler) lesson(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.UUIDParam(r, "id")
	if err != nil {
		return err
	}
	ctx := r.Context()
	courseID, err := publishedLesson(ctx, h.db, id)
	if err != nil {
		return err
	}
	out := LessonView{ID: id, CourseID: courseID, Blocks: map[string][]BlockView{}}
	tr, err := loadTranslations(ctx, h.db, "lesson_translations", "lesson_id", "title", id)
	if err != nil {
		return err
	}
	out.Labels = labels(tr)

	rows, err := h.db.Query(ctx,
		`SELECT language_code, content_type, text_content, media_id FROM lesson_contents
		  WHERE lesson_id = $1 ORDER BY language_code, sort_order`, id)
	if err != nil {
		return err
	}
	for rows.Next() {
		var lang string
		var b BlockView
		var mediaID *string
		if err := rows.Scan(&lang, &b.Type, &b.Text, &mediaID); err != nil {
			rows.Close()
			return err
		}
		b.MediaURL = mediaURL(mediaID)
		out.Blocks[lang] = append(out.Blocks[lang], b)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if err := h.db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM lesson_progress WHERE lesson_id = $1 AND user_id = $2 AND status = 'completed')`,
		id, userID(r)).Scan(&out.Completed); err != nil {
		return err
	}
	// Leçon suivante publiée du même cours, pour enchaîner.
	err = h.db.QueryRow(ctx,
		`SELECT n.id FROM lessons cur
		   JOIN lessons n ON n.course_id = cur.course_id AND n.status = 'published'
		              AND (n.sort_order, n.created_at) > (cur.sort_order, cur.created_at)
		  WHERE cur.id = $1 ORDER BY n.sort_order, n.created_at LIMIT 1`, id).Scan(&out.NextLessonID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, out)
	return nil
}

// complete marque la leçon comme terminée (idempotent).
func (h *CandidateHandler) complete(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.UUIDParam(r, "id")
	if err != nil {
		return err
	}
	if _, err := publishedLesson(r.Context(), h.db, id); err != nil {
		return err
	}
	now := time.Now()
	if _, err := h.db.Exec(r.Context(),
		`INSERT INTO lesson_progress (user_id, lesson_id, status, completed_at, updated_at) VALUES ($1, $2, 'completed', $3, $3)
		 ON CONFLICT (user_id, lesson_id) DO UPDATE SET status = 'completed',
		   completed_at = COALESCE(lesson_progress.completed_at, EXCLUDED.completed_at), updated_at = EXCLUDED.updated_at`,
		userID(r), id, now); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
