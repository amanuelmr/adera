package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/adera-platform/backend/internal/platform/web"
)

func validConfig() Config {
	return Config{
		Env:                EnvDevelopment,
		DatabaseURL:        "postgres://localhost/adera",
		JWTSecret:          "development-secret-at-least-32-bytes!!",
		AccessTokenTTL:     15 * time.Minute,
		RefreshTokenTTL:    24 * time.Hour,
		SessionMaxLifetime: 30 * 24 * time.Hour,
		ArgonMemoryKiB:     19456,
		ArgonIterations:    2,
		ArgonParallelism:   1,
	}
}

// withProduction makes c a valid production configuration, so a production
// case fails only for the reason it tests.
func withProduction(c *Config) {
	c.Env = EnvProduction
	c.JWTSecret = "production-secret-at-least-32-bytes!!"
	c.DatabaseURL = "postgres://u:p@db.example/adera?sslmode=require"
	c.BaseURL = "https://adera.example"
	c.RateLimitEnabled = true
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
		{"signal hash key unset disables signals", func(c *Config) { c.SignalHashKey = "" }, false},
		{"short signal hash key", func(c *Config) { c.SignalHashKey = "too-short" }, true},
		{"signal hash key", func(c *Config) { c.SignalHashKey = "signal-key-at-least-32-bytes-long!!" }, false},
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
			withProduction(c)
			c.SMTPTLS = SMTPTLSNone
		}, true},
		{"non-positive smtp timeout", func(c *Config) {
			withSMTP(c)
			c.SMTPTimeout = 0
		}, true},
		{"push disabled by default", func(*Config) {}, false},
		{"valid push credentials", func(c *Config) {
			c.FCMCredentialsFile = "/run/secrets/fcm.json"
			c.FCMTimeout = 10 * time.Second
		}, false},
		{"non-positive push timeout", func(c *Config) {
			c.FCMCredentialsFile = "/run/secrets/fcm.json"
			c.FCMTimeout = 0
		}, true},
		{"valid production config", withProduction, false},
		{"seed admin password in production", func(c *Config) {
			withProduction(c)
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

func TestPushEnabled(t *testing.T) {
	assert.False(t, validConfig().PushEnabled())

	cfg := validConfig()
	cfg.FCMCredentialsFile = "/run/secrets/fcm.json"
	assert.True(t, cfg.PushEnabled())
}

func TestConfigValidateRejectsMalformedAppVersions(t *testing.T) {
	c := validConfig()
	c.AndroidMinVersion = "v1.2.0"
	err := c.Validate()
	if !assert.Error(t, err) {
		return
	}
	assert.Contains(t, err.Error(), "APP_ANDROID_MIN_VERSION")

	c.AndroidMinVersion = "1.2.0"
	c.AndroidLatestVersion = "1.3"
	assert.NoError(t, c.Validate())
}

func TestClientIPSourceFromEnv(t *testing.T) {
	t.Run("unset is the TCP peer", func(t *testing.T) {
		t.Setenv("CLIENT_IP_SOURCE", "")
		t.Setenv("TRUST_PROXY_HEADERS", "")
		src, err := clientIPSource(&envReader{})
		assert.NoError(t, err)
		assert.Equal(t, web.IPSource{}, src)
	})
	t.Run("legacy TRUST_PROXY_HEADERS is the rightmost hop", func(t *testing.T) {
		t.Setenv("CLIENT_IP_SOURCE", "")
		t.Setenv("TRUST_PROXY_HEADERS", "true")
		src, err := clientIPSource(&envReader{})
		assert.NoError(t, err)
		assert.Equal(t, web.IPSource{Hops: 1}, src)
	})
	t.Run("CLIENT_IP_SOURCE wins", func(t *testing.T) {
		t.Setenv("CLIENT_IP_SOURCE", "header:CF-Connecting-IP")
		t.Setenv("TRUST_PROXY_HEADERS", "true")
		src, err := clientIPSource(&envReader{})
		assert.NoError(t, err)
		assert.Equal(t, web.IPSource{Header: "Cf-Connecting-Ip"}, src)
	})
	t.Run("a bad value stops startup", func(t *testing.T) {
		t.Setenv("CLIENT_IP_SOURCE", "xff:zero")
		_, err := clientIPSource(&envReader{})
		assert.Error(t, err)
	})
}

func TestConfigValidateMetricsToken(t *testing.T) {
	c := validConfig()
	c.MetricsToken = "too-short"
	assert.ErrorContains(t, c.Validate(), "METRICS_TOKEN")
	c.MetricsToken = "metrics-scrape-token-at-least-32-bytes"
	assert.NoError(t, c.Validate())
}

func TestConfigValidateSessionMaxLifetime(t *testing.T) {
	c := validConfig()
	c.SessionMaxLifetime = c.RefreshTokenTTL - time.Minute
	assert.ErrorContains(t, c.Validate(), "SESSION_MAX_LIFETIME")
	c.SessionMaxLifetime = 3 * 365 * 24 * time.Hour
	assert.ErrorContains(t, c.Validate(), "SESSION_MAX_LIFETIME")
	c.SessionMaxLifetime = c.RefreshTokenTTL
	assert.NoError(t, c.Validate())
}

func TestProductionRefusesUnsafeSettings(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
		want   string
	}{
		{"rate limiting off", func(c *Config) { c.RateLimitEnabled = false }, "RATE_LIMIT_ENABLED"},
		{"database without sslmode", func(c *Config) { c.DatabaseURL = "postgres://u:p@db.example/adera" }, "sslmode"},
		{"database sslmode prefer", func(c *Config) { c.DatabaseURL = "postgres://u:p@db.example/adera?sslmode=prefer" }, "sslmode"},
		{"database sslmode disable", func(c *Config) { c.DatabaseURL = "host=db.example sslmode=disable" }, "sslmode"},
		{"plain http base url", func(c *Config) { c.BaseURL = "http://adera.example" }, "BASE_URL"},
		{"storage without tls", func(c *Config) {
			c.StorageEnabled, c.StorageAccessKey, c.StorageSecretKey = true, "k", "s"
			c.StoragePublicBaseURL = "https://media.adera.example"
		}, "STORAGE_USE_SSL"},
		{"storage public url over http", func(c *Config) {
			c.StorageEnabled, c.StorageAccessKey, c.StorageSecretKey = true, "k", "s"
			c.StorageUseSSL = true
			c.StoragePublicBaseURL = "http://media.adera.example"
		}, "STORAGE_PUBLIC_BASE_URL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := validConfig()
			withProduction(&c)
			tt.mutate(&c)
			assert.ErrorContains(t, c.Validate(), tt.want)

			c.Env = EnvDevelopment
			assert.NoError(t, c.Validate(), "allowed in development")
		})
	}

	ok := validConfig()
	withProduction(&ok)
	ok.StorageEnabled, ok.StorageAccessKey, ok.StorageSecretKey = true, "k", "s"
	ok.StorageUseSSL, ok.StoragePublicBaseURL = true, "https://media.adera.example"
	for _, dsn := range []string{
		"postgresql://u:p@ep-x-pooler.eu-central-1.aws.neon.tech/neondb?sslmode=require&channel_binding=require",
		"postgres://u:p@db.example/adera?sslmode=verify-full",
		"host=db.example dbname=adera sslmode='verify-ca'",
	} {
		ok.DatabaseURL = dsn
		assert.NoError(t, ok.Validate(), dsn)
	}
}

func TestLoadRefusesMalformedValues(t *testing.T) {
	t.Setenv("APP_ENV", EnvDevelopment)
	t.Setenv("DATABASE_URL", "postgres://localhost/adera")
	t.Setenv("RATE_LIMIT_ENABLED", "flase")
	t.Setenv("ACCESS_TOKEN_TTL", "15 minutes")
	t.Setenv("SMTP_PORT", "five-eight-seven")
	_, err := Load()
	assert.ErrorContains(t, err, "RATE_LIMIT_ENABLED")
	assert.ErrorContains(t, err, "ACCESS_TOKEN_TTL")
	assert.ErrorContains(t, err, "SMTP_PORT")
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/adera")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")

	t.Setenv("APP_ENV", EnvDevelopment)
	c, err := Load()
	assert.NoError(t, err)
	assert.Equal(t, []string{"http://localhost:3000"}, c.CORSAllowedOrigins)

	t.Setenv("APP_ENV", EnvProduction)
	t.Setenv("JWT_SECRET", "production-secret-at-least-32-bytes!!")
	t.Setenv("DATABASE_URL", "postgres://u:p@db.example/adera?sslmode=require")
	t.Setenv("BASE_URL", "https://adera.example")
	c, err = Load()
	assert.NoError(t, err)
	assert.Empty(t, c.CORSAllowedOrigins, "no cross-origin callers by default in production")
}
