package auth

import (
	"context"
	"fmt"
	"log/slog"
)

// SMSSender envoie un SMS. Le fournisseur réel reste à choisir (Q12).
type SMSSender interface {
	Send(ctx context.Context, phone, message string) error
}

// LogSender écrit le SMS dans les journaux au lieu de l'envoyer. Développement uniquement.
type LogSender struct{}

func (LogSender) Send(ctx context.Context, phone, message string) error {
	slog.WarnContext(ctx, "SMS non envoyé (SMS_PROVIDER=log)", "to", phone, "message", message)
	return nil
}

func NewSMSSender(provider string) (SMSSender, error) {
	switch provider {
	case "log":
		return LogSender{}, nil
	default:
		return nil, fmt.Errorf("fournisseur SMS inconnu : %q", provider)
	}
}
