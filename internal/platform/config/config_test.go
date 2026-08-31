package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func validConfig() Config {
	return Config{
		Env:              EnvDevelopment,
		DatabaseURL:      "postgres://localhost/adera",
		JWTSecret:        "development-secret-at-least-32-bytes!!",
		AccessTokenTTL:   15 * time.Minute,
		RefreshTokenTTL:  24 * time.Hour,
		ArgonMemoryKiB:   19456,
		ArgonIterations:  2,
		ArgonParallelism: 1,
	}
}

// withSMTP turns on a valid outbound-email configuration.
func withSMTP(c *Config) {
	c.SMTPHost = "smtp.example.com"
	c.SMTPPort = 587
	c.SMTPFromAddress = "no-reply@adera.example.com"
	c.SMTPFromName = "Adera"
	c.SMTPTLS = SMTPTLSStartTLS
	c.SMTPTimeout = 10 * time.Second
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr bool
	}{
		{"valid config", func(*Config) {}, false},
		{"unknown env", func(c *Config) { c.Env = "staging" }, true},
		{"missing database url", func(c *Config) { c.DatabaseURL = "" }, true},
		{"missing jwt secret in dev is allowed", func(c *Config) { c.JWTSecret = "" }, false},
		{"missing jwt secret in production", func(c *Config) {
			c.Env = EnvProduction
			c.JWTSecret = ""
		}, true},
		{"short jwt secret", func(c *Config) { c.JWTSecret = "too-short" }, true},
		{"zero access token ttl", func(c *Config) { c.AccessTokenTTL = 0 }, true},
		{"access token ttl over an hour", func(c *Config) { c.AccessTokenTTL = 2 * time.Hour }, true},
		{"refresh token ttl under an hour", func(c *Config) { c.RefreshTokenTTL = time.Minute }, true},
		{"argon memory below owasp minimum", func(c *Config) { c.ArgonMemoryKiB = 1024 }, true},
		{"argon iterations below minimum", func(c *Config) { c.ArgonIterations = 1 }, true},
		{"argon parallelism below minimum", func(c *Config) { c.ArgonParallelism = 0 }, true},
		{"storage enabled without credentials", func(c *Config) {
			c.StorageEnabled = true
		}, true},
		{"storage enabled with credentials", func(c *Config) {
			c.StorageEnabled = true
			c.StorageAccessKey = "key"
			c.StorageSecretKey = "secret"
		}, false},
		{"smtp disabled by default", func(*Config) {}, false},
		{"smtp host without from address", func(c *Config) {
			c.SMTPHost = "smtp.example.com"
			c.SMTPPort = 587
			c.SMTPTLS = SMTPTLSStartTLS
			c.SMTPTimeout = 10 * time.Second
		}, true},
		{"valid smtp settings", func(c *Config) { withSMTP(c) }, false},
		{"invalid smtp from address", func(c *Config) {
			withSMTP(c)
			c.SMTPFromAddress = "not-an-address"
		}, true},
		{"smtp port out of range", func(c *Config) {
			withSMTP(c)
			c.SMTPPort = 70000
		}, true},
		{"smtp username without password", func(c *Config) {
			withSMTP(c)
			c.SMTPUsername = "mailer"
			c.SMTPPassword = ""
		}, true},
		{"unknown smtp tls mode", func(c *Config) {
			withSMTP(c)
			c.SMTPTLS = "ssl"
		}, true},
		{"plaintext smtp allowed outside production", func(c *Config) {
			withSMTP(c)
			c.SMTPTLS = SMTPTLSNone
		}, false},
		{"plaintext smtp refused in production", func(c *Config) {
			withSMTP(c)
			c.Env = EnvProduction
			c.JWTSecret = "production-secret-at-least-32-bytes!!"
			c.SMTPTLS = SMTPTLSNone
		}, true},
		{"non-positive smtp timeout", func(c *Config) {
			withSMTP(c)
			c.SMTPTimeout = 0
		}, true},
		{"seed admin password in production", func(c *Config) {
			c.Env = EnvProduction
			c.SeedAdminPassword = "admin12345!"
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.mutate(&cfg)
			err := cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigIsDev(t *testing.T) {
	assert.True(t, Config{Env: EnvDevelopment}.IsDev())
	assert.True(t, Config{Env: EnvTest}.IsDev())
	assert.False(t, Config{Env: EnvProduction}.IsDev())
}

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"single value", "http://localhost:3000", []string{"http://localhost:3000"}},
		{"multiple values", "a,b,c", []string{"a", "b", "c"}},
		{"trims whitespace", " a , b ,c ", []string{"a", "b", "c"}},
		{"drops empty entries", "a,,b,", []string{"a", "b"}},
		{"empty string", "", []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, splitAndTrim(tt.in))
		})
	}
}

func TestSMTPEnabled(t *testing.T) {
	assert.False(t, validConfig().SMTPEnabled())

	cfg := validConfig()
	withSMTP(&cfg)
	assert.True(t, cfg.SMTPEnabled())
}
