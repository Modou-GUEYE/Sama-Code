// Package testutil fournit une base de test et un faux expéditeur SMS.
package testutil

import (
	"context"
	"os"
	"regexp"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"samacode/backend/internal/database"
	"samacode/backend/migrations"
)

// DB se connecte à TEST_DATABASE_URL, applique les migrations et vide les tables
// alimentées par les tests. Le test est ignoré si la variable n'est pas définie.
// Les paquets partagent la même base : lancer les tests avec `go test -p 1`.
func DB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL non défini : test d'intégration ignoré")
	}
	ctx := context.Background()
	pool, err := database.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := database.Migrate(ctx, pool, migrations.FS); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE users, otp_codes, refresh_tokens, audit_logs, questions, exam_series, course_categories, media_assets CASCADE`); err != nil {
		t.Fatal(err)
	}
	return pool
}

// SMS mémorise le dernier code à 6 chiffres envoyé à chaque numéro.
type SMS struct {
	mu    sync.Mutex
	codes map[string]string
}

var sixDigits = regexp.MustCompile(`\b\d{6}\b`)

func (s *SMS) Send(_ context.Context, phone, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.codes == nil {
		s.codes = map[string]string{}
	}
	s.codes[phone] = sixDigits.FindString(message)
	return nil
}

func (s *SMS) LastCode(t *testing.T, phone string) string {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	code, ok := s.codes[phone]
	if !ok || code == "" {
		t.Fatalf("aucun code SMS envoyé à %s", phone)
	}
	return code
}
