package questions

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"samacode/backend/internal/audit"
	"samacode/backend/internal/database"
	"samacode/backend/internal/httpx"
	"samacode/backend/internal/workflow"
)

var (
	errBadReference = httpx.NewError(http.StatusBadRequest, "invalid_reference", "Un média ou une leçon référencé n'existe pas.")
	errOptionInUse  = httpx.NewError(http.StatusConflict, "option_in_use",
		"Cette proposition a déjà été choisie dans des examens : elle ne peut pas être supprimée. Modifiez-la plutôt.")
	errInPublishedSeries = httpx.NewError(http.StatusConflict, "question_in_published_series",
		"Cette question fait partie d'une série publiée : dépubliez d'abord la série.")
)

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service { return &Service{db: db} }

// Get charge une question complète (contenus, propositions et bonnes réponses).
func (s *Service) Get(ctx context.Context, id string) (Question, error) {
	return Load(ctx, s.db, id)
}

// Load charge une question complète avec q (pool ou transaction).
func Load(ctx context.Context, q database.Querier, id string) (Question, error) {
	var out Question
	err := q.QueryRow(ctx,
		`SELECT id, question_type, related_lesson_id, status, version, created_at, updated_at
		   FROM questions WHERE id = $1`, id,
	).Scan(&out.ID, &out.QuestionType, &out.RelatedLessonID, &out.Status, &out.Version, &out.CreatedAt, &out.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Question{}, httpx.ErrNotFound
	}
	if err != nil {
		return Question{}, err
	}

	out.Contents = map[string]Content{}
	rows, err := q.Query(ctx,
		`SELECT language_code, text_content, audio_media_id, image_media_id, explanation_text, explanation_audio_media_id
		   FROM question_contents WHERE question_id = $1`, id)
	if err != nil {
		return Question{}, err
	}
	for rows.Next() {
		var lang string
		var c Content
		if err := rows.Scan(&lang, &c.Text, &c.AudioMediaID, &c.ImageMediaID, &c.Explanation, &c.ExplanationAudioMediaID); err != nil {
			rows.Close()
			return Question{}, err
		}
		out.Contents[lang] = c
	}
	if err := rows.Err(); err != nil {
		return Question{}, err
	}

	rows, err = q.Query(ctx,
		`SELECT o.id, o.option_code, o.is_correct, oc.language_code, oc.text_content, oc.audio_media_id, oc.image_media_id
		   FROM answer_options o
		   LEFT JOIN option_contents oc ON oc.option_id = o.id
		  WHERE o.question_id = $1
		  ORDER BY o.sort_order, o.option_code`, id)
	if err != nil {
		return Question{}, err
	}
	defer rows.Close()
	index := map[string]int{}
	for rows.Next() {
		var o Option
		var lang *string
		var c OptionContent
		var text *string
		if err := rows.Scan(&o.ID, &o.Code, &o.IsCorrect, &lang, &text, &c.AudioMediaID, &c.ImageMediaID); err != nil {
			return Question{}, err
		}
		i, seen := index[o.ID]
		if !seen {
			o.Contents = map[string]OptionContent{}
			out.Options = append(out.Options, o)
			i = len(out.Options) - 1
			index[o.ID] = i
		}
		if lang != nil {
			if text != nil {
				c.Text = *text
			}
			out.Options[i].Contents[*lang] = c
		}
	}
	return out, rows.Err()
}

func (s *Service) Create(ctx context.Context, p httpx.Principal, in Input) (Question, error) {
	if err := in.validate(); err != nil {
		return Question{}, err
	}
	var id string
	err := database.WithTx(ctx, s.db, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			`INSERT INTO questions (question_type, related_lesson_id) VALUES ($1, $2) RETURNING id`,
			in.QuestionType, in.RelatedLessonID).Scan(&id); err != nil {
			return err
		}
		if err := writeContents(ctx, tx, id, in); err != nil {
			return err
		}
		return audit.Log(ctx, tx, audit.Entry{ActorUserID: &p.UserID, Action: "question.created", EntityType: "question", EntityID: &id})
	})
	if err != nil {
		return Question{}, mapErr(err)
	}
	return s.Get(ctx, id)
}

// Update remplace la question. Modifier une question publiée la repasse en brouillon
// avec une nouvelle version : les séries déjà publiées gardent leur instantané (D15).
func (s *Service) Update(ctx context.Context, p httpx.Principal, id string, in Input) (Question, error) {
	if err := in.validate(); err != nil {
		return Question{}, err
	}
	err := database.WithTx(ctx, s.db, func(tx pgx.Tx) error {
		var status string
		err := tx.QueryRow(ctx, `SELECT status FROM questions WHERE id = $1 FOR UPDATE`, id).Scan(&status)
		if errors.Is(err, pgx.ErrNoRows) {
			return httpx.ErrNotFound
		}
		if err != nil {
			return err
		}
		if status == workflow.Published {
			if err := ensureNotInPublishedSeries(ctx, tx, id); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx,
			`UPDATE questions SET question_type = $2, related_lesson_id = $3,
			        status = CASE WHEN status = 'published' THEN 'draft' ELSE status END,
			        version = CASE WHEN status = 'published' THEN version + 1 ELSE version END
			  WHERE id = $1`,
			id, in.QuestionType, in.RelatedLessonID); err != nil {
			return err
		}
		if err := writeContents(ctx, tx, id, in); err != nil {
			return err
		}
		return audit.Log(ctx, tx, audit.Entry{ActorUserID: &p.UserID, Action: "question.updated", EntityType: "question", EntityID: &id})
	})
	if err != nil {
		return Question{}, mapErr(err)
	}
	return s.Get(ctx, id)
}

// SetStatus applique le cycle de publication ; soumettre ou publier exige fr + wo.
func (s *Service) SetStatus(ctx context.Context, p httpx.Principal, id, status string) (Question, error) {
	err := database.WithTx(ctx, s.db, func(tx pgx.Tx) error {
		q, err := Load(ctx, tx, id)
		if err != nil {
			return err
		}
		if err := workflow.Check(q.Status, status, p); err != nil {
			return err
		}
		if status == workflow.InReview || status == workflow.Published {
			if err := q.Complete(); err != nil {
				return err
			}
		}
		if q.Status == workflow.Published {
			if err := ensureNotInPublishedSeries(ctx, tx, id); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx, `UPDATE questions SET status = $2 WHERE id = $1`, id, status); err != nil {
			return err
		}
		return audit.Log(ctx, tx, audit.Entry{
			ActorUserID: &p.UserID, Action: "question.status", EntityType: "question", EntityID: &id,
			Metadata: map[string]any{"from": q.Status, "to": status},
		})
	})
	if err != nil {
		return Question{}, err
	}
	return s.Get(ctx, id)
}

// Summary : ligne de la liste des questions.
type Summary struct {
	ID           string `json:"id"`
	QuestionType string `json:"question_type"`
	Status       string `json:"status"`
	Version      int    `json:"version"`
	TextFr       string `json:"text_fr"`
	HasWolof     bool   `json:"has_wolof"` // audio wolof de l'énoncé présent (D25)
	OptionCount  int    `json:"option_count"`
	SeriesCount  int    `json:"series_count"`
}

func (s *Service) List(ctx context.Context, status, search string, page httpx.Page) ([]Summary, int, error) {
	if status != "" && !workflow.Valid(status) {
		return nil, 0, workflow.ErrInvalidStatus
	}
	search = strings.TrimSpace(search)
	const where = `WHERE ($1 = '' OR q.status = $1)
	                 AND ($2 = '' OR fr.text_content ILIKE '%' || $2 || '%')`
	var total int
	if err := s.db.QueryRow(ctx,
		`SELECT count(*) FROM questions q
		   LEFT JOIN question_contents fr ON fr.question_id = q.id AND fr.language_code = 'fr' `+where,
		status, search).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Query(ctx,
		`SELECT q.id, q.question_type, q.status, q.version, COALESCE(fr.text_content, ''),
		        EXISTS (SELECT 1 FROM question_contents wo WHERE wo.question_id = q.id AND wo.language_code = 'wo' AND wo.audio_media_id IS NOT NULL),
		        (SELECT count(*) FROM answer_options o WHERE o.question_id = q.id),
		        (SELECT count(*) FROM series_questions sq WHERE sq.question_id = q.id)
		   FROM questions q
		   LEFT JOIN question_contents fr ON fr.question_id = q.id AND fr.language_code = 'fr' `+where+`
		  ORDER BY q.updated_at DESC LIMIT $3 OFFSET $4`,
		status, search, page.Size, page.Offset())
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Summary])
	return items, total, err
}

// writeContents remplace contenus et propositions. Les propositions sont mises à jour
// par lettre (et non recréées) : les réponses d'examens passés y restent liées.
func writeContents(ctx context.Context, tx pgx.Tx, id string, in Input) error {
	if _, err := tx.Exec(ctx, `DELETE FROM question_contents WHERE question_id = $1`, id); err != nil {
		return err
	}
	for lang, c := range in.Contents {
		if strings.TrimSpace(c.Text) == "" && c.AudioMediaID == nil && c.ExplanationAudioMediaID == nil {
			continue
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO question_contents (question_id, language_code, text_content, audio_media_id, image_media_id,
			                                explanation_text, explanation_audio_media_id)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			id, lang, c.Text, c.AudioMediaID, c.ImageMediaID, nilIfBlank(c.Explanation), c.ExplanationAudioMediaID); err != nil {
			return err
		}
	}

	codes := make([]string, 0, len(in.Options))
	for i, o := range in.Options {
		codes = append(codes, o.Code)
		var optionID string
		if err := tx.QueryRow(ctx,
			`INSERT INTO answer_options (question_id, option_code, is_correct, sort_order) VALUES ($1, $2, $3, $4)
			 ON CONFLICT (question_id, option_code) DO UPDATE SET is_correct = EXCLUDED.is_correct, sort_order = EXCLUDED.sort_order
			 RETURNING id`,
			id, o.Code, o.IsCorrect, i).Scan(&optionID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM option_contents WHERE option_id = $1`, optionID); err != nil {
			return err
		}
		for lang, c := range o.Contents {
			if strings.TrimSpace(c.Text) == "" && c.ImageMediaID == nil && c.AudioMediaID == nil {
				continue
			}
			if _, err := tx.Exec(ctx,
				`INSERT INTO option_contents (option_id, language_code, text_content, audio_media_id, image_media_id)
				 VALUES ($1, $2, $3, $4, $5)`,
				optionID, lang, c.Text, c.AudioMediaID, c.ImageMediaID); err != nil {
				return err
			}
		}
	}
	_, err := tx.Exec(ctx, `DELETE FROM answer_options WHERE question_id = $1 AND NOT (option_code = ANY($2))`, id, codes)
	return err
}

func ensureNotInPublishedSeries(ctx context.Context, tx pgx.Tx, id string) error {
	var inPublished bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM series_questions sq JOIN exam_series s ON s.id = sq.series_id
		                 WHERE sq.question_id = $1 AND s.status = 'published')`, id).Scan(&inPublished); err != nil {
		return err
	}
	if inPublished {
		return errInPublishedSeries
	}
	return nil
}

func mapErr(err error) error {
	if database.IsForeignKeyViolation(err) {
		if strings.Contains(err.Error(), "selected_options") {
			return errOptionInUse
		}
		return errBadReference
	}
	return err
}

func nilIfBlank(s *string) *string {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil
	}
	t := strings.TrimSpace(*s)
	return &t
}

var (
	errQuestionInSeries = httpx.NewError(http.StatusConflict, "question_in_series", "Cette question fait partie d'une série : retirez-la d'abord de la série.")
	errQuestionAnswered = httpx.NewError(http.StatusConflict, "question_answered", "Des candidats ont déjà répondu à cette question : archivez-la plutôt, pour garder leur historique.")
)

// Delete supprime définitivement une question qui n'est dans aucune série et à
// laquelle aucun candidat n'a répondu (sinon l'historique des examens serait cassé).
func (s *Service) Delete(ctx context.Context, p httpx.Principal, id string) error {
	return database.WithTx(ctx, s.db, func(tx pgx.Tx) error {
		var status, label string
		err := tx.QueryRow(ctx,
			`SELECT q.status, COALESCE(c.text_content, '') FROM questions q
			   LEFT JOIN question_contents c ON c.question_id = q.id AND c.language_code = 'fr'
			  WHERE q.id = $1 FOR UPDATE OF q`, id).Scan(&status, &label)
		if errors.Is(err, pgx.ErrNoRows) {
			return httpx.ErrNotFound
		}
		if err != nil {
			return err
		}
		if err := workflow.CheckDelete(status, p); err != nil {
			return err
		}
		var inSeries, answered bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM series_questions WHERE question_id = $1),
			        EXISTS (SELECT 1 FROM candidate_answers WHERE question_id = $1)`, id).Scan(&inSeries, &answered); err != nil {
			return err
		}
		switch {
		case inSeries:
			return errQuestionInSeries
		case answered:
			return errQuestionAnswered
		}
		if _, err := tx.Exec(ctx, `DELETE FROM questions WHERE id = $1`, id); err != nil {
			if database.IsForeignKeyViolation(err) {
				return errQuestionAnswered
			}
			return err
		}
		return audit.Log(ctx, tx, audit.Entry{
			ActorUserID: &p.UserID, Action: "question.deleted", EntityType: "question", EntityID: &id,
			Metadata: map[string]any{"status": status, "text_fr": label},
		})
	})
}
