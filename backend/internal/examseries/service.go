// Package examseries gère les séries d'examen fixes (CDC §5.1) : composition,
// ordre des questions, cycle de publication et instantanés immuables (D15).
package examseries

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"samacode/backend/internal/audit"
	"samacode/backend/internal/database"
	"samacode/backend/internal/httpx"
	"samacode/backend/internal/questions"
	"samacode/backend/internal/workflow"
)

var (
	errSeriesExists   = httpx.NewError(http.StatusConflict, "series_exists", "Cette série existe déjà pour ce permis.")
	errSeriesNumber   = httpx.Validation("series_number", "Numéro de série invalide pour ce permis.")
	errUnknownLicense = httpx.Validation("license_category_id", "Catégorie de permis inconnue.")
	errBadQuestions   = httpx.Validation("question_ids", "Liste de questions invalide (doublon, question inconnue ou archivée).")
	errTooMany        = httpx.Validation("question_ids", "Trop de questions pour ce permis.")
	errWrongCount     = httpx.NewError(http.StatusUnprocessableEntity, "series_incomplete", "La série doit contenir exactement le nombre de questions prévu pour ce permis.")
	errNotPublished   = httpx.NewError(http.StatusUnprocessableEntity, "questions_not_published", "Toutes les questions de la série doivent être publiées.")
)

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service { return &Service{db: db} }

type Series struct {
	ID                 string     `json:"id"`
	LicenseCategoryID  string     `json:"license_category_id"`
	LicenseCode        string     `json:"license_code"`
	SeriesNumber       int        `json:"series_number"`
	Status             string     `json:"status"`
	CurrentVersion     int        `json:"current_version"`
	QuestionCount      int        `json:"question_count"`
	QuestionsPerSeries int        `json:"questions_per_series"`
	PublishedAt        *time.Time `json:"published_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

const seriesSelect = `
	SELECT s.id, s.license_category_id, lc.code, s.series_number, s.status, s.current_version,
	       (SELECT count(*) FROM series_questions sq WHERE sq.series_id = s.id),
	       lc.questions_per_series,
	       (SELECT max(published_at) FROM exam_series_snapshots sn WHERE sn.series_id = s.id),
	       s.updated_at
	  FROM exam_series s JOIN license_categories lc ON lc.id = s.license_category_id`

func scanSeries(row pgx.Row) (Series, error) {
	var s Series
	err := row.Scan(&s.ID, &s.LicenseCategoryID, &s.LicenseCode, &s.SeriesNumber, &s.Status, &s.CurrentVersion,
		&s.QuestionCount, &s.QuestionsPerSeries, &s.PublishedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Series{}, httpx.ErrNotFound
	}
	return s, err
}

func (s *Service) List(ctx context.Context, licenseCategoryID string) ([]Series, error) {
	if licenseCategoryID != "" && !httpx.IsUUID(licenseCategoryID) {
		return nil, errUnknownLicense
	}
	rows, err := s.db.Query(ctx, seriesSelect+`
		 WHERE ($1 = '' OR s.license_category_id::text = $1)
		 ORDER BY lc.sort_order, s.series_number`, licenseCategoryID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Series, error) { return scanSeries(row) })
}

// SeriesQuestion : question de la série, dans l'ordre.
type SeriesQuestion struct {
	Position     int    `json:"position"`
	QuestionID   string `json:"question_id"`
	QuestionType string `json:"question_type"`
	Status       string `json:"status"`
	TextFr       string `json:"text_fr"`
}

type Detail struct {
	Series
	Questions []SeriesQuestion `json:"questions"`
}

func (s *Service) Get(ctx context.Context, id string) (Detail, error) {
	series, err := scanSeries(s.db.QueryRow(ctx, seriesSelect+` WHERE s.id = $1`, id))
	if err != nil {
		return Detail{}, err
	}
	rows, err := s.db.Query(ctx,
		`SELECT sq.position, q.id, q.question_type, q.status, COALESCE(fr.text_content, '')
		   FROM series_questions sq
		   JOIN questions q ON q.id = sq.question_id
		   LEFT JOIN question_contents fr ON fr.question_id = q.id AND fr.language_code = 'fr'
		  WHERE sq.series_id = $1 ORDER BY sq.position`, id)
	if err != nil {
		return Detail{}, err
	}
	qs, err := pgx.CollectRows(rows, pgx.RowToStructByPos[SeriesQuestion])
	if err != nil {
		return Detail{}, err
	}
	return Detail{Series: series, Questions: qs}, nil
}

type CreateInput struct {
	LicenseCategoryID string `json:"license_category_id"`
	SeriesNumber      int    `json:"series_number"`
}

func (s *Service) Create(ctx context.Context, p httpx.Principal, in CreateInput) (Detail, error) {
	if !httpx.IsUUID(in.LicenseCategoryID) {
		return Detail{}, errUnknownLicense
	}
	var id string
	err := database.WithTx(ctx, s.db, func(tx pgx.Tx) error {
		var seriesCount int
		err := tx.QueryRow(ctx, `SELECT series_count FROM license_categories WHERE id = $1`, in.LicenseCategoryID).Scan(&seriesCount)
		if errors.Is(err, pgx.ErrNoRows) {
			return errUnknownLicense
		}
		if err != nil {
			return err
		}
		if in.SeriesNumber < 1 || in.SeriesNumber > seriesCount {
			return errSeriesNumber.With("max", seriesCount)
		}
		err = tx.QueryRow(ctx,
			`INSERT INTO exam_series (license_category_id, series_number) VALUES ($1, $2) RETURNING id`,
			in.LicenseCategoryID, in.SeriesNumber).Scan(&id)
		if database.IsUniqueViolation(err) {
			return errSeriesExists
		}
		if err != nil {
			return err
		}
		return audit.Log(ctx, tx, audit.Entry{ActorUserID: &p.UserID, Action: "series.created", EntityType: "exam_series", EntityID: &id})
	})
	if err != nil {
		return Detail{}, err
	}
	return s.Get(ctx, id)
}

// SetQuestions remplace la composition et l'ordre. Une série publiée modifiée repasse
// en brouillon ; les sessions déjà démarrées gardent leur instantané.
func (s *Service) SetQuestions(ctx context.Context, p httpx.Principal, id string, questionIDs []string) (Detail, error) {
	seen := map[string]bool{}
	for _, q := range questionIDs {
		if !httpx.IsUUID(q) || seen[q] {
			return Detail{}, errBadQuestions
		}
		seen[q] = true
	}
	err := database.WithTx(ctx, s.db, func(tx pgx.Tx) error {
		var perSeries int
		err := tx.QueryRow(ctx,
			`SELECT lc.questions_per_series FROM exam_series s JOIN license_categories lc ON lc.id = s.license_category_id
			  WHERE s.id = $1 FOR UPDATE OF s`, id).Scan(&perSeries)
		if errors.Is(err, pgx.ErrNoRows) {
			return httpx.ErrNotFound
		}
		if err != nil {
			return err
		}
		if len(questionIDs) > perSeries {
			return errTooMany.With("max", perSeries)
		}
		var valid int
		if err := tx.QueryRow(ctx,
			`SELECT count(*) FROM questions WHERE id = ANY($1::uuid[]) AND status <> 'archived'`, questionIDs).Scan(&valid); err != nil {
			return err
		}
		if valid != len(questionIDs) {
			return errBadQuestions
		}
		if _, err := tx.Exec(ctx, `DELETE FROM series_questions WHERE series_id = $1`, id); err != nil {
			return err
		}
		for i, q := range questionIDs {
			if _, err := tx.Exec(ctx,
				`INSERT INTO series_questions (series_id, question_id, position) VALUES ($1, $2, $3)`, id, q, i+1); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx,
			`UPDATE exam_series SET status = CASE WHEN status = 'archived' THEN status ELSE 'draft' END, updated_at = now()
			  WHERE id = $1`, id); err != nil {
			return err
		}
		return audit.Log(ctx, tx, audit.Entry{
			ActorUserID: &p.UserID, Action: "series.questions_set", EntityType: "exam_series", EntityID: &id,
			Metadata: map[string]any{"count": len(questionIDs)},
		})
	})
	if err != nil {
		return Detail{}, err
	}
	return s.Get(ctx, id)
}

// SetStatus applique le cycle de publication. Publier fige un instantané versionné.
func (s *Service) SetStatus(ctx context.Context, p httpx.Principal, id, status string) (Detail, error) {
	err := database.WithTx(ctx, s.db, func(tx pgx.Tx) error {
		var current string
		err := tx.QueryRow(ctx, `SELECT status FROM exam_series WHERE id = $1 FOR UPDATE`, id).Scan(&current)
		if errors.Is(err, pgx.ErrNoRows) {
			return httpx.ErrNotFound
		}
		if err != nil {
			return err
		}
		if err := workflow.Check(current, status, p); err != nil {
			return err
		}
		meta := map[string]any{"from": current, "to": status}
		switch status {
		case workflow.InReview:
			if err := s.checkPublishable(ctx, tx, id); err != nil {
				return err
			}
		case workflow.Published:
			version, err := s.snapshot(ctx, tx, p, id)
			if err != nil {
				return err
			}
			meta["version"] = version
		}
		if _, err := tx.Exec(ctx, `UPDATE exam_series SET status = $2 WHERE id = $1`, id, status); err != nil {
			return err
		}
		return audit.Log(ctx, tx, audit.Entry{
			ActorUserID: &p.UserID, Action: "series.status", EntityType: "exam_series", EntityID: &id, Metadata: meta,
		})
	})
	if err != nil {
		return Detail{}, err
	}
	return s.Get(ctx, id)
}

// checkPublishable : nombre exact de questions (CDC §9.3), toutes publiées et complètes.
func (s *Service) checkPublishable(ctx context.Context, tx pgx.Tx, id string) error {
	var count, perSeries, notPublished int
	if err := tx.QueryRow(ctx,
		`SELECT (SELECT count(*) FROM series_questions WHERE series_id = s.id),
		        lc.questions_per_series,
		        (SELECT count(*) FROM series_questions sq JOIN questions q ON q.id = sq.question_id
		          WHERE sq.series_id = s.id AND q.status <> 'published')
		   FROM exam_series s JOIN license_categories lc ON lc.id = s.license_category_id
		  WHERE s.id = $1`, id).Scan(&count, &perSeries, &notPublished); err != nil {
		return err
	}
	if count != perSeries {
		return errWrongCount.With("expected", perSeries).With("actual", count)
	}
	if notPublished > 0 {
		return errNotPublished.With("count", notPublished)
	}
	return nil
}

// Snapshot : contenu figé d'une série publiée, source de vérité des sessions d'examen.
type Snapshot struct {
	SeriesID           string             `json:"series_id"`
	LicenseCategoryID  string             `json:"license_category_id"`
	LicenseCode        string             `json:"license_code"`
	SeriesNumber       int                `json:"series_number"`
	Version            int                `json:"version"`
	QuestionsPerSeries int                `json:"questions_per_series"`
	MaxFaults          int                `json:"max_faults"`
	Questions          []SnapshotQuestion `json:"questions"`
}

type SnapshotQuestion struct {
	Position        int                          `json:"position"`
	QuestionID      string                       `json:"question_id"`
	QuestionVersion int                          `json:"question_version"`
	QuestionType    string                       `json:"question_type"`
	RelatedLessonID *string                      `json:"related_lesson_id"`
	Contents        map[string]questions.Content `json:"contents"`
	Options         []questions.Option           `json:"options"`
}

func (s *Service) snapshot(ctx context.Context, tx pgx.Tx, p httpx.Principal, id string) (int, error) {
	if err := s.checkPublishable(ctx, tx, id); err != nil {
		return 0, err
	}
	var snap Snapshot
	if err := tx.QueryRow(ctx,
		`SELECT s.id, s.license_category_id, lc.code, s.series_number, s.current_version + 1,
		        lc.questions_per_series, lc.max_faults
		   FROM exam_series s JOIN license_categories lc ON lc.id = s.license_category_id
		  WHERE s.id = $1`, id).Scan(&snap.SeriesID, &snap.LicenseCategoryID, &snap.LicenseCode, &snap.SeriesNumber,
		&snap.Version, &snap.QuestionsPerSeries, &snap.MaxFaults); err != nil {
		return 0, err
	}

	rows, err := tx.Query(ctx, `SELECT question_id, position FROM series_questions WHERE series_id = $1 ORDER BY position`, id)
	if err != nil {
		return 0, err
	}
	type ref struct {
		id       string
		position int
	}
	refs, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (ref, error) {
		var r ref
		return r, row.Scan(&r.id, &r.position)
	})
	if err != nil {
		return 0, err
	}
	for _, r := range refs {
		q, err := questions.Load(ctx, tx, r.id)
		if err != nil {
			return 0, err
		}
		if err := q.Complete(); err != nil {
			return 0, fmt.Errorf("question %s : %w", q.ID, err)
		}
		snap.Questions = append(snap.Questions, SnapshotQuestion{
			Position: r.position, QuestionID: q.ID, QuestionVersion: q.Version, QuestionType: q.QuestionType,
			RelatedLessonID: q.RelatedLessonID, Contents: q.Contents, Options: q.Options,
		})
	}

	payload, err := json.Marshal(snap)
	if err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO exam_series_snapshots (series_id, version, payload, published_by) VALUES ($1, $2, $3, $4)`,
		id, snap.Version, payload, p.UserID); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `UPDATE exam_series SET current_version = $2 WHERE id = $1`, id, snap.Version); err != nil {
		return 0, err
	}
	return snap.Version, nil
}
