package moderation

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adera-platform/backend/internal/platform/web"
)

func assertValidation(t *testing.T, err error) {
	t.Helper()
	var webErr *web.Error
	require.ErrorAs(t, err, &webErr)
	assert.Equal(t, 422, webErr.Status)
}

func TestFileReportRejectsUnsupportedReason(t *testing.T) {
	s := &Service{}
	targetID := uuid.New()

	_, err := s.FileReport(context.Background(), uuid.New(), nil, &targetID, "not_a_reason", "")

	assertValidation(t, err)
}

func TestFileReportRejectsLongDetails(t *testing.T) {
	s := &Service{}
	targetID := uuid.New()

	_, err := s.FileReport(context.Background(), uuid.New(), nil, &targetID, "spam", strings.Repeat("a", 2001))

	assertValidation(t, err)
}

func TestFileReportRequiresExactlyOneSubject(t *testing.T) {
	s := &Service{}
	reviewID, targetID := uuid.New(), uuid.New()

	_, err := s.FileReport(context.Background(), uuid.New(), nil, nil, "spam", "")
	assertValidation(t, err)

	_, err = s.FileReport(context.Background(), uuid.New(), &reviewID, &targetID, "spam", "")
	assertValidation(t, err)
}

func TestAllReportReasonsAreNonEmpty(t *testing.T) {
	assert.NotEmpty(t, ReportReasons)
	for reason, ok := range ReportReasons {
		assert.True(t, ok, reason)
	}
}

func TestQueueRejectsUnsupportedStatus(t *testing.T) {
	s := &Service{}

	_, err := s.Queue(context.Background(), "bogus", 10)

	assertValidation(t, err)
}

func TestResolveReportRejectsUnsupportedStatus(t *testing.T) {
	s := &Service{}

	_, err := s.ResolveReport(context.Background(), uuid.New(), uuid.New(), "bogus", "")

	assertValidation(t, err)
}

func TestDecideReviewRejectsUnsupportedAction(t *testing.T) {
	s := &Service{}

	_, err := s.DecideReview(context.Background(), uuid.New(), uuid.New(), "delete_forever", "")

	assertValidation(t, err)
}

func TestDecideTargetRejectsUnsupportedAction(t *testing.T) {
	s := &Service{}

	_, err := s.DecideTarget(context.Background(), uuid.New(), uuid.New(), "delete_forever", "")

	assertValidation(t, err)
}

func TestDecideEvidenceRejectsUnsupportedDecision(t *testing.T) {
	s := &Service{}

	err := s.DecideEvidence(context.Background(), uuid.New(), uuid.New(), "maybe", "")

	assertValidation(t, err)
}

func TestAddNoteRejectsUnsupportedSubjectType(t *testing.T) {
	s := &Service{}

	err := s.AddNote(context.Background(), uuid.New(), "spaceship", uuid.New(), "note")

	assertValidation(t, err)
}

func TestAddNoteRejectsEmptyOrLongNote(t *testing.T) {
	s := &Service{}

	err := s.AddNote(context.Background(), uuid.New(), "review", uuid.New(), "  ")
	assertValidation(t, err)

	err = s.AddNote(context.Background(), uuid.New(), "review", uuid.New(), strings.Repeat("a", 2001))
	assertValidation(t, err)
}
