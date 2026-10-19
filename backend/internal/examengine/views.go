// Package examengine gère les examens blancs : démarrage sur l'instantané d'une série
// publiée, réponses, chronomètre côté serveur, soumission idempotente et corrections.
package examengine

import (
	"time"

	"samacode/backend/internal/examseries"
	"samacode/backend/internal/questions"
	"samacode/backend/internal/scoring"
)

// Content : texte et médias dans une langue, tels que vus par le candidat.
type Content struct {
	Text     string  `json:"text"`
	AudioURL *string `json:"audio_url"`
	ImageURL *string `json:"image_url"`
}

type CandidateOption struct {
	ID       string             `json:"id"`
	Code     string             `json:"code"`
	Contents map[string]Content `json:"contents"`
}

// CandidateQuestion : question telle qu'envoyée pendant l'examen, SANS bonnes réponses
// ni explications (CLAUDE.md, règle 3).
type CandidateQuestion struct {
	Position     int                `json:"position"`
	QuestionID   string             `json:"question_id"`
	QuestionType string             `json:"question_type"`
	Contents     map[string]Content `json:"contents"`
	Options      []CandidateOption  `json:"options"`
}

type Session struct {
	ID              string              `json:"id"`
	SeriesID        string              `json:"series_id"`
	LicenseCode     string              `json:"license_code"`
	SeriesNumber    int                 `json:"series_number"`
	SeriesVersion   int                 `json:"series_version"`
	Status          string              `json:"status"`
	SubmitReason    *string             `json:"submit_reason"`
	DurationMinutes int                 `json:"duration_minutes"`
	MaxFaults       int                 `json:"max_faults"`
	StartedAt       time.Time           `json:"started_at"`
	ExpiresAt       time.Time           `json:"expires_at"`
	SubmittedAt     *time.Time          `json:"submitted_at"`
	ServerTime      time.Time           `json:"server_time"`
	Questions       []CandidateQuestion `json:"questions"`
	Answers         map[string][]string `json:"answers"`
	Result          *scoring.Result     `json:"result"`
}

// ReviewQuestion : correction d'une question, après soumission uniquement.
type ReviewQuestion struct {
	CandidateQuestion
	SelectedOptionIDs []string           `json:"selected_option_ids"`
	CorrectOptionIDs  []string           `json:"correct_option_ids"`
	Outcome           scoring.Outcome    `json:"outcome"`
	Explanations      map[string]Content `json:"explanations"`
	RelatedLessonID   *string            `json:"related_lesson_id"`
}

type Review struct {
	SessionID string           `json:"session_id"`
	Result    scoring.Result   `json:"result"`
	Questions []ReviewQuestion `json:"questions"`
}

// HistoryItem : une tentative dans l'historique du candidat.
type HistoryItem struct {
	ID           string          `json:"id"`
	LicenseCode  string          `json:"license_code"`
	SeriesNumber int             `json:"series_number"`
	Status       string          `json:"status"`
	StartedAt    time.Time       `json:"started_at"`
	SubmittedAt  *time.Time      `json:"submitted_at"`
	Result       *scoring.Result `json:"result"`
}

// SeriesSummary : série publiée proposée au candidat, avec ses propres statistiques.
type SeriesSummary struct {
	ID              string `json:"id"`
	SeriesNumber    int    `json:"series_number"`
	QuestionCount   int    `json:"question_count"`
	DurationMinutes int    `json:"duration_minutes"`
	MaxFaults       int    `json:"max_faults"`
	Attempts        int    `json:"attempts"`
	BestFaults      *int   `json:"best_faults"`
	LastPassed      *bool  `json:"last_passed"`
}

func mediaURL(id *string) *string {
	if id == nil {
		return nil
	}
	u := "/api/v1/media/" + *id + "/file"
	return &u
}

func candidateQuestion(q examseries.SnapshotQuestion) CandidateQuestion {
	out := CandidateQuestion{
		Position: q.Position, QuestionID: q.QuestionID, QuestionType: q.QuestionType,
		Contents: map[string]Content{},
	}
	for lang, c := range q.Contents {
		out.Contents[lang] = Content{Text: c.Text, AudioURL: mediaURL(c.AudioMediaID), ImageURL: mediaURL(c.ImageMediaID)}
	}
	for _, o := range q.Options {
		co := CandidateOption{ID: o.ID, Code: o.Code, Contents: map[string]Content{}}
		for lang, c := range o.Contents {
			co.Contents[lang] = Content{Text: c.Text, AudioURL: mediaURL(c.AudioMediaID), ImageURL: mediaURL(c.ImageMediaID)}
		}
		out.Options = append(out.Options, co)
	}
	return out
}

func explanations(contents map[string]questions.Content) map[string]Content {
	out := map[string]Content{}
	for lang, c := range contents {
		if c.Explanation == nil && c.ExplanationAudioMediaID == nil {
			continue
		}
		text := ""
		if c.Explanation != nil {
			text = *c.Explanation
		}
		out[lang] = Content{Text: text, AudioURL: mediaURL(c.ExplanationAudioMediaID)}
	}
	return out
}

func scoringQuestions(snap examseries.Snapshot) []scoring.Question {
	out := make([]scoring.Question, 0, len(snap.Questions))
	for _, q := range snap.Questions {
		sq := scoring.Question{ID: q.QuestionID}
		for _, o := range q.Options {
			if o.IsCorrect {
				sq.CorrectOptionIDs = append(sq.CorrectOptionIDs, o.ID)
			}
		}
		out = append(out, sq)
	}
	return out
}
