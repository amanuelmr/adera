package claims

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adera-platform/backend/internal/platform/web"
)

func TestSubmitRejectsInvalidMethod(t *testing.T) {
	s := &Service{}

	_, err := s.Submit(context.Background(), uuid.New(), uuid.New(), "carrier_pigeon", "hello")

	var webErr *web.Error
	require.ErrorAs(t, err, &webErr)
	assert.Equal(t, 422, webErr.Status)
}

func TestSubmitRejectsLongMessage(t *testing.T) {
	s := &Service{}

	_, err := s.Submit(context.Background(), uuid.New(), uuid.New(), "email", strings.Repeat("a", 2001))

	var webErr *web.Error
	require.ErrorAs(t, err, &webErr)
	assert.Equal(t, 422, webErr.Status)
}

func TestSubmitAcceptsEveryDocumentedMethod(t *testing.T) {
	for method := range claimMethods {
		assert.True(t, claimMethods[method], method)
	}
	assert.Len(t, claimMethods, 4)
}

func TestDecideRejectsInvalidDecision(t *testing.T) {
	s := &Service{}

	_, err := s.Decide(context.Background(), uuid.New(), uuid.New(), "maybe", "")

	var webErr *web.Error
	require.ErrorAs(t, err, &webErr)
	assert.Equal(t, 422, webErr.Status)
}

func TestDecideRejectsLongNote(t *testing.T) {
	s := &Service{}

	_, err := s.Decide(context.Background(), uuid.New(), uuid.New(), StatusApproved, strings.Repeat("a", 2001))

	var webErr *web.Error
	require.ErrorAs(t, err, &webErr)
	assert.Equal(t, 422, webErr.Status)
}
