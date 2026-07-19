package security

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AccessClaims are the JWT claims carried by short-lived access tokens.
// Roles are embedded so request authorization needs no DB round-trip; role
// changes take effect within one access-token TTL (15 minutes by default).
type AccessClaims struct {
	SessionID string   `json:"sid"`
	Roles     []string `json:"roles"`
	jwt.RegisteredClaims
}

// TokenManager signs and verifies access tokens (HS256, algorithm pinned).
type TokenManager struct {
	secret []byte
	ttl    time.Duration
	issuer string
}

func NewTokenManager(secret string, ttl time.Duration) *TokenManager {
	return &TokenManager{secret: []byte(secret), ttl: ttl, issuer: "adera-api"}
}

// AccessTokenTTL exposes the configured lifetime for response bodies.
func (m *TokenManager) AccessTokenTTL() time.Duration { return m.ttl }

// Sign issues an access token for the user/session.
func (m *TokenManager) Sign(userID, sessionID uuid.UUID, roles []string, now time.Time) (string, error) {
	claims := AccessClaims{
		SessionID: sessionID.String(),
		Roles:     roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("signing access token: %w", err)
	}
	return token, nil
}

// ErrInvalidToken covers every verification failure; callers must not leak
// more detail to clients.
var ErrInvalidToken = errors.New("invalid token")

// Verify parses and validates an access token.
func (m *TokenManager) Verify(token string) (userID, sessionID uuid.UUID, roles []string, err error) {
	var claims AccessClaims
	parsed, err := jwt.ParseWithClaims(token, &claims, func(t *jwt.Token) (any, error) {
		return m.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(m.issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !parsed.Valid {
		return uuid.Nil, uuid.Nil, nil, ErrInvalidToken
	}
	userID, err = uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, uuid.Nil, nil, ErrInvalidToken
	}
	sessionID, err = uuid.Parse(claims.SessionID)
	if err != nil {
		return uuid.Nil, uuid.Nil, nil, ErrInvalidToken
	}
	return userID, sessionID, claims.Roles, nil
}
