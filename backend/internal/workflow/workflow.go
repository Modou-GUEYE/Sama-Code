// Package workflow applique le cycle de publication des contenus (CDC §6.2) :
// brouillon → validation pédagogique → publication, avec les droits de chaque rôle (§6.3).
package workflow

import (
	"net/http"
	"slices"

	"samacode/backend/internal/httpx"
)

const (
	Draft     = "draft"
	InReview  = "in_review"
	Published = "published"
	Archived  = "archived"
)

// Rôles pouvant gérer les contenus.
const (
	RoleEditor    = "editor"
	RoleValidator = "validator"
	RoleAdmin     = "admin"
)

var ContentRoles = []string{RoleEditor, RoleValidator, RoleAdmin}

type transition struct{ from, to string }

// Qui peut faire passer un contenu d'un statut à un autre.
var allowed = map[transition][]string{
	{Draft, InReview}:     {RoleEditor, RoleValidator, RoleAdmin}, // soumettre à validation
	{InReview, Draft}:     {RoleEditor, RoleValidator, RoleAdmin}, // retirer / refuser
	{InReview, Published}: {RoleValidator, RoleAdmin},             // publier
	{Published, Draft}:    {RoleValidator, RoleAdmin},             // dépublier pour corriger
	{Published, Archived}: {RoleValidator, RoleAdmin},
	{Archived, Draft}:     {RoleAdmin},
}

var (
	ErrInvalidStatus     = httpx.Validation("status", "Statut invalide (draft, in_review, published, archived).")
	ErrInvalidTransition = httpx.NewError(http.StatusConflict, "invalid_transition", "Ce changement de statut n'est pas permis.")
	ErrForbidden         = httpx.NewError(http.StatusForbidden, "forbidden_transition", "Votre rôle ne permet pas ce changement de statut.")
)

func Valid(status string) bool {
	return status == Draft || status == InReview || status == Published || status == Archived
}

// Check vérifie qu'un utilisateur peut faire passer un contenu de from à to.
func Check(from, to string, p httpx.Principal) error {
	if !Valid(to) {
		return ErrInvalidStatus
	}
	roles, ok := allowed[transition{from, to}]
	if !ok {
		return ErrInvalidTransition.With("from", from).With("to", to)
	}
	if !slices.ContainsFunc(roles, p.HasRole) {
		return ErrForbidden
	}
	return nil
}
