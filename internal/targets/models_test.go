package targets

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/adera-platform/backend/internal/businesses"
)

func TestValidateSocialLinks(t *testing.T) {
	assert.NoError(t, ValidateSocialLinks(nil))
	assert.NoError(t, ValidateSocialLinks(map[string]string{
		"tiktok":    "https://www.tiktok.com/@somecafe",
		"telegram":  "https://t.me/somestore",
		"instagram": "https://instagram.com/somecafe",
	}))

	// Unknown platform key.
	assert.Error(t, ValidateSocialLinks(map[string]string{"myspace": "https://myspace.com/x"}))
	// Wrong domain for the key (link laundering).
	assert.Error(t, ValidateSocialLinks(map[string]string{"tiktok": "https://evil.example.com/@somecafe"}))
	// Suffix spoofing: nottiktok.com must not pass as tiktok.com.
	assert.Error(t, ValidateSocialLinks(map[string]string{"tiktok": "https://nottiktok.com/@x"}))
	// Subdomain of the allowed domain is fine.
	assert.NoError(t, ValidateSocialLinks(map[string]string{"tiktok": "https://vm.tiktok.com/xyz"}))
	// http is rejected.
	assert.Error(t, ValidateSocialLinks(map[string]string{"telegram": "http://t.me/x"}))
	// javascript: URLs rejected.
	assert.Error(t, ValidateSocialLinks(map[string]string{"facebook": "javascript:alert(1)"}))
}

func TestSlugify(t *testing.T) {
	id := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	assert.Equal(t, "tomoca-coffee-6ba7b810", businesses.Slugify("Tomoca Coffee", id))
	// Non-ASCII letters are dropped (slugs are a-z0-9 only).
	assert.Equal(t, "caf-42-6ba7b810", businesses.Slugify("  Café #42! ", id))
	// Pure-Ethiopic names fall back to an id-based slug rather than
	// producing an empty string.
	assert.Equal(t, "t-6ba7b810", businesses.Slugify("ቶሞካ ቡና", id))
}

func TestValidTargetType(t *testing.T) {
	assert.True(t, ValidTargetType("restaurant"))
	assert.True(t, ValidTargetType("online_seller"))
	assert.False(t, ValidTargetType("spaceship"))
	assert.False(t, ValidTargetType(""))
}
