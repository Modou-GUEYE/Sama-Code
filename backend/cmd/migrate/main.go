// Commande migrate : applique les migrations SQL (à lancer avant chaque déploiement).
package main

import (
	"context"
	"log/slog"
	"os"

	"samacode/backend/internal/database"
	"samacode/backend/migrations"
)

func main() {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		slog.Error("DATABASE_URL est obligatoire")
		os.Exit(1)
	}
	ctx := context.Background()
	pool, err := database.Connect(ctx, url)
	if err != nil {
		slog.Error("connexion impossible", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := database.Migrate(ctx, pool, migrations.FS); err != nil {
		slog.Error("migration échouée", "err", err)
		os.Exit(1)
	}
	slog.Info("base à jour")
}
