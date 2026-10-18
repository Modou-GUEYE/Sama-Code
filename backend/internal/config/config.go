// Package config charge la configuration depuis les variables d'environnement.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Env               string // development | test | production
	HTTPAddr          string
	DatabaseURL       string
	AutoMigrate       bool
	TrustProxyHeaders bool // lire X-Forwarded-For (uniquement derrière un reverse proxy de confiance)
	JWTSecret         []byte
	OTPSecret         []byte
	SMSProvider       string // log (dev) | … (à choisir, Q12)
	MediaDir          string // stockage local des médias (dev) ; S3 en production (Q8)
	AccessTokenTTL    time.Duration
	RefreshTokenTTL   time.Duration
}

func (c Config) IsProduction() bool { return c.Env == "production" }

func Load() (Config, error) {
	c := Config{
		Env:         getenv("APP_ENV", "development"),
		HTTPAddr:    getenv("HTTP_ADDR", ":8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   []byte(os.Getenv("JWT_SECRET")),
		OTPSecret:   []byte(os.Getenv("OTP_SECRET")),
		SMSProvider: getenv("SMS_PROVIDER", "log"),
		MediaDir:    getenv("MEDIA_DIR", "var/media"),
	}

	var errs []error
	var err error
	if c.AutoMigrate, err = boolEnv("AUTO_MIGRATE", !c.IsProduction()); err != nil {
		errs = append(errs, err)
	}
	if c.TrustProxyHeaders, err = boolEnv("TRUST_PROXY_HEADERS", false); err != nil {
		errs = append(errs, err)
	}
	if c.AccessTokenTTL, err = durationEnv("ACCESS_TOKEN_TTL", 15*time.Minute); err != nil {
		errs = append(errs, err)
	}
	if c.RefreshTokenTTL, err = durationEnv("REFRESH_TOKEN_TTL", 30*24*time.Hour); err != nil {
		errs = append(errs, err)
	}

	if c.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL est obligatoire"))
	}
	if len(c.JWTSecret) < 32 {
		errs = append(errs, errors.New("JWT_SECRET doit contenir au moins 32 caractères"))
	}
	if len(c.OTPSecret) < 32 {
		errs = append(errs, errors.New("OTP_SECRET doit contenir au moins 32 caractères"))
	}
	if c.IsProduction() && c.SMSProvider == "log" {
		errs = append(errs, errors.New("SMS_PROVIDER=log est interdit en production"))
	}
	return c, errors.Join(errs...)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func boolEnv(key string, fallback bool) (bool, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%s : valeur booléenne invalide %q", key, v)
	}
	return b, nil
}

func durationEnv(key string, fallback time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("%s : durée invalide %q (ex. 15m, 720h)", key, v)
	}
	return d, nil
}
