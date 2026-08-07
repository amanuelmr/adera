package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adera-platform/backend/internal/platform/web"
)

// These guards run before Register touches the hasher or the database, so
// they're safe to exercise against a zero-value Service. The deeper
// lifecycle (session issuance, refresh rotation and reuse detection,
// password reset) is covered end-to-end against real Postgres by
// internal/app/auth_integration_test.go.

func TestRegisterRejectsInvalidDisplayName(t *testing.T) {
	s := &Service{}

	_, _, err := s.Register(context.Background(), RegisterInput{
		DisplayName: "A", Email: "user@example.com", Password: "password123",
	}, "")

	var webErr *web.Error
	require.ErrorAs(t, err, &webErr)
	assert.Equal(t, 422, webErr.Status)
}

func TestRegisterRejectsInvalidPassword(t *testing.T) {
	s := &Service{}

	_, _, err := s.Register(context.Background(), RegisterInput{
		DisplayName: "Abebe", Email: "user@example.com", Password: "short",
	}, "")

	var webErr *web.Error
	require.ErrorAs(t, err, &webErr)
	assert.Equal(t, 422, webErr.Status)
}

func TestRegisterRejectsInvalidEmail(t *testing.T) {
	s := &Service{}

	_, _, err := s.Register(context.Background(), RegisterInput{
		DisplayName: "Abebe", Email: "not-an-email", Password: "password123",
	}, "")

	var webErr *web.Error
	require.ErrorAs(t, err, &webErr)
	assert.Equal(t, 422, webErr.Status)
	_, hasDetail := webErr.Details["email"]
	assert.True(t, hasDetail)
}

func TestRegisterRejectsInvalidPhone(t *testing.T) {
	s := &Service{}

	_, _, err := s.Register(context.Background(), RegisterInput{
		DisplayName: "Abebe", Phone: "not-a-phone", Password: "password123",
	}, "")

	var webErr *web.Error
	require.ErrorAs(t, err, &webErr)
	_, hasDetail := webErr.Details["phone"]
	assert.True(t, hasDetail)
}

func TestRegisterRequiresEmailOrPhone(t *testing.T) {
	s := &Service{}

	_, _, err := s.Register(context.Background(), RegisterInput{
		DisplayName: "Abebe", Password: "password123",
	}, "")

	var webErr *web.Error
	require.ErrorAs(t, err, &webErr)
	_, hasDetail := webErr.Details["identifier"]
	assert.True(t, hasDetail)
}

func TestRegisterRejectsUnsupportedLanguage(t *testing.T) {
	s := &Service{}

	_, _, err := s.Register(context.Background(), RegisterInput{
		DisplayName: "Abebe", Email: "user@example.com", Password: "password123", Language: "fr",
	}, "")

	var webErr *web.Error
	require.ErrorAs(t, err, &webErr)
	_, hasDetail := webErr.Details["preferred_language"]
	assert.True(t, hasDetail)
}

func TestConfirmVerificationRejectsUnsupportedChannel(t *testing.T) {
	s := &Service{}

	err := s.ConfirmVerification(context.Background(), uuid.New(), "carrier_pigeon", "123456")

	var webErr *web.Error
	require.ErrorAs(t, err, &webErr)
	assert.Equal(t, 422, webErr.Status)
}
