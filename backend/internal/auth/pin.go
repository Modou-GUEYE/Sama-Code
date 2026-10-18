package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/crypto/argon2"

	"samacode/backend/internal/httpx"
)

const pinLength = 4 // D17

var (
	errInvalidPIN = httpx.NewError(http.StatusBadRequest, "invalid_pin", "Le code PIN doit contenir exactement 4 chiffres.")
	errWeakPIN    = httpx.NewError(http.StatusBadRequest, "weak_pin", "Ce code PIN est trop facile à deviner. Choisissez-en un autre.")
)

// ValidatePIN exige 4 chiffres et refuse les suites évidentes (0000, 1234, 9876…).
func ValidatePIN(pin string) error {
	if len(pin) != pinLength {
		return errInvalidPIN
	}
	for i := 0; i < len(pin); i++ {
		if pin[i] < '0' || pin[i] > '9' {
			return errInvalidPIN
		}
	}
	same, asc, desc := true, true, true
	for i := 1; i < len(pin); i++ {
		d := int(pin[i]) - int(pin[i-1])
		same = same && d == 0
		asc = asc && d == 1
		desc = desc && d == -1
	}
	if same || asc || desc {
		return errWeakPIN
	}
	return nil
}

// Paramètres argon2id recommandés par l'OWASP (19 Mio, 2 itérations, 1 thread).
const (
	argonMemoryKiB = 19 * 1024
	argonTime      = 2
	argonThreads   = 1
	argonSaltLen   = 16
	argonKeyLen    = 32
)

func HashPIN(pin string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(pin), salt, argonTime, argonMemoryKiB, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemoryKiB, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)), nil
}

var errMalformedHash = errors.New("hachage argon2id mal formé")

func VerifyPIN(pin, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, errMalformedHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false, errMalformedHash
	}
	var memory, iterations uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &threads); err != nil {
		return false, errMalformedHash
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, errMalformedHash
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, errMalformedHash
	}
	got := argon2.IDKey([]byte(pin), salt, iterations, memory, threads, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

// dummyPINHash sert à vérifier un PIN même quand le numéro est inconnu, pour que
// le temps de réponse ne révèle pas l'existence d'un compte.
var dummyPINHash = sync.OnceValue(func() string {
	h, err := HashPIN("0000")
	if err != nil {
		panic(err)
	}
	return h
})
