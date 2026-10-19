// Package questions gère la banque de questions : énoncés et explications en
// français et en wolof (texte + audio + image), propositions et bonnes réponses.
package questions

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"samacode/backend/internal/httpx"
)

// Content : énoncé et explication dans une langue.
type Content struct {
	Text                    string  `json:"text"`
	AudioMediaID            *string `json:"audio_media_id"`
	ImageMediaID            *string `json:"image_media_id"`
	Explanation             *string `json:"explanation"`
	ExplanationAudioMediaID *string `json:"explanation_audio_media_id"`
}

type OptionContent struct {
	Text         string  `json:"text"`
	AudioMediaID *string `json:"audio_media_id"`
	ImageMediaID *string `json:"image_media_id"`
}

type Option struct {
	ID        string                   `json:"id,omitempty"`
	Code      string                   `json:"code"`
	IsCorrect bool                     `json:"is_correct"`
	Contents  map[string]OptionContent `json:"contents"`
}

type Question struct {
	ID              string             `json:"id"`
	QuestionType    string             `json:"question_type"`
	RelatedLessonID *string            `json:"related_lesson_id"`
	Status          string             `json:"status"`
	Version         int                `json:"version"`
	Contents        map[string]Content `json:"contents"`
	Options         []Option           `json:"options"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

// Input : corps de création et de modification (remplacement complet).
type Input struct {
	QuestionType    string             `json:"question_type"`
	RelatedLessonID *string            `json:"related_lesson_id"`
	Contents        map[string]Content `json:"contents"`
	Options         []Option           `json:"options"`
}

const (
	minOptions = 2
	maxOptions = 6
	maxText    = 2000
)

var optionCode = regexp.MustCompile(`^[A-F]$`)

var errIncomplete = httpx.NewError(http.StatusUnprocessableEntity, "question_incomplete",
	"La question doit être complète en français avant d'être soumise ou publiée.")

var errWolofText = httpx.Validation("contents.wo", "En wolof, seul l'audio est enregistré (D25).")

// validate contrôle un brouillon : le texte est en français ; le wolof n'a que des audios (D25).
func (in *Input) validate() error {
	if in.QuestionType != "single" && in.QuestionType != "multiple" {
		return httpx.Validation("question_type", "Type invalide (single ou multiple).")
	}
	if in.RelatedLessonID != nil && !httpx.IsUUID(*in.RelatedLessonID) {
		return httpx.Validation("related_lesson_id", "Leçon invalide.")
	}
	for lang, c := range in.Contents {
		if lang != "fr" && lang != "wo" {
			return httpx.Validation("contents", "Langue non prise en charge : "+lang)
		}
		c.Text = strings.TrimSpace(c.Text)
		if lang == "wo" && (c.Text != "" || c.ImageMediaID != nil || (c.Explanation != nil && strings.TrimSpace(*c.Explanation) != "")) {
			return errWolofText
		}
		if len(c.Text) > maxText {
			return httpx.Validation("contents."+lang+".text", "Énoncé trop long.")
		}
		if err := checkMediaIDs("contents."+lang, c.AudioMediaID, c.ImageMediaID, c.ExplanationAudioMediaID); err != nil {
			return err
		}
		in.Contents[lang] = c
	}
	if fr, ok := in.Contents["fr"]; !ok || fr.Text == "" {
		return httpx.Validation("contents.fr.text", "L'énoncé en français est obligatoire.")
	}

	if len(in.Options) < minOptions || len(in.Options) > maxOptions {
		return httpx.Validation("options", fmt.Sprintf("Entre %d et %d propositions.", minOptions, maxOptions))
	}
	seen := map[string]bool{}
	correct := 0
	for i := range in.Options {
		o := &in.Options[i]
		o.Code = strings.ToUpper(strings.TrimSpace(o.Code))
		if !optionCode.MatchString(o.Code) || seen[o.Code] {
			return httpx.Validation("options", "Chaque proposition a une lettre unique de A à F.")
		}
		seen[o.Code] = true
		if o.IsCorrect {
			correct++
		}
		for lang, c := range o.Contents {
			if lang != "fr" && lang != "wo" {
				return httpx.Validation("options", "Langue non prise en charge : "+lang)
			}
			c.Text = strings.TrimSpace(c.Text)
			if lang == "wo" && (c.Text != "" || c.ImageMediaID != nil) {
				return errWolofText
			}
			if err := checkMediaIDs("options."+o.Code+"."+lang, c.AudioMediaID, c.ImageMediaID); err != nil {
				return err
			}
			o.Contents[lang] = c
		}
		if fr := o.Contents["fr"]; fr.Text == "" && fr.ImageMediaID == nil {
			return httpx.Validation("options."+o.Code, "Chaque proposition a un texte ou une image en français.")
		}
	}
	switch {
	case in.QuestionType == "single" && correct != 1:
		return httpx.Validation("options", "Une question à réponse unique a exactement une bonne réponse.")
	case in.QuestionType == "multiple" && correct < 1:
		return httpx.Validation("options", "Une question à réponses multiples a au moins une bonne réponse.")
	}
	return nil
}

// Complete : exigé pour soumettre ou publier. Le texte est en français ; les audios
// (français et wolof) sont recommandés mais pas encore bloquants (D25).
func (q *Question) Complete() error {
	c, ok := q.Contents["fr"]
	if !ok || strings.TrimSpace(c.Text) == "" {
		return errIncomplete.With("missing", "contents.fr")
	}
	for _, o := range q.Options {
		oc := o.Contents["fr"]
		if strings.TrimSpace(oc.Text) == "" && oc.ImageMediaID == nil {
			return errIncomplete.With("missing", "options."+o.Code+".fr")
		}
	}
	return nil
}

func checkMediaIDs(field string, ids ...*string) error {
	for _, id := range ids {
		if id != nil && !httpx.IsUUID(*id) {
			return httpx.Validation(field, "Identifiant de média invalide.")
		}
	}
	return nil
}
