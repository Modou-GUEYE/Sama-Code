// Commande api : serveur HTTP de l'API Sama Code.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"samacode/backend/internal/auth"
	"samacode/backend/internal/config"
	"samacode/backend/internal/database"
	"samacode/backend/internal/media"
	"samacode/backend/internal/server"
	"samacode/backend/migrations"
)

func main() {
	if err := run(); err != nil {
		slog.Error("arrêt sur erreur", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.IsProduction() {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if cfg.AutoMigrate {
		if err := database.Migrate(ctx, pool, migrations.FS); err != nil {
			return err
		}
	}

	sms, err := auth.NewSMSSender(cfg.SMSProvider)
	if err != nil {
		return err
	}
	tokens := auth.NewTokens(cfg.JWTSecret, cfg.AccessTokenTTL)
	authCfg := auth.DefaultConfig()
	authCfg.RefreshTokenTTL = cfg.RefreshTokenTTL

	mediaStorage, err := media.NewLocalStorage(cfg.MediaDir)
	if err != nil {
		return err
	}

	authSvc := auth.NewService(pool, tokens, sms, cfg.OTPSecret, authCfg)
	if cfg.SMSProvider == "log" && !cfg.IsProduction() {
		authSvc.ExposeDevCodes()
		slog.Warn("mode dev : le code SMS est renvoyé dans la réponse de /auth/otp/request (dev_code)")
	}

	srv := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: server.NewRouter(server.Deps{
			DB:                pool,
			Tokens:            tokens,
			Auth:              authSvc,
			Media:             mediaStorage,
			TrustProxyHeaders: cfg.TrustProxyHeaders,
			Docs:              cfg.DocsEnabled,
		}),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("API démarrée", "addr", cfg.HTTPAddr, "env", cfg.Env)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		slog.Info("arrêt en cours…")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
	return nil
}
