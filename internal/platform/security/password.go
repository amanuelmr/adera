// Package security implements password hashing, token generation/verification,
// and constant-time secret handling.
package security

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2Params configures Argon2id. Defaults come from config and satisfy the
// OWASP minimum (m=19456 KiB, t=2, p=1); see docs/research/security-and-legal-risks.md §4.
type Argon2Params struct {
	MemoryKiB   uint32
	Iterations  uint32
	Parallelism uint8
}

const (
	saltLen = 16
	keyLen  = 32
)

// Hasher hashes and verifies passwords with Argon2id, encoding hashes in PHC
// string format so parameters can be upgraded without breaking old hashes.
type Hasher struct {
	params Argon2Params
}

func NewHasher(p Argon2Params) *Hasher { return &Hasher{params: p} }

// Hash derives an Argon2id hash in PHC format.
func (h *Hasher) Hash(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generating salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, h.params.Iterations, h.params.MemoryKiB, h.params.Parallelism, keyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, h.params.MemoryKiB, h.params.Iterations, h.params.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// ErrHashMismatch is returned when the password does not match the hash.
var ErrHashMismatch = errors.New("password does not match")

// Verify checks password against an encoded PHC hash in constant time. The
// stored hash's own parameters are used, so parameter upgrades roll forward
// naturally on next login.
func (h *Hasher) Verify(password, encoded string) error {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return errors.New("unsupported password hash format")
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return fmt.Errorf("parsing hash version: %w", err)
	}
	var memory, iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return fmt.Errorf("parsing hash parameters: %w", err)
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return fmt.Errorf("decoding salt: %w", err)
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return fmt.Errorf("decoding hash: %w", err)
	}
	got := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(want)))
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return ErrHashMismatch
	}
	return nil
}

// RandomToken returns a URL-safe token with 256 bits of entropy.
func RandomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// RandomDigits returns an n-digit numeric OTP code.
func RandomDigits(n int) (string, error) {
	const digits = "0123456789"
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating code: %w", err)
	}
	for i := range b {
		b[i] = digits[int(b[i])%10]
	}
	return string(b), nil
}

// HashToken returns the hex-free SHA-256 digest of a token for storage.
// Raw tokens are never persisted.
func HashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// ConstantTimeEqual compares two byte slices without leaking timing.
func ConstantTimeEqual(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}
