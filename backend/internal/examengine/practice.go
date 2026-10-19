package examengine

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"samacode/backend/internal/examseries"
	"samacode/backend/internal/httpx"
	"samacode/backend/internal/questions"
	"samacode/backend/internal/scoring"
)

// Entraînement (CDC §4.3, D22) : mêmes questions que les séries publiées, correction
// et explication après chaque réponse validée, sans chronomètre. Les bonnes réponses
// ne sont envoyées qu'à la correction de la question (CLAUDE.md, règle 3).

var errNoAnswer = httpx.Validation("option_ids", "Choisissez au moins une réponse.")

// PracticeSeries : série publiée et progression du candidat à l'entraînement.
type PracticeSeries struct {
	ID            string `json:"id"`
	SeriesNumber  int    `json:"series_number"`
	QuestionCount int    `json:"question_count"`
	Practiced     int    `json:"practiced"` // questions déjà tentées au moins une fois
	Mastered      int    `json:"mastered"`  // questions dont la dernière réponse est juste
}

type PracticeLicense struct {
	ID       string           `json:"id"`
	Code     string           `json:"code"`
	Name     string           `json:"name"`
	Series   []PracticeSeries `json:"series"`
	Mistakes int              `json:"mistakes"` // questions dont la dernière réponse est fausse
}

// PracticeItem : question à travailler, avec la série dont elle vient.
type PracticeItem struct {
	SeriesID string            `json:"series_id"`
	Question CandidateQuestion `json:"question"`
	// Dernier résultat du candidat sur cette question dans cette série (null = pas encore
	// répondue) : permet de reprendre l'entraînement là où il s'est arrêté.
	LastOutcome *scoring.Outcome `json:"last_outcome"`
}

type PracticeCheck struct {
	Outcome          scoring.Outcome    `json:"outcome"`
	CorrectOptionIDs []string           `json:"correct_option_ids"`
	Explanations     map[string]Content `json:"explanations"`
}

// publishedSnapshots : instantanés courants des séries publiées d'un permis (ou de tous).
func (s *Service) publishedSnapshots(ctx context.Context, licenseID string) ([]*examseries.Snapshot, error) {
	rows, err := s.db.Query(ctx,
		`SELECT sn.id FROM exam_series s
		   JOIN license_categories lc ON lc.id = s.license_category_id
		   JOIN exam_series_snapshots sn ON sn.series_id = s.id AND sn.version = s.current_version
		  WHERE s.status = 'published' AND lc.is_active AND ($1 = '' OR s.license_category_id::text = $1)
		  ORDER BY lc.sort_order, s.series_number`, licenseID)
	if err != nil {
		return nil, err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}
	out := make([]*examseries.Snapshot, 0, len(ids))
	for _, id := range ids {
		snap, err := s.snapshot(ctx, s.db, id)
		if err != nil {
			return nil, err
		}
		out = append(out, snap)
	}
	return out, nil
}

// practiceKey : une question dans une série (la progression est comptée par série).
type practiceKey struct{ seriesID, questionID string }

// lastPracticeResults : dernière réponse d'entraînement du candidat, par question et par série.
func (s *Service) lastPracticeResults(ctx context.Context, userID string) (map[practiceKey]bool, error) {
	rows, err := s.db.Query(ctx,
		`SELECT DISTINCT ON (series_id, question_id) series_id, question_id, is_correct FROM practice_answers
		  WHERE user_id = $1 AND series_id IS NOT NULL
		  ORDER BY series_id, question_id, answered_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[practiceKey]bool{}
	for rows.Next() {
		var k practiceKey
		var ok bool
		if err := rows.Scan(&k.seriesID, &k.questionID, &ok); err != nil {
			return nil, err
		}
		out[k] = ok
	}
	return out, rows.Err()
}

// PracticeSummary : permis, séries et erreurs à revoir du candidat.
func (s *Service) PracticeSummary(ctx context.Context, userID string) ([]PracticeLicense, error) {
	snaps, err := s.publishedSnapshots(ctx, "")
	if err != nil {
		return nil, err
	}
	last, err := s.lastPracticeResults(ctx, userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, `SELECT id, code, name FROM license_categories WHERE is_active ORDER BY sort_order, code`)
	if err != nil {
		return nil, err
	}
	licenses, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (PracticeLicense, error) {
		var l PracticeLicense
		return l, r.Scan(&l.ID, &l.Code, &l.Name)
	})
	if err != nil {
		return nil, err
	}
	for i := range licenses {
		l := &licenses[i]
		l.Series = []PracticeSeries{}
		wrong := map[string]bool{}
		for _, snap := range snaps {
			if snap.LicenseCategoryID != l.ID {
				continue
			}
			ps := PracticeSeries{ID: snap.SeriesID, SeriesNumber: snap.SeriesNumber, QuestionCount: len(snap.Questions)}
			for _, q := range snap.Questions {
				ok, tried := last[practiceKey{snap.SeriesID, q.QuestionID}]
				if !tried {
					continue
				}
				ps.Practiced++
				if ok {
					ps.Mastered++
				} else {
					wrong[q.QuestionID] = true
				}
			}
			l.Series = append(l.Series, ps)
		}
		l.Mistakes = len(wrong)
	}
	return licenses, nil
}

// PracticeQuestions : questions d'une série publiée, dans l'ordre, sans les bonnes
// réponses, avec le dernier résultat du candidat sur chacune.
func (s *Service) PracticeQuestions(ctx context.Context, userID, seriesID string) ([]PracticeItem, error) {
	snap, err := s.currentSnapshot(ctx, seriesID)
	if err != nil {
		return nil, err
	}
	last, err := s.lastPracticeResults(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]PracticeItem, 0, len(snap.Questions))
	for _, q := range snap.Questions {
		item := PracticeItem{SeriesID: seriesID, Question: candidateQuestion(q)}
		if ok, tried := last[practiceKey{seriesID, q.QuestionID}]; tried {
			o := scoring.Incorrect
			if ok {
				o = scoring.Correct
			}
			item.LastOutcome = &o
		}
		out = append(out, item)
	}
	return out, nil
}

// PracticeMistakes : questions dont la dernière réponse d'entraînement, dans leur série, est fausse.
func (s *Service) PracticeMistakes(ctx context.Context, userID, licenseID string) ([]PracticeItem, error) {
	if !httpx.IsUUID(licenseID) {
		return nil, httpx.ErrNotFound
	}
	snaps, err := s.publishedSnapshots(ctx, licenseID)
	if err != nil {
		return nil, err
	}
	last, err := s.lastPracticeResults(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := []PracticeItem{}
	seen := map[string]bool{}
	for _, snap := range snaps {
		for _, q := range snap.Questions {
			// Une question ratée dans deux séries n'apparaît qu'une fois.
			if ok, tried := last[practiceKey{snap.SeriesID, q.QuestionID}]; tried && !ok && !seen[q.QuestionID] {
				seen[q.QuestionID] = true
				out = append(out, PracticeItem{SeriesID: snap.SeriesID, Question: candidateQuestion(q)})
			}
		}
	}
	return out, nil
}

// PracticeCheck corrige une réponse d'entraînement et l'enregistre (thèmes à revoir).
func (s *Service) PracticeCheck(ctx context.Context, userID, seriesID, questionID string, optionIDs []string) (PracticeCheck, error) {
	if !httpx.IsUUID(questionID) {
		return PracticeCheck{}, httpx.ErrNotFound
	}
	if len(optionIDs) == 0 {
		return PracticeCheck{}, errNoAnswer
	}
	snap, err := s.currentSnapshot(ctx, seriesID)
	if err != nil {
		return PracticeCheck{}, err
	}
	i := slices.IndexFunc(snap.Questions, func(q examseries.SnapshotQuestion) bool { return q.QuestionID == questionID })
	if i < 0 {
		return PracticeCheck{}, errUnknownQuestion
	}
	q := snap.Questions[i]
	for _, o := range optionIDs {
		if !slices.ContainsFunc(q.Options, func(opt questions.Option) bool { return opt.ID == o }) {
			return PracticeCheck{}, errBadOptions
		}
	}
	if q.QuestionType == "single" && len(optionIDs) > 1 {
		return PracticeCheck{}, errSingleChoice
	}

	sq := scoringQuestions(examseries.Snapshot{Questions: []examseries.SnapshotQuestion{q}})
	r := scoring.Evaluate(sq, map[string][]string{questionID: optionIDs}, 0)
	outcome := r.Outcomes[questionID]
	if _, err := s.db.Exec(ctx,
		`INSERT INTO practice_answers (user_id, series_id, question_id, is_correct, answered_at) VALUES ($1, $2, $3, $4, $5)`,
		userID, seriesID, questionID, outcome == scoring.Correct, s.now()); err != nil {
		return PracticeCheck{}, err
	}
	correct := sq[0].CorrectOptionIDs
	if correct == nil {
		correct = []string{}
	}
	return PracticeCheck{Outcome: outcome, CorrectOptionIDs: correct, Explanations: explanations(q.Contents)}, nil
}

func (s *Service) currentSnapshot(ctx context.Context, seriesID string) (*examseries.Snapshot, error) {
	if !httpx.IsUUID(seriesID) {
		return nil, errSeriesUnavailable
	}
	var snapID string
	err := s.db.QueryRow(ctx,
		`SELECT sn.id FROM exam_series s
		   JOIN license_categories lc ON lc.id = s.license_category_id
		   JOIN exam_series_snapshots sn ON sn.series_id = s.id AND sn.version = s.current_version
		  WHERE s.id = $1 AND s.status = 'published' AND lc.is_active`, seriesID).Scan(&snapID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errSeriesUnavailable
	}
	if err != nil {
		return nil, err
	}
	return s.snapshot(ctx, s.db, snapID)
}

// ---------------------------------------------------------------------------
// Routes
// ---------------------------------------------------------------------------

func (h *Handler) practiceRoutes(r chi.Router) {
	r.Get("/practice", httpx.Handle(h.practiceSummary))
	r.Get("/practice/series/{id}/questions", httpx.Handle(h.practiceQuestions))
	r.Get("/practice/license-categories/{id}/mistakes", httpx.Handle(h.practiceMistakes))
	r.Post("/practice/series/{id}/questions/{questionId}/check", httpx.Handle(h.practiceCheck))
}

func (h *Handler) practiceSummary(w http.ResponseWriter, r *http.Request) error {
	items, err := h.svc.PracticeSummary(r.Context(), userID(r))
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
	return nil
}

func (h *Handler) practiceQuestions(w http.ResponseWriter, r *http.Request) error {
	items, err := h.svc.PracticeQuestions(r.Context(), userID(r), chi.URLParam(r, "id"))
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
	return nil
}

func (h *Handler) practiceMistakes(w http.ResponseWriter, r *http.Request) error {
	items, err := h.svc.PracticeMistakes(r.Context(), userID(r), chi.URLParam(r, "id"))
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
	return nil
}

func (h *Handler) practiceCheck(w http.ResponseWriter, r *http.Request) error {
	var req answerRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	res, err := h.svc.PracticeCheck(r.Context(), userID(r), chi.URLParam(r, "id"), chi.URLParam(r, "questionId"), req.OptionIDs)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, res)
	return nil
}
