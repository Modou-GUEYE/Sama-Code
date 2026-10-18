package auth

import (
	"errors"
	"testing"
)

func TestNormalizePhone(t *testing.T) {
	valid := map[string]string{
		"771234567":          "+221771234567",
		"77 123 45 67":       "+221771234567",
		"77-123-45-67":       "+221771234567",
		"+221 77 123 45 67":  "+221771234567",
		"+221771234567":      "+221771234567",
		"00221 76 000 00 00": "+221760000000",
		"221781234567":       "+221781234567",
		"701234567":          "+221701234567",
		"751234567":          "+221751234567",
	}
	for in, want := range valid {
		got, err := NormalizePhone(in)
		if err != nil || got != want {
			t.Errorf("NormalizePhone(%q) = %q, %v ; attendu %q", in, got, err, want)
		}
	}

	invalid := []string{
		"", "77123456", "7712345678", "331234567", // fixe
		"+33612345678", "77a234567", "77 123 45 6+7", "+2217712345678",
	}
	for _, in := range invalid {
		if got, err := NormalizePhone(in); !errors.Is(err, ErrInvalidPhone) {
			t.Errorf("NormalizePhone(%q) = %q, %v ; attendu ErrInvalidPhone", in, got, err)
		}
	}
}

func TestValidatePIN(t *testing.T) {
	for _, pin := range []string{"2580", "1357", "9071", "1212", "4096"} {
		if err := ValidatePIN(pin); err != nil {
			t.Errorf("ValidatePIN(%q) = %v ; attendu nil", pin, err)
		}
	}
	for _, pin := range []string{"", "123", "12345", "12a4", "١٢٣٤"} {
		if err := ValidatePIN(pin); err != errInvalidPIN {
			t.Errorf("ValidatePIN(%q) = %v ; attendu errInvalidPIN", pin, err)
		}
	}
	for _, pin := range []string{"0000", "7777", "1234", "6789", "4321", "9876", "0123"} {
		if err := ValidatePIN(pin); err != errWeakPIN {
			t.Errorf("ValidatePIN(%q) = %v ; attendu errWeakPIN", pin, err)
		}
	}
}

func TestHashAndVerifyPIN(t *testing.T) {
	h1, err := HashPIN("2580")
	if err != nil {
		t.Fatal(err)
	}
	h2, _ := HashPIN("2580")
	if h1 == h2 {
		t.Error("deux hachages du même PIN doivent différer (sel aléatoire)")
	}
	if ok, err := VerifyPIN("2580", h1); !ok || err != nil {
		t.Errorf("bon PIN refusé : %v, %v", ok, err)
	}
	if ok, err := VerifyPIN("2581", h1); ok || err != nil {
		t.Errorf("mauvais PIN accepté : %v, %v", ok, err)
	}
	if _, err := VerifyPIN("2580", "pas-un-hash"); err == nil {
		t.Error("un hachage mal formé doit renvoyer une erreur")
	}
}
