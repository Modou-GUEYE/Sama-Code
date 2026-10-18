// Package auth gère l'authentification par téléphone : OTP SMS pour l'inscription,
// la connexion et le changement de PIN ; PIN à 4 chiffres pour la connexion courante (D11, D17).
package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"samacode/backend/internal/audit"
	"samacode/backend/internal/database"
	"samacode/backend/internal/httpx"
	"samacode/backend/internal/users"
)

type Purpose string

const (
	PurposeRegister Purpose = "register"
	PurposeLogin    Purpose = "login"
	PurposeResetPIN Purpose = "reset_pin"
)

func (p Purpose) valid() bool {
	return p == PurposeRegister || p == PurposeLogin || p == PurposeResetPIN
}

var (
	errInvalidPurpose      = httpx.Validation("purpose", "Objectif invalide (register, login ou reset_pin).")
	errPhoneTaken          = httpx.NewError(http.StatusConflict, "phone_already_registered", "Ce numéro a déjà un compte. Connectez-vous.")
	errUnknownPhone        = httpx.NewError(http.StatusNotFound, "phone_not_registered", "Aucun compte avec ce numéro. Inscrivez-vous.")
	errAccountDisabled     = httpx.NewError(http.StatusForbidden, "account_disabled", "Ce compte est désactivé.")
	errOTPTooSoon          = httpx.NewError(http.StatusTooManyRequests, "otp_too_soon", "Patientez avant de demander un nouveau code.")
	errOTPQuota            = httpx.NewError(http.StatusTooManyRequests, "otp_quota_exceeded", "Trop de codes demandés. Réessayez dans une heure.")
	errOTPInvalid          = httpx.NewError(http.StatusBadRequest, "otp_invalid", "Code incorrect ou expiré.")
	errOTPLocked           = httpx.NewError(http.StatusTooManyRequests, "otp_too_many_attempts", "Trop d'essais. Demandez un nouveau code.")
	errSMSUnavailable      = httpx.NewError(http.StatusBadGateway, "sms_unavailable", "Le SMS n'a pas pu être envoyé. Réessayez.")
	errInvalidCredentials  = httpx.NewError(http.StatusUnauthorized, "invalid_credentials", "Numéro ou code PIN incorrect.")
	errAccountLocked       = httpx.NewError(http.StatusLocked, "account_locked", "Compte bloqué après trop d'erreurs. Réessayez plus tard ou connectez-vous avec un code SMS.")
	errInvalidVerification = httpx.NewError(http.StatusUnauthorized, "invalid_verification_token", "Vérification expirée. Recommencez avec un nouveau code SMS.")
	errInvalidRefresh      = httpx.NewError(http.StatusUnauthorized, "invalid_refresh_token", "Session expirée. Reconnectez-vous.")
)

type Config struct {
	OTPTTL            time.Duration
	OTPResendInterval time.Duration
	OTPMaxAttempts    int // essais par code
	OTPMaxPerHour     int // codes par numéro et par heure
	PINMaxAttempts    int // erreurs consécutives avant blocage
	PINLockDuration   time.Duration
	RefreshTokenTTL   time.Duration
}

func DefaultConfig() Config {
	return Config{
		OTPTTL:            5 * time.Minute,
		OTPResendInterval: time.Minute,
		OTPMaxAttempts:    5,
		OTPMaxPerHour:     5,
		PINMaxAttempts:    5,
		PINLockDuration:   15 * time.Minute,
		RefreshTokenTTL:   30 * 24 * time.Hour,
	}
}

type Service struct {
	db        *pgxpool.Pool
	tokens    *Tokens
	sms       SMSSender
	otpSecret []byte
	cfg       Config
	now       func() time.Time

	// Développement uniquement : dernier code envoyé par numéro, renvoyé par l'API
	// pour tester l'app sans fournisseur SMS. nil = désactivé.
	devCodesMu sync.Mutex
	devCodes   map[string]string
}

// ExposeDevCodes fait renvoyer le code OTP dans la réponse HTTP. Jamais en production.
func (s *Service) ExposeDevCodes() { s.devCodes = map[string]string{} }

func (s *Service) takeDevCode(rawPhone string) (string, bool) {
	if s.devCodes == nil {
		return "", false
	}
	phone, err := NormalizePhone(rawPhone)
	if err != nil {
		return "", false
	}
	s.devCodesMu.Lock()
	defer s.devCodesMu.Unlock()
	code, ok := s.devCodes[phone]
	delete(s.devCodes, phone)
	return code, ok
}

func NewService(db *pgxpool.Pool, tokens *Tokens, sms SMSSender, otpSecret []byte, cfg Config) *Service {
	return &Service{db: db, tokens: tokens, sms: sms, otpSecret: otpSecret, cfg: cfg, now: time.Now}
}

type TokenPair struct {
	UserID                string    `json:"user_id"`
	AccessToken           string    `json:"access_token"`
	AccessTokenExpiresAt  time.Time `json:"access_token_expires_at"`
	RefreshToken          string    `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
}

// ---------------------------------------------------------------------------
// OTP
// ---------------------------------------------------------------------------

func (s *Service) RequestOTP(ctx context.Context, rawPhone string, purpose Purpose) error {
	phone, err := NormalizePhone(rawPhone)
	if err != nil {
		return err
	}
	if !purpose.valid() {
		return errInvalidPurpose
	}

	var status string
	err = s.db.QueryRow(ctx, `SELECT status FROM users WHERE phone_number = $1`, phone).Scan(&status)
	exists := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	switch {
	case purpose == PurposeRegister && exists:
		return errPhoneTaken
	case purpose != PurposeRegister && !exists:
		return errUnknownPhone
	case exists && status != "active":
		return errAccountDisabled
	}

	now := s.now()
	var sentLastHour int
	var lastSent *time.Time
	if err := s.db.QueryRow(ctx,
		`SELECT count(*), max(created_at) FROM otp_codes WHERE phone_number = $1 AND created_at > $2`,
		phone, now.Add(-time.Hour)).Scan(&sentLastHour, &lastSent); err != nil {
		return err
	}
	if lastSent != nil {
		if wait := s.cfg.OTPResendInterval - now.Sub(*lastSent); wait > 0 {
			return errOTPTooSoon.With("retry_after_seconds", int(math.Ceil(wait.Seconds())))
		}
	}
	if sentLastHour >= s.cfg.OTPMaxPerHour {
		return errOTPQuota
	}

	code, err := generateOTP()
	if err != nil {
		return err
	}
	err = database.WithTx(ctx, s.db, func(tx pgx.Tx) error {
		// Seul le dernier code demandé reste valide.
		if _, err := tx.Exec(ctx,
			`UPDATE otp_codes SET consumed_at = $3 WHERE phone_number = $1 AND purpose = $2 AND consumed_at IS NULL`,
			phone, purpose, now); err != nil {
			return err
		}
		_, err := tx.Exec(ctx,
			`INSERT INTO otp_codes (phone_number, purpose, code_hash, expires_at, created_at) VALUES ($1, $2, $3, $4, $5)`,
			phone, purpose, s.hashOTP(phone, purpose, code), now.Add(s.cfg.OTPTTL), now)
		return err
	})
	if err != nil {
		return err
	}

	msg := fmt.Sprintf("Sama Code : votre code est %s. Il expire dans %d minutes. Ne le partagez avec personne.",
		code, int(s.cfg.OTPTTL.Minutes()))
	if err := s.sms.Send(ctx, phone, msg); err != nil {
		slog.ErrorContext(ctx, "échec d'envoi du SMS", "err", err)
		return errSMSUnavailable
	}
	if s.devCodes != nil {
		s.devCodesMu.Lock()
		s.devCodes[phone] = code
		s.devCodesMu.Unlock()
	}
	return nil
}

type VerifyResult struct {
	// Inscription et changement de PIN : preuve de possession du numéro, à présenter à l'étape suivante.
	VerificationToken string `json:"verification_token,omitempty"`
	// Connexion par OTP : session ouverte directement.
	Tokens *TokenPair `json:"tokens,omitempty"`
}

func (s *Service) VerifyOTP(ctx context.Context, rawPhone string, purpose Purpose, code string) (VerifyResult, error) {
	phone, err := NormalizePhone(rawPhone)
	if err != nil {
		return VerifyResult{}, err
	}
	if !purpose.valid() {
		return VerifyResult{}, errInvalidPurpose
	}

	now := s.now()
	var res VerifyResult
	// Un code faux doit être compté même si la vérification échoue : on valide la
	// transaction et on renvoie l'erreur métier ensuite.
	var failure error
	err = database.WithTx(ctx, s.db, func(tx pgx.Tx) error {
		var otpID, codeHash string
		var attempts int
		var expiresAt time.Time
		err := tx.QueryRow(ctx,
			`SELECT id, code_hash, attempts, expires_at FROM otp_codes
			  WHERE phone_number = $1 AND purpose = $2 AND consumed_at IS NULL
			  ORDER BY created_at DESC LIMIT 1 FOR UPDATE`,
			phone, purpose).Scan(&otpID, &codeHash, &attempts, &expiresAt)
		if errors.Is(err, pgx.ErrNoRows) {
			failure = errOTPInvalid
			return nil
		}
		if err != nil {
			return err
		}
		if !now.Before(expiresAt) {
			failure = errOTPInvalid
			return nil
		}
		if attempts >= s.cfg.OTPMaxAttempts {
			failure = errOTPLocked
			return nil
		}

		if !hmac.Equal([]byte(codeHash), []byte(s.hashOTP(phone, purpose, code))) {
			attempts++
			if _, err := tx.Exec(ctx, `UPDATE otp_codes SET attempts = $2 WHERE id = $1`, otpID, attempts); err != nil {
				return err
			}
			if attempts >= s.cfg.OTPMaxAttempts {
				failure = errOTPLocked
			} else {
				failure = errOTPInvalid.With("remaining_attempts", s.cfg.OTPMaxAttempts-attempts)
			}
			return nil
		}

		if _, err := tx.Exec(ctx, `UPDATE otp_codes SET consumed_at = $2 WHERE id = $1`, otpID, now); err != nil {
			return err
		}

		if purpose != PurposeLogin {
			res.VerificationToken, err = s.tokens.IssueVerification(otpID, phone, purpose, now)
			return err
		}

		// Connexion par OTP : débloque aussi un compte bloqué par trop d'erreurs de PIN.
		var userID, status string
		if err := tx.QueryRow(ctx,
			`SELECT id, status FROM users WHERE phone_number = $1 FOR UPDATE`, phone).Scan(&userID, &status); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				failure = errUnknownPhone
				return nil
			}
			return err
		}
		if status != "active" {
			failure = errAccountDisabled
			return nil
		}
		if _, err := tx.Exec(ctx,
			`UPDATE users SET failed_pin_attempts = 0, locked_until = NULL WHERE id = $1`, userID); err != nil {
			return err
		}
		pair, err := s.issueTokens(ctx, tx, userID, now)
		res.Tokens = &pair
		return err
	})
	if err != nil {
		return VerifyResult{}, err
	}
	if failure != nil {
		return VerifyResult{}, failure
	}
	return res, nil
}

// ---------------------------------------------------------------------------
// Inscription, connexion, PIN
// ---------------------------------------------------------------------------

type RegisterInput struct {
	VerificationToken string `json:"verification_token"`
	DisplayName       string `json:"display_name"`
	PreferredLanguage string `json:"preferred_language"`
	LearningMode      string `json:"learning_mode"`
	PIN               string `json:"pin"`
}

func (s *Service) Register(ctx context.Context, in RegisterInput) (TokenPair, error) {
	name, err := users.NormalizeDisplayName(in.DisplayName)
	if err != nil {
		return TokenPair{}, err
	}
	if err := users.ValidateLanguage(in.PreferredLanguage); err != nil {
		return TokenPair{}, err
	}
	if err := users.ValidateLearningMode(in.LearningMode); err != nil {
		return TokenPair{}, err
	}
	if err := ValidatePIN(in.PIN); err != nil {
		return TokenPair{}, err
	}
	pinHash, err := HashPIN(in.PIN)
	if err != nil {
		return TokenPair{}, err
	}

	now := s.now()
	var pair TokenPair
	err = database.WithTx(ctx, s.db, func(tx pgx.Tx) error {
		phone, err := s.consumeVerification(ctx, tx, in.VerificationToken, PurposeRegister, now)
		if err != nil {
			return err
		}
		var userID string
		err = tx.QueryRow(ctx,
			`INSERT INTO users (display_name, phone_number, phone_verified_at, pin_hash, preferred_language, learning_mode)
			 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
			name, phone, now, pinHash, in.PreferredLanguage, in.LearningMode).Scan(&userID)
		if database.IsUniqueViolation(err) {
			return errPhoneTaken
		}
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO user_roles (user_id, role_id) SELECT $1, id FROM roles WHERE code = 'candidate'`, userID); err != nil {
			return err
		}
		if err := audit.Log(ctx, tx, audit.Entry{
			ActorUserID: &userID, Action: "user.registered", EntityType: "user", EntityID: &userID,
		}); err != nil {
			return err
		}
		pair, err = s.issueTokens(ctx, tx, userID, now)
		return err
	})
	return pair, err
}

func (s *Service) Login(ctx context.Context, rawPhone, pin string) (TokenPair, error) {
	phone, err := NormalizePhone(rawPhone)
	if err != nil {
		return TokenPair{}, err
	}

	now := s.now()
	var pair TokenPair
	var failure error
	err = database.WithTx(ctx, s.db, func(tx pgx.Tx) error {
		var userID, status string
		var pinHash *string
		var failed int
		var lockedUntil *time.Time
		err := tx.QueryRow(ctx,
			`SELECT id, status, pin_hash, failed_pin_attempts, locked_until FROM users WHERE phone_number = $1 FOR UPDATE`,
			phone).Scan(&userID, &status, &pinHash, &failed, &lockedUntil)
		if errors.Is(err, pgx.ErrNoRows) {
			_, _ = VerifyPIN(pin, dummyPINHash())
			failure = errInvalidCredentials
			return nil
		}
		if err != nil {
			return err
		}
		if status != "active" {
			failure = errAccountDisabled
			return nil
		}
		if lockedUntil != nil && now.Before(*lockedUntil) {
			failure = errAccountLocked.With("locked_until", lockedUntil.UTC())
			return nil
		}

		ok := false
		if pinHash != nil {
			if ok, err = VerifyPIN(pin, *pinHash); err != nil {
				return err
			}
		}
		if !ok {
			failed++
			if failed >= s.cfg.PINMaxAttempts {
				until := now.Add(s.cfg.PINLockDuration)
				if _, err := tx.Exec(ctx,
					`UPDATE users SET failed_pin_attempts = 0, locked_until = $2 WHERE id = $1`, userID, until); err != nil {
					return err
				}
				if err := audit.Log(ctx, tx, audit.Entry{
					Action: "user.locked", EntityType: "user", EntityID: &userID,
					Metadata: map[string]any{"locked_until": until.UTC()},
				}); err != nil {
					return err
				}
				failure = errAccountLocked.With("locked_until", until.UTC())
				return nil
			}
			if _, err := tx.Exec(ctx, `UPDATE users SET failed_pin_attempts = $2 WHERE id = $1`, userID, failed); err != nil {
				return err
			}
			failure = errInvalidCredentials.With("remaining_attempts", s.cfg.PINMaxAttempts-failed)
			return nil
		}

		if _, err := tx.Exec(ctx,
			`UPDATE users SET failed_pin_attempts = 0, locked_until = NULL WHERE id = $1`, userID); err != nil {
			return err
		}
		pair, err = s.issueTokens(ctx, tx, userID, now)
		return err
	})
	if err != nil {
		return TokenPair{}, err
	}
	if failure != nil {
		return TokenPair{}, failure
	}
	return pair, nil
}

// ResetPIN définit un nouveau PIN après vérification par OTP, débloque le compte
// et ferme toutes les autres sessions.
func (s *Service) ResetPIN(ctx context.Context, verificationToken, pin string) (TokenPair, error) {
	if err := ValidatePIN(pin); err != nil {
		return TokenPair{}, err
	}
	pinHash, err := HashPIN(pin)
	if err != nil {
		return TokenPair{}, err
	}

	now := s.now()
	var pair TokenPair
	err = database.WithTx(ctx, s.db, func(tx pgx.Tx) error {
		phone, err := s.consumeVerification(ctx, tx, verificationToken, PurposeResetPIN, now)
		if err != nil {
			return err
		}
		var userID string
		err = tx.QueryRow(ctx,
			`UPDATE users SET pin_hash = $2, failed_pin_attempts = 0, locked_until = NULL
			  WHERE phone_number = $1 AND status = 'active' RETURNING id`,
			phone, pinHash).Scan(&userID)
		if errors.Is(err, pgx.ErrNoRows) {
			return errUnknownPhone
		}
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`UPDATE refresh_tokens SET revoked_at = $2 WHERE user_id = $1 AND revoked_at IS NULL`, userID, now); err != nil {
			return err
		}
		if err := audit.Log(ctx, tx, audit.Entry{
			ActorUserID: &userID, Action: "user.pin_reset", EntityType: "user", EntityID: &userID,
		}); err != nil {
			return err
		}
		pair, err = s.issueTokens(ctx, tx, userID, now)
		return err
	})
	return pair, err
}

// ---------------------------------------------------------------------------
// Sessions (jetons de rafraîchissement)
// ---------------------------------------------------------------------------

// Refresh échange un jeton de rafraîchissement valide contre une nouvelle paire (rotation).
func (s *Service) Refresh(ctx context.Context, raw string) (TokenPair, error) {
	now := s.now()
	var pair TokenPair
	var failure error
	err := database.WithTx(ctx, s.db, func(tx pgx.Tx) error {
		var tokenID, userID, status string
		var expiresAt time.Time
		err := tx.QueryRow(ctx,
			`SELECT rt.id, rt.user_id, rt.expires_at, u.status
			   FROM refresh_tokens rt JOIN users u ON u.id = rt.user_id
			  WHERE rt.token_hash = $1 AND rt.revoked_at IS NULL
			  FOR UPDATE OF rt`,
			hashToken(raw)).Scan(&tokenID, &userID, &expiresAt, &status)
		if errors.Is(err, pgx.ErrNoRows) {
			failure = errInvalidRefresh
			return nil
		}
		if err != nil {
			return err
		}
		if !now.Before(expiresAt) {
			failure = errInvalidRefresh
			return nil
		}
		if status != "active" {
			failure = errAccountDisabled
			return nil
		}
		if _, err := tx.Exec(ctx, `UPDATE refresh_tokens SET revoked_at = $2 WHERE id = $1`, tokenID, now); err != nil {
			return err
		}
		pair, err = s.issueTokens(ctx, tx, userID, now)
		return err
	})
	if err != nil {
		return TokenPair{}, err
	}
	if failure != nil {
		return TokenPair{}, failure
	}
	return pair, nil
}

// Logout révoque le jeton de rafraîchissement ; un jeton inconnu est ignoré.
func (s *Service) Logout(ctx context.Context, raw string) error {
	_, err := s.db.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = $2 WHERE token_hash = $1 AND revoked_at IS NULL`,
		hashToken(raw), s.now())
	return err
}

// ---------------------------------------------------------------------------
// Utilitaires internes
// ---------------------------------------------------------------------------

func (s *Service) issueTokens(ctx context.Context, q database.Querier, userID string, now time.Time) (TokenPair, error) {
	rows, err := q.Query(ctx,
		`SELECT r.code FROM user_roles ur JOIN roles r ON r.id = ur.role_id WHERE ur.user_id = $1 ORDER BY r.code`, userID)
	if err != nil {
		return TokenPair{}, err
	}
	roles, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return TokenPair{}, err
	}

	access, accessExp, err := s.tokens.IssueAccess(userID, roles, now)
	if err != nil {
		return TokenPair{}, err
	}
	refresh, err := newRefreshToken()
	if err != nil {
		return TokenPair{}, err
	}
	refreshExp := now.Add(s.cfg.RefreshTokenTTL)
	if _, err := q.Exec(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at, created_at) VALUES ($1, $2, $3, $4)`,
		userID, hashToken(refresh), refreshExp, now); err != nil {
		return TokenPair{}, err
	}
	return TokenPair{
		UserID:      userID,
		AccessToken: access, AccessTokenExpiresAt: accessExp.UTC(),
		RefreshToken: refresh, RefreshTokenExpiresAt: refreshExp.UTC(),
	}, nil
}

// consumeVerification valide un jeton de vérification et le marque comme utilisé.
func (s *Service) consumeVerification(ctx context.Context, tx pgx.Tx, raw string, purpose Purpose, now time.Time) (string, error) {
	claims, err := s.tokens.parseVerification(raw, purpose, now)
	if err != nil {
		return "", err
	}
	tag, err := tx.Exec(ctx,
		`UPDATE otp_codes SET token_used_at = $2
		  WHERE id = $1 AND phone_number = $3 AND purpose = $4 AND token_used_at IS NULL`,
		claims.ID, now, claims.Phone, purpose)
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() != 1 {
		return "", errInvalidVerification
	}
	return claims.Phone, nil
}

func generateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func (s *Service) hashOTP(phone string, purpose Purpose, code string) string {
	mac := hmac.New(sha256.New, s.otpSecret)
	mac.Write([]byte(phone + "|" + string(purpose) + "|" + code))
	return hex.EncodeToString(mac.Sum(nil))
}
