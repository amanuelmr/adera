package security

import (
	"crypto/hmac"
	"crypto/sha256"

	"github.com/google/uuid"
)

// CodeHasher hashes one-time codes (verification, password reset) for
// storage. A six-digit code has only a million values, so a plain digest is
// reversed by trying them all; keyed with a server secret, a copy of the
// table alone reveals nothing. The user and purpose are part of the input,
// so a hash is only good for the code it was issued as.
type CodeHasher struct {
	key []byte
}

// NewCodeHasher derives the code key from secret, labelled so it never
// equals a key used for anything else.
func NewCodeHasher(secret string) *CodeHasher {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte("adera one-time code key v1"))
	return &CodeHasher{key: m.Sum(nil)}
}

// Hash returns the stored form of code issued to userID for purpose.
func (h *CodeHasher) Hash(userID uuid.UUID, purpose, code string) []byte {
	m := hmac.New(sha256.New, h.key)
	m.Write(userID[:])
	m.Write([]byte{0})
	m.Write([]byte(purpose))
	m.Write([]byte{0})
	m.Write([]byte(code))
	return m.Sum(nil)
}
