package examengine

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"samacode/backend/internal/database"
	"samacode/backend/internal/examseries"
	"samacode/backend/internal/httpx"
	"samacode/backend/internal/questions"
	"samacode/backend/internal/scoring"
)

// Délai de tolérance après expires_at pour une réponse partie juste avant la fin
// (latence réseau). Au-delà, la session est soumise automatiquement.
const answerGrace = 5 * time.Second

var (
	errSeriesUnavailable = httpx.NewError(http.StatusNotFound, "series_unavailable", "Cette série n'est pas disponible.")
	errLicenseMismatch   = httpx.NewError(http.StatusBadRequest, "series_license_mismatch", "Cette série n'appartient pas à ce permis.")
	errInProgress        = httpx.NewError(http.StatusConflict, "session_in_progress", "Un examen est déjà en cours. Reprenez-le ou abandonnez-le.")
	errNotInProgress     = httpx.NewError(http.StatusConflict, "session_submitted", "Cet examen est terminé : les réponses ne peuvent plus être modifiées.")
	errExpired           = httpx.NewError(http.StatusConflict, "session_expired", "Le temps est écoulé : l'examen a été soumis automatiquement.")
	errNotSubmitted      = httpx.NewError(http.StatusConflict, "session_not_submitted", "La correction est disponible après la fin de l'examen.")
	errUnknownQuestion   = httpx.NewError(http.StatusNotFound, "question_not_in_session", "Cette question ne fait pas partie de l'examen.")
	errBadOptions        = httpx.Validation("option_ids", "Propositions invalides pour cette question.")
	errSingleChoice      = httpx.Validation("option_ids", "Une seule réponse possible pour cette question.")
)

type Service struct {
	db  *pgxpool.Pool
	now func() time.Time

	// Les instantanés sont immuables : on les garde en mémoire une fois lus.
	snapshots sync.Map // snapshot id → *examseries.Snapshot
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db, now: time.Now}
}

// sessionRow : état d'une session en base.
type sessionRow struct {
	id, userID, seriesID, snapshotID, status string
	submitReason                             *string
	durationMinutes                          int
	startedAt, expiresAt                     time.Time
	submittedAt                              *time.Time
}

const sessionColumns = `id, user_id, series_id, snapshot_id, status, submit_reason, duration_minutes, started_at, expires_at, submitted_at`

func scanSession(row pgx.Row) (sessionRow, error) {
	var s sessionRow
	err := row.Scan(&s.id, &s.userID, &s.seriesID, &s.snapshotID, &s.status, &s.submitReason, &s.durationMinutes,
		&s.startedAt, &s.expiresAt, &s.submittedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return sessionRow{}, httpx.ErrNotFound
	}
	return s, err
}

// lockSession charge la session du candidat (sinon 404 : on ne révèle pas l'existence).
func lockSession(ctx context.Context, tx pgx.Tx, userID, id string) (sessionRow, error) {
	return scanSession(tx.QueryRow(ctx,
		`SELECT `+sessionColumns+` FROM exam_sessions WHERE id = $1 AND user_id = $2 FOR UPDATE`, id, userID))
}

func (s *Service) snapshot(ctx context.Context, q database.Querier, id string) (*examseries.Snapshot, error) {
	if v, ok := s.snapshots.Load(id); ok {
		return v.(*examseries.Snapshot), nil
	}
	var payload []byte
	if err := q.QueryRow(ctx, `SELECT payload FROM exam_series_snapshots WHERE id = $1`, id).Scan(&payload); err != nil {
		return nil, err
	}
	var snap examseries.Snapshot
	if err := json.Unmarshal(payload, &snap); err != nil {
		return nil, err
	}
	s.snapshots.Store(id, &snap)
	return &snap, nil
}

// ---------------------------------------------------------------------------
// Séries proposées au candidat
// ---------------------------------------------------------------------------

func (s *Service) ListSeries(ctx context.Context, userID, licenseID string) ([]SeriesSummary, error) {
	if !httpx.IsUUID(licenseID) {
		return nil, httpx.ErrNotFound
	}
	if err := s.finalizeExpired(ctx, userID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx,
		`SELECT s.id, s.series_number, lc.questions_per_series, lc.exam_duration_minutes, lc.max_faults,
		        (SELECT count(*) FROM exam_sessions es WHERE es.series_id = s.id AND es.user_id = $2 AND es.status = 'submitted'),
		        (SELECT min(r.fault_count) FROM exam_results r JOIN exam_sessions es ON es.id = r.session_id
		          WHERE es.series_id = s.id AND es.user_id = $2),
		        (SELECT r.passed FROM exam_results r JOIN exam_sessions es ON es.id = r.session_id
		          WHERE es.series_id = s.id AND es.user_id = $2 ORDER BY r.evaluated_at DESC LIMIT 1)
		   FROM exam_series s JOIN license_categories lc ON lc.id = s.license_category_id
		  WHERE s.license_category_id = $1 AND s.status = 'published' AND s.current_version > 0 AND lc.is_active
		  ORDER BY s.series_number`, licenseID, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[SeriesSummary])
}

// ---------------------------------------------------------------------------
// Démarrage et consultation
// ---------------------------------------------------------------------------

type StartInput struct {
	LicenseCategoryID string `json:"license_category_id"`
	SeriesID          string `json:"series_id"`
	// Abandonner l'examen en cours (soumis en l'état) pour en démarrer un nouveau.
	AbandonCurrent bool `json:"abandon_current"`
}

func (s *Service) Start(ctx context.Context, userID string, in StartInput) (Session, error) {
	if !httpx.IsUUID(in.LicenseCategoryID) || !httpx.IsUUID(in.SeriesID) {
		return Session{}, errSeriesUnavailable
	}
	now := s.now()
	var id string
	err := database.WithTx(ctx, s.db, func(tx pgx.Tx) error {
		current, err := scanSession(tx.QueryRow(ctx,
			`SELECT `+sessionColumns+` FROM exam_sessions WHERE user_id = $1 AND status = 'in_progress' FOR UPDATE`, userID))
		switch {
		case errors.Is(err, httpx.ErrNotFound):
		case err != nil:
			return err
		case !now.Before(current.expiresAt):
			if err := s.submit(ctx, tx, current, "timeout", now); err != nil {
				return err
			}
		case !in.AbandonCurrent:
			return errInProgress.With("session_id", current.id)
		default:
			if err := s.submit(ctx, tx, current, "abandoned", now); err != nil {
				return err
			}
		}

		var licenseID, snapshotID string
		var duration int
		err = tx.QueryRow(ctx,
			`SELECT s.license_category_id, sn.id, lc.exam_duration_minutes
			   FROM exam_series s
			   JOIN license_categories lc ON lc.id = s.license_category_id
			   JOIN exam_series_snapshots sn ON sn.series_id = s.id AND sn.version = s.current_version
			  WHERE s.id = $1 AND s.status = 'published' AND lc.is_active`, in.SeriesID,
		).Scan(&licenseID, &snapshotID, &duration)
		if errors.Is(err, pgx.ErrNoRows) {
			return errSeriesUnavailable
		}
		if err != nil {
			return err
		}
		if licenseID != in.LicenseCategoryID {
			return errLicenseMismatch
		}
		return tx.QueryRow(ctx,
			`INSERT INTO exam_sessions (user_id, series_id, snapshot_id, duration_minutes, started_at, expires_at)
			 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
			userID, in.SeriesID, snapshotID, duration, now, now.Add(time.Duration(duration)*time.Minute)).Scan(&id)
	})
	if err != nil {
		return Session{}, err
	}
	return s.Get(ctx, userID, id)
}

// Current renvoie l'examen en cours du candidat (pour proposer de le reprendre).
func (s *Service) Current(ctx context.Context, userID string) (Session, error) {
	if err := s.finalizeExpired(ctx, userID); err != nil {
		return Session{}, err
	}
	var id string
	err := s.db.QueryRow(ctx, `SELECT id FROM exam_sessions WHERE user_id = $1 AND status = 'in_progress'`, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, httpx.ErrNotFound
	}
	if err != nil {
		return Session{}, err
	}
	return s.Get(ctx, userID, id)
}

// Get renvoie la session ; une session expirée est d'abord soumise automatiquement.
func (s *Service) Get(ctx context.Context, userID, id string) (Session, error) {
	if !httpx.IsUUID(id) {
		return Session{}, httpx.ErrNotFound
	}
	now := s.now()
	var row sessionRow
	err := database.WithTx(ctx, s.db, func(tx pgx.Tx) error {
		var err error
		if row, err = lockSession(ctx, tx, userID, id); err != nil {
			return err
		}
		if row.status == "in_progress" && !now.Before(row.expiresAt) {
			if err := s.submit(ctx, tx, row, "timeout", now); err != nil {
				return err
			}
			row, err = lockSession(ctx, tx, userID, id)
		}
		return err
	})
	if err != nil {
		return Session{}, err
	}
	return s.view(ctx, row, now)
}

func (s *Service) view(ctx context.Context, row sessionRow, now time.Time) (Session, error) {
	snap, err := s.snapshot(ctx, s.db, row.snapshotID)
	if err != nil {
		return Session{}, err
	}
	answers, err := loadAnswers(ctx, s.db, row.id)
	if err != nil {
		return Session{}, err
	}
	out := Session{
		ID: row.id, SeriesID: row.seriesID, LicenseCode: snap.LicenseCode, SeriesNumber: snap.SeriesNumber,
		SeriesVersion: snap.Version, Status: row.status, SubmitReason: row.submitReason,
		DurationMinutes: row.durationMinutes, MaxFaults: snap.MaxFaults,
		StartedAt: row.startedAt, ExpiresAt: row.expiresAt, SubmittedAt: row.submittedAt, ServerTime: now,
		Answers: answers,
	}
	for _, q := range snap.Questions {
		out.Questions = append(out.Questions, candidateQuestion(q))
	}
	if row.status == "submitted" {
		r, err := loadResult(ctx, s.db, row.id)
		if err != nil {
			return Session{}, err
		}
		out.Result = &r
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Réponses
// ---------------------------------------------------------------------------

// Answer enregistre ou remplace la réponse à une question ; une liste vide l'efface.
func (s *Service) Answer(ctx context.Context, userID, id, questionID string, optionIDs []string) error {
	if !httpx.IsUUID(id) || !httpx.IsUUID(questionID) {
		return httpx.ErrNotFound
	}
	slices.Sort(optionIDs)
	optionIDs = slices.Compact(optionIDs)
	now := s.now()
	var expired bool
	err := database.WithTx(ctx, s.db, func(tx pgx.Tx) error {
		row, err := lockSession(ctx, tx, userID, id)
		if err != nil {
			return err
		}
		if row.status != "in_progress" {
			return errNotInProgress
		}
		if now.After(row.expiresAt.Add(answerGrace)) {
			expired = true
			return s.submit(ctx, tx, row, "timeout", now)
		}
		snap, err := s.snapshot(ctx, tx, row.snapshotID)
		if err != nil {
			return err
		}
		i := slices.IndexFunc(snap.Questions, func(q examseries.SnapshotQuestion) bool { return q.QuestionID == questionID })
		if i < 0 {
			return errUnknownQuestion
		}
		q := snap.Questions[i]
		for _, o := range optionIDs {
			if !slices.ContainsFunc(q.Options, func(opt questions.Option) bool { return opt.ID == o }) {
				return errBadOptions
			}
		}
		if q.QuestionType == "single" && len(optionIDs) > 1 {
			return errSingleChoice
		}

		if len(optionIDs) == 0 {
			_, err := tx.Exec(ctx, `DELETE FROM candidate_answers WHERE session_id = $1 AND question_id = $2`, id, questionID)
			return err
		}
		var answerID string
		if err := tx.QueryRow(ctx,
			`INSERT INTO candidate_answers (session_id, question_id, answered_at) VALUES ($1, $2, $3)
			 ON CONFLICT (session_id, question_id) DO UPDATE SET answered_at = EXCLUDED.answered_at
			 RETURNING id`, id, questionID, now).Scan(&answerID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM selected_options WHERE candidate_answer_id = $1`, answerID); err != nil {
			return err
		}
		for _, o := range optionIDs {
			if _, err := tx.Exec(ctx,
				`INSERT INTO selected_options (candidate_answer_id, question_id, option_id) VALUES ($1, $2, $3)`,
				answerID, questionID, o); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if expired {
		return errExpired
	}
	return nil
}

// ---------------------------------------------------------------------------
// Soumission, résultat, correction, historique
// ---------------------------------------------------------------------------

// Submit est idempotent : une session déjà soumise renvoie son résultat inchangé.
func (s *Service) Submit(ctx context.Context, userID, id string) (Session, error) {
	if !httpx.IsUUID(id) {
		return Session{}, httpx.ErrNotFound
	}
	now := s.now()
	err := database.WithTx(ctx, s.db, func(tx pgx.Tx) error {
		row, err := lockSession(ctx, tx, userID, id)
		if err != nil {
			return err
		}
		if row.status != "in_progress" {
			return nil
		}
		reason := "manual"
		if !now.Before(row.expiresAt.Add(answerGrace)) {
			reason = "timeout"
		}
		return s.submit(ctx, tx, row, reason, now)
	})
	if err != nil {
		return Session{}, err
	}
	return s.Get(ctx, userID, id)
}

// submit évalue et clôt une session verrouillée (FOR UPDATE) encore en cours.
func (s *Service) submit(ctx context.Context, tx pgx.Tx, row sessionRow, reason string, now time.Time) error {
	snap, err := s.snapshot(ctx, tx, row.snapshotID)
	if err != nil {
		return err
	}
	answers, err := loadAnswers(ctx, tx, row.id)
	if err != nil {
		return err
	}
	r := scoring.Evaluate(scoringQuestions(*snap), answers, snap.MaxFaults)
	if _, err := tx.Exec(ctx,
		`INSERT INTO exam_results (session_id, correct_count, fault_count, unanswered_count, max_faults, passed, evaluated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		row.id, r.CorrectCount, r.FaultCount, r.UnansweredCount, r.MaxFaults, r.Passed, now); err != nil {
		return err
	}
	_, err = tx.Exec(ctx,
		`UPDATE exam_sessions SET status = 'submitted', submit_reason = $2, submitted_at = $3 WHERE id = $1`,
		row.id, reason, now)
	return err
}

// finalizeExpired soumet les examens du candidat dont le temps est écoulé.
func (s *Service) finalizeExpired(ctx context.Context, userID string) error {
	now := s.now()
	return database.WithTx(ctx, s.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT `+sessionColumns+` FROM exam_sessions
			  WHERE user_id = $1 AND status = 'in_progress' AND expires_at <= $2 FOR UPDATE`, userID, now)
		if err != nil {
			return err
		}
		expired, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (sessionRow, error) { return scanSession(r) })
		if err != nil {
			return err
		}
		for _, row := range expired {
			if err := s.submit(ctx, tx, row, "timeout", now); err != nil {
				return err
			}
		}
		return nil
	})
}

// Review : corrections détaillées, uniquement après soumission (CLAUDE.md, règle 3).
func (s *Service) Review(ctx context.Context, userID, id string) (Review, error) {
	session, err := s.Get(ctx, userID, id)
	if err != nil {
		return Review{}, err
	}
	if session.Status != "submitted" || session.Result == nil {
		return Review{}, errNotSubmitted
	}
	row, err := scanSession(s.db.QueryRow(ctx, `SELECT `+sessionColumns+` FROM exam_sessions WHERE id = $1`, id))
	if err != nil {
		return Review{}, err
	}
	snap, err := s.snapshot(ctx, s.db, row.snapshotID)
	if err != nil {
		return Review{}, err
	}
	evaluated := scoring.Evaluate(scoringQuestions(*snap), session.Answers, snap.MaxFaults)
	out := Review{SessionID: id, Result: *session.Result}
	for _, q := range snap.Questions {
		rq := ReviewQuestion{
			CandidateQuestion: candidateQuestion(q),
			SelectedOptionIDs: session.Answers[q.QuestionID],
			Outcome:           evaluated.Outcomes[q.QuestionID],
			Explanations:      explanations(q.Contents),
			RelatedLessonID:   q.RelatedLessonID,
			CorrectOptionIDs:  []string{},
		}
		if rq.SelectedOptionIDs == nil {
			rq.SelectedOptionIDs = []string{}
		}
		for _, o := range q.Options {
			if o.IsCorrect {
				rq.CorrectOptionIDs = append(rq.CorrectOptionIDs, o.ID)
			}
		}
		out.Questions = append(out.Questions, rq)
	}
	return out, nil
}

func (s *Service) History(ctx context.Context, userID string, page httpx.Page) ([]HistoryItem, error) {
	if err := s.finalizeExpired(ctx, userID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx,
		`SELECT es.id, lc.code, s.series_number, es.status, es.started_at, es.submitted_at,
		        r.correct_count, r.fault_count, r.unanswered_count, r.max_faults, r.passed
		   FROM exam_sessions es
		   JOIN exam_series s ON s.id = es.series_id
		   JOIN license_categories lc ON lc.id = s.license_category_id
		   LEFT JOIN exam_results r ON r.session_id = es.id
		  WHERE es.user_id = $1
		  ORDER BY es.started_at DESC LIMIT $2 OFFSET $3`, userID, page.Size, page.Offset())
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (HistoryItem, error) {
		var h HistoryItem
		var correct, faults, unanswered, maxFaults *int
		var passed *bool
		if err := row.Scan(&h.ID, &h.LicenseCode, &h.SeriesNumber, &h.Status, &h.StartedAt, &h.SubmittedAt,
			&correct, &faults, &unanswered, &maxFaults, &passed); err != nil {
			return h, err
		}
		if passed != nil {
			h.Result = &scoring.Result{CorrectCount: *correct, FaultCount: *faults, UnansweredCount: *unanswered,
				MaxFaults: *maxFaults, Passed: *passed}
		}
		return h, nil
	})
}

func loadAnswers(ctx context.Context, q database.Querier, sessionID string) (map[string][]string, error) {
	rows, err := q.Query(ctx,
		`SELECT ca.question_id, so.option_id FROM candidate_answers ca
		   JOIN selected_options so ON so.candidate_answer_id = ca.id
		  WHERE ca.session_id = $1 ORDER BY ca.question_id, so.option_id`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]string{}
	for rows.Next() {
		var qid, oid string
		if err := rows.Scan(&qid, &oid); err != nil {
			return nil, err
		}
		out[qid] = append(out[qid], oid)
	}
	return out, rows.Err()
}

func loadResult(ctx context.Context, q database.Querier, sessionID string) (scoring.Result, error) {
	var r scoring.Result
	err := q.QueryRow(ctx,
		`SELECT correct_count, fault_count, unanswered_count, max_faults, passed FROM exam_results WHERE session_id = $1`,
		sessionID).Scan(&r.CorrectCount, &r.FaultCount, &r.UnansweredCount, &r.MaxFaults, &r.Passed)
	return r, err
}
