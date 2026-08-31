// Package reviews owns review submission, editing, listing, helpful votes,
// and the transactional maintenance of rating aggregates. Public aggregates
// count only published reviews, and every status/score change adjusts
// target_rating_stats in the same transaction.
package reviews

import (
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/adera-platform/backend/internal/platform/web"
)

// Moderation statuses.
const (
	StatusPending     = "pending"
	StatusPublished   = "published"
	StatusUnderReview = "under_review"
	StatusRejected    = "rejected"
	StatusHidden      = "hidden"
	StatusRemoved     = "removed"
)

// Verification levels, weakest to strongest.
const (
	VerifyNone     = "unverified"
	VerifyMedia    = "media_attached"
	VerifyReceipt  = "receipt_submitted"
	VerifyLocation = "location_verified"
	VerifyPartner  = "partner_verified"
)

// VerifiedLevels are the levels counted in "verified review" aggregates
// (media alone attaches context but does not verify the transaction).
var VerifiedLevels = map[string]bool{
	VerifyReceipt: true, VerifyLocation: true, VerifyPartner: true,
}

// Discovery sources; the social ones enable the Reality Check question.
var DiscoverySources = []string{
	"tiktok", "instagram", "youtube", "facebook", "telegram",
	"friend", "google_maps", "walk_in", "other",
}

// SocialSources answer "was this discovered through social media?".
var SocialSources = map[string]bool{
	"tiktok": true, "instagram": true, "youtube": true, "facebook": true, "telegram": true,
}

// Expectation-match answers for social-media discoveries.
var ExpectationMatches = []string{"better", "as_expected", "worse", "very_different"}

const (
	IncentiveNone  = "none"
	IncentiveOther = "other"

	ConnectionNone  = "none"
	ConnectionOther = "other"
)

// IncentiveTypes describe compensation received in connection with a review.
var IncentiveTypes = []string{
	IncentiveNone, "discount", "free_product_or_service", "payment",
	"contest_entry", "loyalty_points", IncentiveOther,
}

// MaterialConnections describe relationships that could affect how readers
// weigh a review.
var MaterialConnections = []string{
	ConnectionNone, "current_employee", "former_employee", "owner_or_executive",
	"family_or_friend", "business_partner", ConnectionOther,
}

// Policy constants (documented in docs/rating-and-ranking.md and
// docs/moderation-policy.md).
const (
	// A user may submit a new review of the same target only after the
	// cooldown; within it they should update their existing review.
	RepeatCooldown = 30 * 24 * time.Hour
	// Anti-flooding cap.
	MaxReviewsPerDay = 5
)

// Review is the full review record.
type Review struct {
	ID                 uuid.UUID  `json:"id"`
	TargetID           uuid.UUID  `json:"target_id"`
	UserID             uuid.UUID  `json:"user_id"`
	OverallRating      int        `json:"overall_rating"`
	Title              string     `json:"title,omitempty"`
	Body               string     `json:"body"`
	Language           string     `json:"language,omitempty"`
	ExperienceDate     *time.Time `json:"experience_date,omitempty"`
	PricePaid          *float64   `json:"price_paid,omitempty"`
	Currency           string     `json:"currency"`
	WouldRecommend     *bool      `json:"would_recommend,omitempty"`
	ReturnLikelihood   *int       `json:"return_likelihood,omitempty"`
	DiscoverySource    string     `json:"discovery_source,omitempty"`
	ExpectationMatch   string     `json:"expectation_match,omitempty"`
	SocialMediaURL     string     `json:"social_media_url,omitempty"`
	IncentiveType      string     `json:"incentive_type"`
	MaterialConnection string     `json:"material_connection"`
	DisclosureDetails  string     `json:"disclosure_details,omitempty"`
	VerificationLevel  string     `json:"verification_level"`
	ModerationStatus   string     `json:"moderation_status"`
	EditCount          int        `json:"edit_count"`
	EditedAt           *time.Time `json:"edited_at,omitempty"`
	Version            int        `json:"version"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`

	CriterionScores map[string]int `json:"criterion_scores,omitempty"`
}

// ListedReview is a review as shown in listings, with social context.
type ListedReview struct {
	Review
	ReviewerName        string     `json:"reviewer_name"`
	ReviewerReviewCount int        `json:"reviewer_review_count"`
	HelpfulCount        int        `json:"helpful_count"`
	ViewerVoted         bool       `json:"viewer_voted"`
	Media               []MediaRef `json:"media,omitempty"`
	Response            *struct {
		ID        uuid.UUID `json:"id"`
		Body      string    `json:"body"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	} `json:"business_response,omitempty"`
}

// MediaRef is a public, ready media object reference. ThumbURL is empty when
// the upload was already small enough that no derivative was stored, in which
// case clients fall back to URL.
type MediaRef struct {
	ID       uuid.UUID `json:"id"`
	URL      string    `json:"url"`
	ThumbURL string    `json:"thumb_url,omitempty"`
}

// Input is the create/update payload after transport decoding.
type Input struct {
	TargetID           uuid.UUID
	OverallRating      int
	Title              string
	Body               string
	Language           string
	ExperienceDate     *time.Time
	PricePaid          *float64
	Currency           string
	WouldRecommend     *bool
	ReturnLikelihood   *int
	DiscoverySource    string
	ExpectationMatch   string
	SocialMediaURL     string
	IncentiveType      string
	MaterialConnection string
	DisclosureDetails  string
	CriterionScores    map[string]int // keyed by criterion code
}

var socialURLDomains = []string{
	"tiktok.com", "instagram.com", "facebook.com", "fb.com",
	"t.me", "telegram.me", "youtube.com", "youtu.be",
}

// Validate enforces domain rules that do not need the database.
func (in *Input) Validate() error {
	e := web.ErrValidation("invalid review")
	if in.OverallRating < 1 || in.OverallRating > 5 {
		return e.WithDetail("overall_rating", "must be 1-5")
	}
	in.Title = strings.TrimSpace(in.Title)
	if len([]rune(in.Title)) > 120 {
		return e.WithDetail("title", "at most 120 characters")
	}
	in.Body = strings.TrimSpace(in.Body)
	if l := len([]rune(in.Body)); l < 20 || l > 5000 {
		return e.WithDetail("body", "must be 20-5000 characters")
	}
	if in.Language != "" && !isSupportedLanguage(in.Language) {
		return e.WithDetail("language", "unsupported language code")
	}
	if in.ExperienceDate != nil && in.ExperienceDate.After(time.Now().Add(24*time.Hour)) {
		return e.WithDetail("experience_date", "cannot be in the future")
	}
	if in.PricePaid != nil && (*in.PricePaid <= 0 || *in.PricePaid > 100_000_000) {
		return e.WithDetail("price_paid", "must be a positive amount")
	}
	if in.Currency == "" {
		in.Currency = "ETB"
	}
	if len(in.Currency) != 3 || strings.ToUpper(in.Currency) != in.Currency {
		return e.WithDetail("currency", "must be a 3-letter ISO code")
	}
	if in.ReturnLikelihood != nil && (*in.ReturnLikelihood < 1 || *in.ReturnLikelihood > 5) {
		return e.WithDetail("return_likelihood", "must be 1-5")
	}
	if in.DiscoverySource != "" && !contains(DiscoverySources, in.DiscoverySource) {
		return e.WithDetail("discovery_source", "unsupported source")
	}
	if in.ExpectationMatch != "" {
		if !contains(ExpectationMatches, in.ExpectationMatch) {
			return e.WithDetail("expectation_match", "unsupported value")
		}
		if !SocialSources[in.DiscoverySource] {
			return e.WithDetail("expectation_match", "only applies when discovered through social media")
		}
	}
	if in.SocialMediaURL != "" {
		if err := validateSocialURL(in.SocialMediaURL); err != nil {
			return err
		}
	}
	if in.IncentiveType == "" {
		in.IncentiveType = IncentiveNone
	}
	if !contains(IncentiveTypes, in.IncentiveType) {
		return e.WithDetail("incentive_type", "unsupported incentive type")
	}
	if in.MaterialConnection == "" {
		in.MaterialConnection = ConnectionNone
	}
	if !contains(MaterialConnections, in.MaterialConnection) {
		return e.WithDetail("material_connection", "unsupported connection type")
	}
	in.DisclosureDetails = strings.TrimSpace(in.DisclosureDetails)
	if len([]rune(in.DisclosureDetails)) > 500 {
		return e.WithDetail("disclosure_details", "at most 500 characters")
	}
	if (in.IncentiveType == IncentiveOther || in.MaterialConnection == ConnectionOther) &&
		len([]rune(in.DisclosureDetails)) < 3 {
		return e.WithDetail("disclosure_details", "required when a disclosure type is other")
	}
	if in.IncentiveType == IncentiveNone && in.MaterialConnection == ConnectionNone && in.DisclosureDetails != "" {
		return e.WithDetail("disclosure_details", "requires an incentive or material connection")
	}
	return nil
}

// validateSocialURL accepts only https links on supported social platforms.
// The API stores the link; it never fetches it.
func validateSocialURL(raw string) error {
	e := web.ErrValidation("invalid review")
	if len(raw) > 300 {
		return e.WithDetail("social_media_url", "URL too long")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return e.WithDetail("social_media_url", "must be a valid https URL")
	}
	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	for _, d := range socialURLDomains {
		if host == d || strings.HasSuffix(host, "."+d) {
			return nil
		}
	}
	return e.WithDetail("social_media_url", "must link to a supported social platform")
}

func isSupportedLanguage(code string) bool {
	switch code {
	case "am", "en", "om", "ti", "so":
		return true
	}
	return false
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
