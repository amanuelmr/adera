// Package targets owns review targets: the unified reviewable entity
// (business location, online seller, product, service, restaurant, café...)
// with category, place, aliases, and moderation lifecycle.
package targets

import (
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/adera-platform/backend/internal/platform/web"
)

// Target types.
var TargetTypes = []string{
	"business", "location", "online_seller", "product",
	"service", "repair_provider", "restaurant", "cafe",
}

// Moderation statuses for targets.
const (
	ModerationPending   = "pending"
	ModerationPublished = "published"
	ModerationHidden    = "hidden"
	ModerationRemoved   = "removed"
	ModerationMerged    = "merged"
)

// Target is the reviewable entity.
type Target struct {
	ID                 uuid.UUID         `json:"id"`
	TargetType         string            `json:"target_type"`
	CategoryID         uuid.UUID         `json:"category_id"`
	BusinessID         *uuid.UUID        `json:"business_id,omitempty"`
	Name               string            `json:"name"`
	Slug               string            `json:"slug"`
	Description        string            `json:"description"`
	CityID             *uuid.UUID        `json:"city_id,omitempty"`
	AreaID             *uuid.UUID        `json:"area_id,omitempty"`
	AddressText        string            `json:"address_text,omitempty"`
	Latitude           *float64          `json:"latitude,omitempty"`
	Longitude          *float64          `json:"longitude,omitempty"`
	OnlineOnly         bool              `json:"online_only"`
	Phone              string            `json:"phone,omitempty"`
	Website            string            `json:"website,omitempty"`
	SocialLinks        map[string]string `json:"social_links"`
	Aliases            []string          `json:"aliases,omitempty"`
	VerificationStatus string            `json:"verification_status"`
	ModerationStatus   string            `json:"moderation_status"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`

	// Read-time aggregate summary (full stats via /targets/{id}/stats).
	ReviewCount   int      `json:"review_count"`
	AverageRating *float64 `json:"average_rating,omitempty"`
}

// socialDomains maps social_links keys to their allowed host suffixes.
var socialDomains = map[string][]string{
	"tiktok":    {"tiktok.com"},
	"instagram": {"instagram.com"},
	"facebook":  {"facebook.com", "fb.com"},
	"telegram":  {"t.me", "telegram.me"},
	"youtube":   {"youtube.com", "youtu.be"},
}

// ValidateSocialLinks enforces the platform/domain allowlist. Only https URLs
// on known social hosts are stored; the API never fetches them (SSRF-safe).
func ValidateSocialLinks(links map[string]string) error {
	if len(links) > len(socialDomains) {
		return web.ErrValidation("invalid social links").WithDetail("social_links", "too many entries")
	}
	for key, raw := range links {
		domains, ok := socialDomains[key]
		if !ok {
			return web.ErrValidation("invalid social links").WithDetail("social_links", "unsupported platform: "+key)
		}
		if len(raw) > 300 {
			return web.ErrValidation("invalid social links").WithDetail(key, "URL too long")
		}
		u, err := url.Parse(raw)
		if err != nil || u.Scheme != "https" || u.Host == "" {
			return web.ErrValidation("invalid social links").WithDetail(key, "must be a valid https URL")
		}
		host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
		allowed := false
		for _, d := range domains {
			if host == d || strings.HasSuffix(host, "."+d) {
				allowed = true
				break
			}
		}
		if !allowed {
			return web.ErrValidation("invalid social links").WithDetail(key, "URL must be on "+strings.Join(domains, " or "))
		}
	}
	return nil
}

// ValidTargetType reports whether t is a known target type.
func ValidTargetType(t string) bool {
	for _, tt := range TargetTypes {
		if tt == t {
			return true
		}
	}
	return false
}

// editSuggestionFields is the allowlist of keys a community edit suggestion
// may propose changes for.
var editSuggestionFields = map[string]bool{
	"name": true, "description": true, "address_text": true, "phone": true,
	"website": true, "city_id": true, "area_id": true, "online_only": true,
	"social_links": true, "aliases": true, "category_id": true, "target_type": true,
}
