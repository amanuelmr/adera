// Package users owns user identity: the user record, roles, profile
// management, and account lifecycle. Authentication (sessions, tokens) lives
// in internal/auth and builds on this package.
package users

import (
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/adera-platform/backend/internal/platform/web"
)

// Account statuses.
const (
	StatusActive      = "active"
	StatusSuspended   = "suspended"
	StatusDeactivated = "deactivated"
)

// SupportedLanguages are the UI/content language codes the platform accepts.
var SupportedLanguages = []string{"am", "en", "om", "ti", "so"}

// User is the identity record. PasswordHash never leaves this package or auth.
type User struct {
	ID                uuid.UUID
	DisplayName       string
	Email             string // "" when absent
	Phone             string // E.164, "" when absent
	PasswordHash      string
	Status            string
	EmailVerifiedAt   *time.Time
	PhoneVerifiedAt   *time.Time
	PreferredLanguage string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Verified reports whether at least one contact channel is verified.
func (u User) Verified() bool {
	return u.EmailVerifiedAt != nil || u.PhoneVerifiedAt != nil
}

var e164Re = regexp.MustCompile(`^\+[1-9][0-9]{6,14}$`)

// NormalizePhone converts common Ethiopian phone formats to E.164:
// 09xxxxxxxx / 07xxxxxxxx → +2519xxxxxxxx / +2517xxxxxxxx; 2519... → +2519...;
// international numbers must already be E.164. Returns "" when invalid.
func NormalizePhone(raw string) string {
	p := strings.Map(func(r rune) rune {
		if r == ' ' || r == '-' || r == '(' || r == ')' {
			return -1
		}
		return r
	}, strings.TrimSpace(raw))

	switch {
	case strings.HasPrefix(p, "09"), strings.HasPrefix(p, "07"):
		if len(p) == 10 {
			p = "+251" + p[1:]
		}
	case strings.HasPrefix(p, "2519"), strings.HasPrefix(p, "2517"):
		p = "+" + p
	}
	if !e164Re.MatchString(p) {
		return ""
	}
	return p
}

// ValidEmail reports whether s is a plausible email address.
func ValidEmail(s string) bool {
	if len(s) > 254 {
		return false
	}
	addr, err := mail.ParseAddress(s)
	return err == nil && addr.Address == s
}

// ValidateDisplayName enforces the shared display-name rule.
func ValidateDisplayName(name string) error {
	n := strings.TrimSpace(name)
	if len([]rune(n)) < 2 || len([]rune(n)) > 80 {
		return web.ErrValidation("invalid profile").WithDetail("display_name", "must be 2-80 characters")
	}
	return nil
}

// ValidLanguage reports whether code is a supported language.
func ValidLanguage(code string) bool {
	for _, l := range SupportedLanguages {
		if l == code {
			return true
		}
	}
	return false
}

// ValidatePassword enforces the password policy (length-based per current
// NIST/OWASP guidance; no composition rules).
func ValidatePassword(pw string) error {
	if len(pw) < 8 || len(pw) > 128 {
		return web.ErrValidation("invalid password").WithDetail("password", "must be 8-128 characters")
	}
	return nil
}
