// Package audit journalise les actions sensibles (CDC §6.1, §10).
package audit

import (
	"context"

	"samacode/backend/internal/database"
)

type Entry struct {
	ActorUserID *string
	Action      string // ex. user.registered, user.locked
	EntityType  string
	EntityID    *string
	Metadata    map[string]any
}

// Log écrit l'entrée avec q, de préférence la transaction de l'action journalisée.
func Log(ctx context.Context, q database.Querier, e Entry) error {
	if e.Metadata == nil {
		e.Metadata = map[string]any{}
	}
	_, err := q.Exec(ctx,
		`INSERT INTO audit_logs (actor_user_id, action, entity_type, entity_id, metadata)
		 VALUES ($1, $2, $3, $4, $5)`,
		e.ActorUserID, e.Action, e.EntityType, e.EntityID, e.Metadata)
	return err
}
