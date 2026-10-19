// Package courses gère les cours théoriques : thèmes → cours → leçons → blocs de
// contenu, en français et en wolof (texte, audio, image).
package courses

import (
	"context"
	"errors"
	"net/http"
	"regexp"
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
	errBadReference = httpx.NewError(http.StatusBadRequest, "invalid_reference", "Un thème, un permis ou un média référencé n'existe pas.")
	errCodeTaken    = httpx.NewError(http.StatusConflict, "code_taken", "Ce code de thème est déjà utilisé.")
	errBadCode      = httpx.Validation("code", "Code invalide : lettres minuscules, chiffres et tirets (ex. panneaux).")
	errNoFrench     = httpx.Validation("translations.fr", "Le titre en français est obligatoire.")
	errIncomplete   = httpx.NewError(http.StatusUnprocessableEntity, "content_incomplete",
		"Le contenu doit exister en français et en wolof avant d'être soumis ou publié.")
	codePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,40}$`)
)

// Translation : libellé d'un thème, d'un cours ou d'une leçon dans une langue.
type Translation struct {
	Title        string  `json:"title"`
	Description  *string `json:"description,omitempty"`
	AudioMediaID *string `json:"audio_media_id"`
}

type Translations map[string]Translation

func (t Translations) validate() error {
	for lang, tr := range t {
		if lang != "fr" && lang != "wo" {
			return httpx.Validation("translations", "Langue non prise en charge : "+lang)
		}
		tr.Title = strings.TrimSpace(tr.Title)
		if tr.AudioMediaID != nil && !httpx.IsUUID(*tr.AudioMediaID) {
			return httpx.Validation("translations."+lang, "Identifiant de média invalide.")
		}
		t[lang] = tr
	}
	if t["fr"].Title == "" {
		return errNoFrench
	}
	return nil
}

func (t Translations) complete() bool { return t["fr"].Title != "" && t["wo"].Title != "" }

// Block : bloc de contenu d'une leçon (texte, audio ou image).
type Block struct {
	ContentType string  `json:"content_type"`
	Text        *string `json:"text"`
	MediaID     *string `json:"media_id"`
}

type Category struct {
	ID           string       `json:"id"`
	Code         string       `json:"code"`
	IconMediaID  *string      `json:"icon_media_id"`
	SortOrder    int          `json:"sort_order"`
	Translations Translations `json:"translations"`
	CourseCount  int          `json:"course_count"`
}

type Course struct {
	ID                string       `json:"id"`
	CategoryID        string       `json:"category_id"`
	LicenseCategoryID *string      `json:"license_category_id"`
	Status            string       `json:"status"`
	SortOrder         int          `json:"sort_order"`
	Translations      Translations `json:"translations"`
	Lessons           []Lesson     `json:"lessons,omitempty"`
	LessonCount       int          `json:"lesson_count"`
	UpdatedAt         time.Time    `json:"updated_at"`
}

type Lesson struct {
	ID           string             `json:"id"`
	CourseID     string             `json:"course_id"`
	Status       string             `json:"status"`
	SortOrder    int                `json:"sort_order"`
	Translations Translations       `json:"translations"`
	Contents     map[string][]Block `json:"contents,omitempty"`
}

type AdminHandler struct {
	db *pgxpool.Pool
}

func NewAdminHandler(db *pgxpool.Pool) *AdminHandler { return &AdminHandler{db: db} }

// Routes : montées sous /api/v1/admin, réservées aux rôles de contenu.
func (h *AdminHandler) Routes(r chi.Router) {
	r.Get("/course-categories", httpx.Handle(h.listCategories))
	r.Post("/course-categories", httpx.Handle(h.createCategory))
	r.Put("/course-categories/{id}", httpx.Handle(h.updateCategory))

	r.Get("/courses", httpx.Handle(h.listCourses))
	r.Post("/courses", httpx.Handle(h.createCourse))
	r.Get("/courses/{id}", httpx.Handle(h.getCourse))
	r.Put("/courses/{id}", httpx.Handle(h.updateCourse))
	r.Post("/courses/{id}/status", httpx.Handle(h.courseStatus))

	r.Post("/courses/{id}/lessons", httpx.Handle(h.createLesson))
	r.Get("/lessons/{id}", httpx.Handle(h.getLesson))
	r.Put("/lessons/{id}", httpx.Handle(h.updateLesson))
	r.Put("/lessons/{id}/contents", httpx.Handle(h.setLessonContents))
	r.Post("/lessons/{id}/status", httpx.Handle(h.lessonStatus))

	r.Delete("/course-categories/{id}", httpx.Handle(h.deleteCategory))
	r.Delete("/courses/{id}", httpx.Handle(h.deleteCourse))
	r.Delete("/lessons/{id}", httpx.Handle(h.deleteLesson))
}

// ---------------------------------------------------------------------------
// Thèmes
// ---------------------------------------------------------------------------

type categoryInput struct {
	Code         string       `json:"code"`
	IconMediaID  *string      `json:"icon_media_id"`
	SortOrder    int          `json:"sort_order"`
	Translations Translations `json:"translations"`
}

func (in *categoryInput) validate() error {
	in.Code = strings.TrimSpace(in.Code)
	if !codePattern.MatchString(in.Code) {
		return errBadCode
	}
	if in.IconMediaID != nil && !httpx.IsUUID(*in.IconMediaID) {
		return httpx.Validation("icon_media_id", "Identifiant de média invalide.")
	}
	return in.Translations.validate()
}

func (h *AdminHandler) listCategories(w http.ResponseWriter, r *http.Request) error {
	rows, err := h.db.Query(r.Context(),
		`SELECT c.id, c.code, c.icon_media_id, c.sort_order,
		        (SELECT count(*) FROM courses co WHERE co.category_id = c.id)
		   FROM course_categories c ORDER BY c.sort_order, c.code`)
	if err != nil {
		return err
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Category, error) {
		var c Category
		return c, row.Scan(&c.ID, &c.Code, &c.IconMediaID, &c.SortOrder, &c.CourseCount)
	})
	if err != nil {
		return err
	}
	for i := range items {
		if items[i].Translations, err = loadTranslations(r.Context(), h.db, "course_category_translations", "category_id", "name", items[i].ID); err != nil {
			return err
		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
	return nil
}

func (h *AdminHandler) createCategory(w http.ResponseWriter, r *http.Request) error {
	var in categoryInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	if err := in.validate(); err != nil {
		return err
	}
	p := httpx.MustPrincipal(r.Context())
	var id string
	err := database.WithTx(r.Context(), h.db, func(tx pgx.Tx) error {
		if err := tx.QueryRow(r.Context(),
			`INSERT INTO course_categories (code, icon_media_id, sort_order) VALUES ($1, $2, $3) RETURNING id`,
			in.Code, in.IconMediaID, in.SortOrder).Scan(&id); err != nil {
			return err
		}
		if err := writeTranslations(r.Context(), tx, "course_category_translations", "category_id", "name", id, in.Translations); err != nil {
			return err
		}
		return audit.Log(r.Context(), tx, audit.Entry{ActorUserID: &p.UserID, Action: "course_category.created", EntityType: "course_category", EntityID: &id})
	})
	if err != nil {
		return mapErr(err)
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]string{"id": id})
	return nil
}

func (h *AdminHandler) updateCategory(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.UUIDParam(r, "id")
	if err != nil {
		return err
	}
	var in categoryInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	if err := in.validate(); err != nil {
		return err
	}
	p := httpx.MustPrincipal(r.Context())
	err = database.WithTx(r.Context(), h.db, func(tx pgx.Tx) error {
		tag, err := tx.Exec(r.Context(),
			`UPDATE course_categories SET code = $2, icon_media_id = $3, sort_order = $4 WHERE id = $1`,
			id, in.Code, in.IconMediaID, in.SortOrder)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return httpx.ErrNotFound
		}
		if err := writeTranslations(r.Context(), tx, "course_category_translations", "category_id", "name", id, in.Translations); err != nil {
			return err
		}
		return audit.Log(r.Context(), tx, audit.Entry{ActorUserID: &p.UserID, Action: "course_category.updated", EntityType: "course_category", EntityID: &id})
	})
	if err != nil {
		return mapErr(err)
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// ---------------------------------------------------------------------------
// Cours
// ---------------------------------------------------------------------------

type courseInput struct {
	CategoryID        string       `json:"category_id"`
	LicenseCategoryID *string      `json:"license_category_id"`
	SortOrder         int          `json:"sort_order"`
	Translations      Translations `json:"translations"`
}

func (in *courseInput) validate() error {
	if !httpx.IsUUID(in.CategoryID) {
		return httpx.Validation("category_id", "Thème invalide.")
	}
	if in.LicenseCategoryID != nil && !httpx.IsUUID(*in.LicenseCategoryID) {
		return httpx.Validation("license_category_id", "Permis invalide.")
	}
	return in.Translations.validate()
}

func (h *AdminHandler) listCourses(w http.ResponseWriter, r *http.Request) error {
	categoryID := r.URL.Query().Get("category_id")
	if categoryID != "" && !httpx.IsUUID(categoryID) {
		return httpx.Validation("category_id", "Thème invalide.")
	}
	rows, err := h.db.Query(r.Context(),
		`SELECT c.id, c.category_id, c.license_category_id, c.status, c.sort_order, c.updated_at,
		        (SELECT count(*) FROM lessons l WHERE l.course_id = c.id)
		   FROM courses c WHERE ($1 = '' OR c.category_id::text = $1)
		  ORDER BY c.sort_order, c.created_at`, categoryID)
	if err != nil {
		return err
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Course, error) {
		var c Course
		return c, row.Scan(&c.ID, &c.CategoryID, &c.LicenseCategoryID, &c.Status, &c.SortOrder, &c.UpdatedAt, &c.LessonCount)
	})
	if err != nil {
		return err
	}
	for i := range items {
		if items[i].Translations, err = loadTranslations(r.Context(), h.db, "course_translations", "course_id", "title", items[i].ID); err != nil {
			return err
		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
	return nil
}

func (h *AdminHandler) loadCourse(ctx context.Context, id string) (Course, error) {
	var c Course
	err := h.db.QueryRow(ctx,
		`SELECT id, category_id, license_category_id, status, sort_order, updated_at FROM courses WHERE id = $1`, id,
	).Scan(&c.ID, &c.CategoryID, &c.LicenseCategoryID, &c.Status, &c.SortOrder, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Course{}, httpx.ErrNotFound
	}
	if err != nil {
		return Course{}, err
	}
	if c.Translations, err = loadTranslations(ctx, h.db, "course_translations", "course_id", "title", id); err != nil {
		return Course{}, err
	}
	rows, err := h.db.Query(ctx, `SELECT id, course_id, status, sort_order FROM lessons WHERE course_id = $1 ORDER BY sort_order, created_at`, id)
	if err != nil {
		return Course{}, err
	}
	c.Lessons, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (Lesson, error) {
		var l Lesson
		return l, row.Scan(&l.ID, &l.CourseID, &l.Status, &l.SortOrder)
	})
	if err != nil {
		return Course{}, err
	}
	for i := range c.Lessons {
		if c.Lessons[i].Translations, err = loadTranslations(ctx, h.db, "lesson_translations", "lesson_id", "title", c.Lessons[i].ID); err != nil {
			return Course{}, err
		}
	}
	c.LessonCount = len(c.Lessons)
	return c, nil
}

func (h *AdminHandler) getCourse(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.UUIDParam(r, "id")
	if err != nil {
		return err
	}
	c, err := h.loadCourse(r.Context(), id)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, c)
	return nil
}

func (h *AdminHandler) createCourse(w http.ResponseWriter, r *http.Request) error {
	var in courseInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	if err := in.validate(); err != nil {
		return err
	}
	p := httpx.MustPrincipal(r.Context())
	var id string
	err := database.WithTx(r.Context(), h.db, func(tx pgx.Tx) error {
		if err := tx.QueryRow(r.Context(),
			`INSERT INTO courses (category_id, license_category_id, sort_order) VALUES ($1, $2, $3) RETURNING id`,
			in.CategoryID, in.LicenseCategoryID, in.SortOrder).Scan(&id); err != nil {
			return err
		}
		if err := writeTranslations(r.Context(), tx, "course_translations", "course_id", "title", id, in.Translations); err != nil {
			return err
		}
		return audit.Log(r.Context(), tx, audit.Entry{ActorUserID: &p.UserID, Action: "course.created", EntityType: "course", EntityID: &id})
	})
	if err != nil {
		return mapErr(err)
	}
	c, err := h.loadCourse(r.Context(), id)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusCreated, c)
	return nil
}

func (h *AdminHandler) updateCourse(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.UUIDParam(r, "id")
	if err != nil {
		return err
	}
	var in courseInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	if err := in.validate(); err != nil {
		return err
	}
	p := httpx.MustPrincipal(r.Context())
	err = database.WithTx(r.Context(), h.db, func(tx pgx.Tx) error {
		// Modifier un cours publié le renvoie en brouillon : il doit être revalidé.
		tag, err := tx.Exec(r.Context(),
			`UPDATE courses SET category_id = $2, license_category_id = $3, sort_order = $4,
			        status = CASE WHEN status = 'published' THEN 'draft' ELSE status END
			  WHERE id = $1`, id, in.CategoryID, in.LicenseCategoryID, in.SortOrder)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return httpx.ErrNotFound
		}
		if err := writeTranslations(r.Context(), tx, "course_translations", "course_id", "title", id, in.Translations); err != nil {
			return err
		}
		return audit.Log(r.Context(), tx, audit.Entry{ActorUserID: &p.UserID, Action: "course.updated", EntityType: "course", EntityID: &id})
	})
	if err != nil {
		return mapErr(err)
	}
	c, err := h.loadCourse(r.Context(), id)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, c)
	return nil
}

func (h *AdminHandler) courseStatus(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.UUIDParam(r, "id")
	if err != nil {
		return err
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	if err := h.changeStatus(r.Context(), httpx.MustPrincipal(r.Context()), "courses", "course", "course_translations", "course_id", id, req.Status); err != nil {
		return err
	}
	c, err := h.loadCourse(r.Context(), id)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, c)
	return nil
}

// ---------------------------------------------------------------------------
// Leçons
// ---------------------------------------------------------------------------

type lessonInput struct {
	SortOrder    int          `json:"sort_order"`
	Translations Translations `json:"translations"`
}

func (h *AdminHandler) loadLesson(ctx context.Context, id string) (Lesson, error) {
	var l Lesson
	err := h.db.QueryRow(ctx, `SELECT id, course_id, status, sort_order FROM lessons WHERE id = $1`, id).
		Scan(&l.ID, &l.CourseID, &l.Status, &l.SortOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return Lesson{}, httpx.ErrNotFound
	}
	if err != nil {
		return Lesson{}, err
	}
	if l.Translations, err = loadTranslations(ctx, h.db, "lesson_translations", "lesson_id", "title", id); err != nil {
		return Lesson{}, err
	}
	rows, err := h.db.Query(ctx,
		`SELECT language_code, content_type, text_content, media_id FROM lesson_contents
		  WHERE lesson_id = $1 ORDER BY language_code, sort_order`, id)
	if err != nil {
		return Lesson{}, err
	}
	defer rows.Close()
	l.Contents = map[string][]Block{"fr": {}, "wo": {}}
	for rows.Next() {
		var lang string
		var b Block
		if err := rows.Scan(&lang, &b.ContentType, &b.Text, &b.MediaID); err != nil {
			return Lesson{}, err
		}
		l.Contents[lang] = append(l.Contents[lang], b)
	}
	return l, rows.Err()
}

func (h *AdminHandler) getLesson(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.UUIDParam(r, "id")
	if err != nil {
		return err
	}
	l, err := h.loadLesson(r.Context(), id)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, l)
	return nil
}

func (h *AdminHandler) createLesson(w http.ResponseWriter, r *http.Request) error {
	courseID, err := httpx.UUIDParam(r, "id")
	if err != nil {
		return err
	}
	var in lessonInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	if err := in.Translations.validate(); err != nil {
		return err
	}
	p := httpx.MustPrincipal(r.Context())
	var id string
	err = database.WithTx(r.Context(), h.db, func(tx pgx.Tx) error {
		if err := tx.QueryRow(r.Context(),
			`INSERT INTO lessons (course_id, sort_order) VALUES ($1, $2) RETURNING id`, courseID, in.SortOrder).Scan(&id); err != nil {
			return err
		}
		if err := writeTranslations(r.Context(), tx, "lesson_translations", "lesson_id", "title", id, in.Translations); err != nil {
			return err
		}
		return audit.Log(r.Context(), tx, audit.Entry{ActorUserID: &p.UserID, Action: "lesson.created", EntityType: "lesson", EntityID: &id})
	})
	if database.IsForeignKeyViolation(err) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	l, err := h.loadLesson(r.Context(), id)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusCreated, l)
	return nil
}

func (h *AdminHandler) updateLesson(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.UUIDParam(r, "id")
	if err != nil {
		return err
	}
	var in lessonInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	if err := in.Translations.validate(); err != nil {
		return err
	}
	p := httpx.MustPrincipal(r.Context())
	err = database.WithTx(r.Context(), h.db, func(tx pgx.Tx) error {
		tag, err := tx.Exec(r.Context(),
			`UPDATE lessons SET sort_order = $2, status = CASE WHEN status = 'published' THEN 'draft' ELSE status END
			  WHERE id = $1`, id, in.SortOrder)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return httpx.ErrNotFound
		}
		if err := writeTranslations(r.Context(), tx, "lesson_translations", "lesson_id", "title", id, in.Translations); err != nil {
			return err
		}
		return audit.Log(r.Context(), tx, audit.Entry{ActorUserID: &p.UserID, Action: "lesson.updated", EntityType: "lesson", EntityID: &id})
	})
	if err != nil {
		return mapErr(err)
	}
	l, err := h.loadLesson(r.Context(), id)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, l)
	return nil
}

// setLessonContents remplace les blocs d'une leçon : {"fr": [...], "wo": [...]}.
func (h *AdminHandler) setLessonContents(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.UUIDParam(r, "id")
	if err != nil {
		return err
	}
	var in map[string][]Block
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	for lang, blocks := range in {
		if lang != "fr" && lang != "wo" {
			return httpx.Validation("contents", "Langue non prise en charge : "+lang)
		}
		for i, b := range blocks {
			switch b.ContentType {
			case "text":
				if b.Text == nil || strings.TrimSpace(*b.Text) == "" {
					return httpx.Validation("contents."+lang, "Un bloc texte ne peut pas être vide.")
				}
				blocks[i].MediaID = nil
			case "audio", "image":
				if b.MediaID == nil || !httpx.IsUUID(*b.MediaID) {
					return httpx.Validation("contents."+lang, "Un bloc audio ou image a besoin d'un média.")
				}
			default:
				return httpx.Validation("contents."+lang, "Type de bloc invalide (text, audio, image).")
			}
		}
	}
	p := httpx.MustPrincipal(r.Context())
	err = database.WithTx(r.Context(), h.db, func(tx pgx.Tx) error {
		tag, err := tx.Exec(r.Context(),
			`UPDATE lessons SET status = CASE WHEN status = 'published' THEN 'draft' ELSE status END WHERE id = $1`, id)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return httpx.ErrNotFound
		}
		if _, err := tx.Exec(r.Context(), `DELETE FROM lesson_contents WHERE lesson_id = $1`, id); err != nil {
			return err
		}
		for lang, blocks := range in {
			for i, b := range blocks {
				if _, err := tx.Exec(r.Context(),
					`INSERT INTO lesson_contents (lesson_id, language_code, content_type, sort_order, text_content, media_id)
					 VALUES ($1, $2, $3, $4, $5, $6)`, id, lang, b.ContentType, i, b.Text, b.MediaID); err != nil {
					return err
				}
			}
		}
		return audit.Log(r.Context(), tx, audit.Entry{ActorUserID: &p.UserID, Action: "lesson.contents_set", EntityType: "lesson", EntityID: &id})
	})
	if err != nil {
		return mapErr(err)
	}
	l, err := h.loadLesson(r.Context(), id)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, l)
	return nil
}

func (h *AdminHandler) lessonStatus(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.UUIDParam(r, "id")
	if err != nil {
		return err
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	if err := h.changeStatus(r.Context(), httpx.MustPrincipal(r.Context()), "lessons", "lesson", "lesson_translations", "lesson_id", id, req.Status); err != nil {
		return err
	}
	l, err := h.loadLesson(r.Context(), id)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, l)
	return nil
}

// changeStatus applique le cycle de publication à un cours ou une leçon.
// Soumettre ou publier exige le titre en fr et en wo (et du contenu dans les deux langues pour une leçon).
func (h *AdminHandler) changeStatus(ctx context.Context, p httpx.Principal, table, entity, trTable, fk, id, status string) error {
	return database.WithTx(ctx, h.db, func(tx pgx.Tx) error {
		var current string
		err := tx.QueryRow(ctx, `SELECT status FROM `+table+` WHERE id = $1 FOR UPDATE`, id).Scan(&current)
		if errors.Is(err, pgx.ErrNoRows) {
			return httpx.ErrNotFound
		}
		if err != nil {
			return err
		}
		if err := workflow.Check(current, status, p); err != nil {
			return err
		}
		if status == workflow.InReview || status == workflow.Published {
			tr, err := loadTranslations(ctx, tx, trTable, fk, "title", id)
			if err != nil {
				return err
			}
			if !tr.complete() {
				return errIncomplete.With("missing", "translations")
			}
			if entity == "lesson" {
				var langs int
				if err := tx.QueryRow(ctx, `SELECT count(DISTINCT language_code) FROM lesson_contents WHERE lesson_id = $1`, id).Scan(&langs); err != nil {
					return err
				}
				if langs < 2 {
					return errIncomplete.With("missing", "contents")
				}
			}
		}
		if _, err := tx.Exec(ctx, `UPDATE `+table+` SET status = $2 WHERE id = $1`, id, status); err != nil {
			return err
		}
		return audit.Log(ctx, tx, audit.Entry{
			ActorUserID: &p.UserID, Action: entity + ".status", EntityType: entity, EntityID: &id,
			Metadata: map[string]any{"from": current, "to": status},
		})
	})
}

// ---------------------------------------------------------------------------
// Traductions (tables *_translations)
// ---------------------------------------------------------------------------

// loadTranslations lit une table de traductions ; titleCol vaut « name » pour les thèmes.
// Les noms de table et de colonnes sont des constantes du code, jamais des entrées utilisateur.
func loadTranslations(ctx context.Context, q database.Querier, table, fk, titleCol, id string) (Translations, error) {
	descCol := "NULL::text"
	if table == "course_category_translations" {
		descCol = "description"
	}
	rows, err := q.Query(ctx,
		`SELECT language_code, `+titleCol+`, `+descCol+`, audio_media_id FROM `+table+` WHERE `+fk+` = $1`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := Translations{}
	for rows.Next() {
		var lang string
		var t Translation
		if err := rows.Scan(&lang, &t.Title, &t.Description, &t.AudioMediaID); err != nil {
			return nil, err
		}
		out[lang] = t
	}
	return out, rows.Err()
}

func writeTranslations(ctx context.Context, tx pgx.Tx, table, fk, titleCol, id string, tr Translations) error {
	if _, err := tx.Exec(ctx, `DELETE FROM `+table+` WHERE `+fk+` = $1`, id); err != nil {
		return err
	}
	for lang, t := range tr {
		if t.Title == "" {
			continue
		}
		var err error
		if table == "course_category_translations" {
			_, err = tx.Exec(ctx,
				`INSERT INTO course_category_translations (category_id, language_code, name, description, audio_media_id)
				 VALUES ($1, $2, $3, $4, $5)`, id, lang, t.Title, t.Description, t.AudioMediaID)
		} else {
			_, err = tx.Exec(ctx,
				`INSERT INTO `+table+` (`+fk+`, language_code, `+titleCol+`, audio_media_id) VALUES ($1, $2, $3, $4)`,
				id, lang, t.Title, t.AudioMediaID)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func mapErr(err error) error {
	switch {
	case database.IsUniqueViolation(err):
		return errCodeTaken
	case database.IsForeignKeyViolation(err):
		return errBadReference
	}
	return err
}

// ---------------------------------------------------------------------------
// Suppression
// ---------------------------------------------------------------------------

var errCategoryNotEmpty = httpx.NewError(http.StatusConflict, "category_not_empty",
	"Ce thème contient encore des cours : supprimez-les ou déplacez-les d'abord.")

// Un thème n'a pas de cycle de publication : sa suppression est réservée au validateur
// et à l'administrateur, et refusée tant qu'il contient des cours.
func (h *AdminHandler) deleteCategory(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.UUIDParam(r, "id")
	if err != nil {
		return err
	}
	p := httpx.MustPrincipal(r.Context())
	if err := workflow.CheckDelete(workflow.Published, p); err != nil {
		return err
	}
	err = database.WithTx(r.Context(), h.db, func(tx pgx.Tx) error {
		var code string
		var courses int
		err := tx.QueryRow(r.Context(),
			`SELECT code, (SELECT count(*) FROM courses WHERE category_id = c.id) FROM course_categories c WHERE id = $1 FOR UPDATE`,
			id).Scan(&code, &courses)
		if errors.Is(err, pgx.ErrNoRows) {
			return httpx.ErrNotFound
		}
		if err != nil {
			return err
		}
		if courses > 0 {
			return errCategoryNotEmpty.With("courses", courses)
		}
		if _, err := tx.Exec(r.Context(), `DELETE FROM course_categories WHERE id = $1`, id); err != nil {
			return err
		}
		return audit.Log(r.Context(), tx, audit.Entry{
			ActorUserID: &p.UserID, Action: "course_category.deleted", EntityType: "course_category", EntityID: &id,
			Metadata: map[string]any{"code": code},
		})
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// Supprimer un cours supprime ses leçons et leurs blocs (les questions liées perdent
// seulement leur lien « revoir le cours »).
func (h *AdminHandler) deleteCourse(w http.ResponseWriter, r *http.Request) error {
	return h.deleteEntity(w, r, "courses", "course", "course_translations", "course_id")
}

func (h *AdminHandler) deleteLesson(w http.ResponseWriter, r *http.Request) error {
	return h.deleteEntity(w, r, "lessons", "lesson", "lesson_translations", "lesson_id")
}

func (h *AdminHandler) deleteEntity(w http.ResponseWriter, r *http.Request, table, entity, trTable, fk string) error {
	id, err := httpx.UUIDParam(r, "id")
	if err != nil {
		return err
	}
	p := httpx.MustPrincipal(r.Context())
	err = database.WithTx(r.Context(), h.db, func(tx pgx.Tx) error {
		var status string
		err := tx.QueryRow(r.Context(), `SELECT status FROM `+table+` WHERE id = $1 FOR UPDATE`, id).Scan(&status)
		if errors.Is(err, pgx.ErrNoRows) {
			return httpx.ErrNotFound
		}
		if err != nil {
			return err
		}
		if err := workflow.CheckDelete(status, p); err != nil {
			return err
		}
		tr, err := loadTranslations(r.Context(), tx, trTable, fk, "title", id)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(r.Context(), `DELETE FROM `+table+` WHERE id = $1`, id); err != nil {
			return err
		}
		return audit.Log(r.Context(), tx, audit.Entry{
			ActorUserID: &p.UserID, Action: entity + ".deleted", EntityType: entity, EntityID: &id,
			Metadata: map[string]any{"status": status, "title_fr": tr["fr"].Title},
		})
	})
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
