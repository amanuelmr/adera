package reviews

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adera-platform/backend/internal/platform/web"
)

func validInput() Input {
	return Input{
		TargetID:        uuid.New(),
		OverallRating:   4,
		Body:            "This is a perfectly reasonable review body for testing.",
		CriterionScores: map[string]int{},
	}
}

func detailOf(t *testing.T, err error) map[string]string {
	t.Helper()
	appErr, ok := err.(*web.Error)
	require.True(t, ok, "expected *web.Error, got %T: %v", err, err)
	return appErr.Details
}

func TestInputValidateHappyPath(t *testing.T) {
	in := validInput()
	assert.NoError(t, in.Validate())
	assert.Equal(t, "ETB", in.Currency, "currency defaults to ETB")
}

func TestInputValidateRatingBounds(t *testing.T) {
	for _, r := range []int{0, 6, -1} {
		in := validInput()
		in.OverallRating = r
		err := in.Validate()
		require.Error(t, err)
		assert.Contains(t, detailOf(t, err), "overall_rating")
	}
}

func TestInputValidateBodyLength(t *testing.T) {
	in := validInput()
	in.Body = "too short"
	require.Error(t, in.Validate())

	in = validInput()
	in.Body = strings.Repeat("ሀ", 5001) // rune count, not bytes
	require.Error(t, in.Validate())

	in = validInput()
	in.Body = strings.Repeat("ሀ", 20) // 20 Amharic runes = 60 bytes, still valid
	assert.NoError(t, in.Validate())
}

func TestInputValidateExpectationRequiresSocialSource(t *testing.T) {
	in := validInput()
	in.ExpectationMatch = "better" // no discovery source at all
	err := in.Validate()
	require.Error(t, err)
	assert.Contains(t, detailOf(t, err)["expectation_match"], "social media")

	in = validInput()
	in.DiscoverySource = "walk_in"
	in.ExpectationMatch = "worse"
	require.Error(t, in.Validate())

	in = validInput()
	in.DiscoverySource = "tiktok"
	in.ExpectationMatch = "worse"
	assert.NoError(t, in.Validate())
}

func TestInputValidateSocialURL(t *testing.T) {
	in := validInput()
	in.SocialMediaURL = "https://www.tiktok.com/@user/video/123"
	assert.NoError(t, in.Validate())

	in.SocialMediaURL = "https://evil.example.com/video"
	require.Error(t, in.Validate())

	in.SocialMediaURL = "http://tiktok.com/insecure"
	require.Error(t, in.Validate())

	in.SocialMediaURL = "https://youtu.be/xyz"
	assert.NoError(t, in.Validate())
}

func TestInputValidateMisc(t *testing.T) {
	in := validInput()
	future := time.Now().Add(72 * time.Hour)
	in.ExperienceDate = &future
	require.Error(t, in.Validate())

	in = validInput()
	neg := -5.0
	in.PricePaid = &neg
	require.Error(t, in.Validate())

	in = validInput()
	in.Currency = "birr"
	require.Error(t, in.Validate())

	in = validInput()
	bad := 9
	in.ReturnLikelihood = &bad
	require.Error(t, in.Validate())

	in = validInput()
	in.DiscoverySource = "carrier_pigeon"
	require.Error(t, in.Validate())

	in = validInput()
	in.Language = "xx"
	require.Error(t, in.Validate())
}

func TestInputValidateDisclosures(t *testing.T) {
	in := validInput()
	assert.NoError(t, in.Validate())
	assert.Equal(t, IncentiveNone, in.IncentiveType)
	assert.Equal(t, ConnectionNone, in.MaterialConnection)

	in = validInput()
	in.IncentiveType = "discount"
	in.MaterialConnection = "family_or_friend"
	in.DisclosureDetails = "Invited by the owner's family."
	assert.NoError(t, in.Validate())

	in = validInput()
	in.IncentiveType = "gift_card"
	err := in.Validate()
	require.Error(t, err)
	assert.Contains(t, detailOf(t, err), "incentive_type")

	in = validInput()
	in.MaterialConnection = ConnectionOther
	err = in.Validate()
	require.Error(t, err)
	assert.Contains(t, detailOf(t, err), "disclosure_details")

	in = validInput()
	in.DisclosureDetails = "orphaned details"
	err = in.Validate()
	require.Error(t, err)
	assert.Contains(t, detailOf(t, err), "disclosure_details")
}
