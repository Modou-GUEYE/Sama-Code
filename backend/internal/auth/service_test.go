package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"samacode/backend/internal/httpx"
	"samacode/backend/internal/testutil"
)

const testPhone = "+221771234567"

type fixture struct {
	svc *Service
	sms *testutil.SMS
	now time.Time
}

func newFixture(t *testing.T) *fixture {
	db := testutil.DB(t)
	// Horloge simulée partant de l'heure réelle : ParseAccess (middleware) utilise time.Now.
	f := &fixture{sms: &testutil.SMS{}, now: time.Now().UTC()}
	secret := []byte("test-secret-test-secret-test-secret!")
	f.svc = NewService(db, NewTokens(secret, 15*time.Minute), f.sms, secret, DefaultConfig())
	f.svc.now = func() time.Time { return f.now }
	return f
}

func (f *fixture) advance(d time.Duration) { f.now = f.now.Add(d) }

func assertCode(t *testing.T, err error, code string) *httpx.Error {
	t.Helper()
	var apiErr *httpx.Error
	if !errors.As(err, &apiErr) || apiErr.Code != code {
		t.Fatalf("erreur = %v ; attendu le code %q", err, code)
	}
	return apiErr
}

// register inscrit un candidat avec le PIN 2580 et renvoie ses jetons.
func (f *fixture) register(t *testing.T) TokenPair {
	t.Helper()
	ctx := context.Background()
	if err := f.svc.RequestOTP(ctx, "77 123 45 67", PurposeRegister); err != nil {
		t.Fatal(err)
	}
	res, err := f.svc.VerifyOTP(ctx, "771234567", PurposeRegister, f.sms.LastCode(t, testPhone))
	if err != nil {
		t.Fatal(err)
	}
	pair, err := f.svc.Register(ctx, RegisterInput{
		VerificationToken: res.VerificationToken, DisplayName: "  Awa   Diop ",
		PreferredLanguage: "wo", LearningMode: "audio", PIN: "2580",
	})
	if err != nil {
		t.Fatal(err)
	}
	return pair
}

func TestRegisterAndLogin(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	pair := f.register(t)

	p, err := f.svc.tokens.ParseAccess(pair.AccessToken)
	if err != nil || p.UserID != pair.UserID || !p.HasRole("candidate") {
		t.Fatalf("jeton d'accès invalide : %+v, %v", p, err)
	}

	var name, lang, mode string
	if err := f.svc.db.QueryRow(ctx, `SELECT display_name, preferred_language, learning_mode FROM users WHERE id = $1`,
		pair.UserID).Scan(&name, &lang, &mode); err != nil {
		t.Fatal(err)
	}
	if name != "Awa Diop" || lang != "wo" || mode != "audio" {
		t.Errorf("profil enregistré = %q %q %q", name, lang, mode)
	}

	if _, err := f.svc.Login(ctx, "+221 77 123 45 67", "2580"); err != nil {
		t.Fatalf("connexion avec le bon PIN : %v", err)
	}
	assertCode(t, f.svc.RequestOTP(ctx, testPhone, PurposeRegister), "phone_already_registered")
}

func TestVerificationTokenIsSingleUse(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	if err := f.svc.RequestOTP(ctx, testPhone, PurposeRegister); err != nil {
		t.Fatal(err)
	}
	res, err := f.svc.VerifyOTP(ctx, testPhone, PurposeRegister, f.sms.LastCode(t, testPhone))
	if err != nil {
		t.Fatal(err)
	}
	in := RegisterInput{VerificationToken: res.VerificationToken, DisplayName: "Awa",
		PreferredLanguage: "fr", LearningMode: "text", PIN: "2580"}
	if _, err := f.svc.Register(ctx, in); err != nil {
		t.Fatal(err)
	}
	_, err = f.svc.Register(ctx, in)
	assertCode(t, err, "invalid_verification_token")

	// Un jeton d'inscription ne permet pas de changer le PIN.
	_, err = f.svc.ResetPIN(ctx, res.VerificationToken, "1357")
	assertCode(t, err, "invalid_verification_token")
}

func TestOTPRules(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	assertCode(t, f.svc.RequestOTP(ctx, testPhone, PurposeLogin), "phone_not_registered")
	assertCode(t, f.svc.RequestOTP(ctx, testPhone, "hack"), "validation_error")

	if err := f.svc.RequestOTP(ctx, testPhone, PurposeRegister); err != nil {
		t.Fatal(err)
	}
	first := f.sms.LastCode(t, testPhone)

	// Renvoi trop rapide.
	err := f.svc.RequestOTP(ctx, testPhone, PurposeRegister)
	if e := assertCode(t, err, "otp_too_soon"); e.Details["retry_after_seconds"] != 60 {
		t.Errorf("retry_after_seconds = %v", e.Details["retry_after_seconds"])
	}

	// Un nouveau code invalide l'ancien.
	f.advance(61 * time.Second)
	if err := f.svc.RequestOTP(ctx, testPhone, PurposeRegister); err != nil {
		t.Fatal(err)
	}
	second := f.sms.LastCode(t, testPhone)
	if first != second {
		_, err = f.svc.VerifyOTP(ctx, testPhone, PurposeRegister, first)
		assertCode(t, err, "otp_invalid")
	}

	// Code expiré.
	f.advance(6 * time.Minute)
	_, err = f.svc.VerifyOTP(ctx, testPhone, PurposeRegister, second)
	assertCode(t, err, "otp_invalid")

	// 5 essais maximum par code, le bon code est ensuite refusé.
	if err := f.svc.RequestOTP(ctx, testPhone, PurposeRegister); err != nil {
		t.Fatal(err)
	}
	good := f.sms.LastCode(t, testPhone)
	wrong := "000000"
	if good == wrong {
		wrong = "111111"
	}
	for i := 4; i >= 1; i-- {
		_, err = f.svc.VerifyOTP(ctx, testPhone, PurposeRegister, wrong)
		if e := assertCode(t, err, "otp_invalid"); e.Details["remaining_attempts"] != i {
			t.Errorf("remaining_attempts = %v ; attendu %d", e.Details["remaining_attempts"], i)
		}
	}
	_, err = f.svc.VerifyOTP(ctx, testPhone, PurposeRegister, wrong)
	assertCode(t, err, "otp_too_many_attempts")
	_, err = f.svc.VerifyOTP(ctx, testPhone, PurposeRegister, good)
	assertCode(t, err, "otp_too_many_attempts")

	// Quota horaire : 5 codes par numéro.
	f.advance(61 * time.Second)
	if err := f.svc.RequestOTP(ctx, testPhone, PurposeRegister); err != nil {
		t.Fatal(err)
	}
	f.advance(61 * time.Second)
	if err := f.svc.RequestOTP(ctx, testPhone, PurposeRegister); err != nil {
		t.Fatal(err)
	}
	f.advance(61 * time.Second)
	assertCode(t, f.svc.RequestOTP(ctx, testPhone, PurposeRegister), "otp_quota_exceeded")
}

func TestPINLockout(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	f.register(t)

	for i := 4; i >= 1; i-- {
		_, err := f.svc.Login(ctx, testPhone, "1111")
		if e := assertCode(t, err, "invalid_credentials"); e.Details["remaining_attempts"] != i {
			t.Errorf("remaining_attempts = %v ; attendu %d", e.Details["remaining_attempts"], i)
		}
	}
	_, err := f.svc.Login(ctx, testPhone, "1111")
	assertCode(t, err, "account_locked")

	// Même le bon PIN est refusé pendant le blocage.
	_, err = f.svc.Login(ctx, testPhone, "2580")
	assertCode(t, err, "account_locked")

	// Après 15 minutes, le compte est débloqué.
	f.advance(15*time.Minute + time.Second)
	if _, err := f.svc.Login(ctx, testPhone, "2580"); err != nil {
		t.Fatalf("connexion après blocage : %v", err)
	}

	// Une connexion réussie remet le compteur à zéro.
	for range 4 {
		f.svc.Login(ctx, testPhone, "1111") //nolint:errcheck
	}
	if _, err := f.svc.Login(ctx, testPhone, "2580"); err != nil {
		t.Fatal(err)
	}
	_, err = f.svc.Login(ctx, testPhone, "1111")
	if e := assertCode(t, err, "invalid_credentials"); e.Details["remaining_attempts"] != 4 {
		t.Errorf("compteur non remis à zéro : %v", e.Details["remaining_attempts"])
	}

	var locks int
	f.svc.db.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE action = 'user.locked'`).Scan(&locks) //nolint:errcheck
	if locks != 1 {
		t.Errorf("audit user.locked = %d ; attendu 1", locks)
	}
}

func TestOTPLoginUnlocksAccount(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	f.register(t)
	for range 5 {
		f.svc.Login(ctx, testPhone, "1111") //nolint:errcheck
	}

	f.advance(time.Minute)
	if err := f.svc.RequestOTP(ctx, testPhone, PurposeLogin); err != nil {
		t.Fatal(err)
	}
	res, err := f.svc.VerifyOTP(ctx, testPhone, PurposeLogin, f.sms.LastCode(t, testPhone))
	if err != nil || res.Tokens == nil || res.VerificationToken != "" {
		t.Fatalf("connexion par OTP : %+v, %v", res, err)
	}
	if _, err := f.svc.Login(ctx, testPhone, "2580"); err != nil {
		t.Fatalf("le compte devrait être débloqué : %v", err)
	}
}

func TestResetPIN(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	old := f.register(t)

	f.advance(time.Minute)
	if err := f.svc.RequestOTP(ctx, testPhone, PurposeResetPIN); err != nil {
		t.Fatal(err)
	}
	res, err := f.svc.VerifyOTP(ctx, testPhone, PurposeResetPIN, f.sms.LastCode(t, testPhone))
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.svc.ResetPIN(ctx, res.VerificationToken, "0000")
	assertCode(t, err, "weak_pin")
	if _, err := f.svc.ResetPIN(ctx, res.VerificationToken, "1357"); err != nil {
		t.Fatal(err)
	}

	_, err = f.svc.Login(ctx, testPhone, "2580")
	assertCode(t, err, "invalid_credentials")
	if _, err := f.svc.Login(ctx, testPhone, "1357"); err != nil {
		t.Fatal(err)
	}
	// Les anciennes sessions sont fermées.
	_, err = f.svc.Refresh(ctx, old.RefreshToken)
	assertCode(t, err, "invalid_refresh_token")
}

func TestRefreshRotationAndLogout(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	pair := f.register(t)

	next, err := f.svc.Refresh(ctx, pair.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.svc.Refresh(ctx, pair.RefreshToken)
	assertCode(t, err, "invalid_refresh_token")

	if err := f.svc.Logout(ctx, next.RefreshToken); err != nil {
		t.Fatal(err)
	}
	_, err = f.svc.Refresh(ctx, next.RefreshToken)
	assertCode(t, err, "invalid_refresh_token")

	// Expiration.
	pair, err = f.svc.Login(ctx, testPhone, "2580")
	if err != nil {
		t.Fatal(err)
	}
	f.advance(31 * 24 * time.Hour)
	_, err = f.svc.Refresh(ctx, pair.RefreshToken)
	assertCode(t, err, "invalid_refresh_token")
}

func TestUnknownPhoneLogin(t *testing.T) {
	f := newFixture(t)
	_, err := f.svc.Login(context.Background(), testPhone, "2580")
	if e := assertCode(t, err, "invalid_credentials"); e.Details != nil {
		t.Errorf("aucun détail ne doit être renvoyé pour un numéro inconnu : %v", e.Details)
	}
}
