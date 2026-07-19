package users

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizePhone(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"0911234567", "+251911234567"},
		{"0711234567", "+251711234567"},
		{"09 11 23 45 67", "+251911234567"},
		{"0911-23-45-67", "+251911234567"},
		{"+251911234567", "+251911234567"},
		{"251911234567", "+251911234567"},
		{"251711234567", "+251711234567"},
		{"+14155552671", "+14155552671"}, // international E.164 passes through
		{"0911", ""},                     // too short
		{"12345", ""},
		{"", ""},
		{"not a phone", ""},
		{"+0123456789", ""}, // leading zero country code invalid
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, NormalizePhone(tc.in), "input %q", tc.in)
	}
}

func TestValidEmail(t *testing.T) {
	assert.True(t, ValidEmail("user@example.com"))
	assert.True(t, ValidEmail("a.b+tag@sub.example.et"))
	assert.False(t, ValidEmail("not-an-email"))
	assert.False(t, ValidEmail("user@"))
	assert.False(t, ValidEmail("Display Name <user@example.com>"))
}

func TestValidateDisplayName(t *testing.T) {
	assert.NoError(t, ValidateDisplayName("Abebe Kebede"))
	assert.NoError(t, ValidateDisplayName("አበበ ከበደ")) // Amharic names count runes, not bytes
	assert.Error(t, ValidateDisplayName("A"))
	assert.Error(t, ValidateDisplayName(""))
}

func TestValidatePassword(t *testing.T) {
	assert.NoError(t, ValidatePassword("longenough"))
	assert.Error(t, ValidatePassword("short"))
}
