package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"samacode/backend/internal/httpx"
)

const (
	tokenIssuer          = "samacode"
	tokenTypeAccess      = "access"
	tokenTypeVerify      = "verification"
	verificationTokenTTL = 10 * time.Minute
)

// Tokens émet et vérifie les JWT HS256 : jetons d'accès et jetons de vérification
// (preuve qu'un OTP a été validé, à usage unique, pour l'inscription ou le changement de PIN).
type Tokens struct {
	secret    []byte
	accessTTL time.Duration
}

func NewTokens(secret []byte, accessTTL time.Duration) *Tokens {
	return &Tokens{secret: secret, accessTTL: accessTTL}
}

type accessClaims struct {
	Type  string   `json:"typ"`
	Roles []string `json:"roles"`
	jwt.RegisteredClaims
}

type verificationClaims struct {
	Type                 string  `json:"typ"`
	Phone                string  `json:"phone"`
	Purpose              Purpose `json:"purpose"`
	jwt.RegisteredClaims         // ID = id de l'OTP vérifié
}

func (t *Tokens) IssueAccess(userID string, roles []string, now time.Time) (string, time.Time, error) {
	exp := now.Add(t.accessTTL)
	tok, err := t.sign(accessClaims{
		Type:  tokenTypeAccess,
		Roles: roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: tokenIssuer, Subject: userID,
			IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(exp),
		},
	})
	return tok, exp, err
}

func (t *Tokens) ParseAccess(raw string) (httpx.Principal, error) {
	var c accessClaims
	if err := t.parse(raw, &c); err != nil || c.Type != tokenTypeAccess || c.Subject == "" {
		return httpx.Principal{}, httpx.ErrUnauthenticated
	}
	return httpx.Principal{UserID: c.Subject, Roles: c.Roles}, nil
}

func (t *Tokens) IssueVerification(otpID, phone string, purpose Purpose, now time.Time) (string, error) {
	return t.sign(verificationClaims{
		Type: tokenTypeVerify, Phone: phone, Purpose: purpose,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: tokenIssuer, ID: otpID,
			IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(verificationTokenTTL)),
		},
	})
}

func (t *Tokens) parseVerification(raw string, purpose Purpose, now time.Time) (verificationClaims, error) {
	var c verificationClaims
	err := t.parse(raw, &c, jwt.WithTimeFunc(func() time.Time { return now }))
	if err != nil || c.Type != tokenTypeVerify || c.Purpose != purpose || c.ID == "" {
		return verificationClaims{}, errInvalidVerification
	}
	return c, nil
}

func (t *Tokens) sign(claims jwt.Claims) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
}

func (t *Tokens) parse(raw string, claims jwt.Claims, extra ...jwt.ParserOption) error {
	opts := append([]jwt.ParserOption{
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(tokenIssuer),
		jwt.WithExpirationRequired(),
	}, extra...)
	_, err := jwt.ParseWithClaims(raw, claims, func(*jwt.Token) (any, error) { return t.secret, nil }, opts...)
	if err != nil {
		return errors.Join(httpx.ErrUnauthenticated, err)
	}
	return nil
}

// newRefreshToken renvoie un jeton opaque aléatoire ; seul son hachage est stocké.
func newRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
