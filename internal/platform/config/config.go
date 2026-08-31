// Package config loads and validates all runtime configuration from the
// environment. Configuration is read once at startup; the rest of the
// application receives an immutable Config value.
package config

import (
	"errors"
	"fmt"
	"net/mail"
	"os"
	"strconv"
	"strings"
	"time"
)

// Environment names.
const (
	EnvDevelopment = "development"
	EnvTest        = "test"
	EnvProduction  = "production"
)

// SMTP transport-security modes.
const (
	SMTPTLSStartTLS = "starttls" // upgrade a plaintext connection (submission port 587)
	SMTPTLSImplicit = "implicit" // TLS from the first byte (port 465)
	SMTPTLSNone     = "none"     // plaintext; refused in production
)

// Config holds every runtime setting for the API service.
type Config struct {
	Env      string
	HTTPAddr string
	BaseURL  string // public base URL of the API, used in docs and links

	DatabaseURL          string
	DatabaseMaxConns     int32
	DatabaseQueryTimeout time.Duration

	// Auth / tokens.
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	// Argon2id parameters. Defaults follow the OWASP Password Storage
	// Cheat Sheet baseline (19 MiB memory, 2 iterations, 1 lane); see
	// docs/research/security-and-legal-risks.md.
	ArgonMemoryKiB   uint32
	ArgonIterations  uint32
	ArgonParallelism uint8

	// Object storage (S3-compatible; MinIO in development).
	StorageEnabled       bool
	StorageEndpoint      string
	StorageAccessKey     string
	StorageSecretKey     string
	StorageUseSSL        bool
	StoragePublicBucket  string
	StoragePrivateBucket string
	StorageRegion        string
	// Public base URL used to build display URLs for public media objects.
	StoragePublicBaseURL string

	// Outbound email for verification codes and password resets. Delivery is
	// disabled when SMTPHost is empty: verification endpoints then report
	// unavailable rather than pretending a code was sent.
	SMTPHost        string
	SMTPPort        int
	SMTPUsername    string
	SMTPPassword    string
	SMTPFromAddress string
	SMTPFromName    string
	SMTPTLS         string // one of SMTPTLSStartTLS, SMTPTLSImplicit, SMTPTLSNone
	SMTPTimeout     time.Duration

	// Push delivery. FCMCredentialsFile points at a Google service-account
	// key file; delivery is disabled when it is empty and outbox events then
	// accumulate for later replay.
	FCMCredentialsFile string
	FCMTimeout         time.Duration

	CORSAllowedOrigins []string
	TrustProxyHeaders  bool

	RateLimitEnabled bool

	LogLevel  string
	LogFormat string // "json" or "text"

	// Development-only seed admin credentials. Refused in production.
	SeedAdminEmail    string
	SeedAdminPassword string
}

// Load reads configuration from the environment, applying development
// defaults for anything optional.
func Load() (Config, error) {
	cfg := Config{
		Env:      getEnv("APP_ENV", EnvDevelopment),
		HTTPAddr: getEnv("HTTP_ADDR", ":8080"),
		BaseURL:  getEnv("BASE_URL", "http://localhost:8080"),

		DatabaseURL:          os.Getenv("DATABASE_URL"),
		DatabaseMaxConns:     int32(getEnvInt("DATABASE_MAX_CONNS", 10)),
		DatabaseQueryTimeout: getEnvDuration("DATABASE_QUERY_TIMEOUT", 5*time.Second),

		JWTSecret:       os.Getenv("JWT_SECRET"),
		AccessTokenTTL:  getEnvDuration("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL: getEnvDuration("REFRESH_TOKEN_TTL", 30*24*time.Hour),

		ArgonMemoryKiB:   uint32(getEnvInt("ARGON_MEMORY_KIB", 19456)),
		ArgonIterations:  uint32(getEnvInt("ARGON_ITERATIONS", 2)),
		ArgonParallelism: uint8(getEnvInt("ARGON_PARALLELISM", 1)),

		StorageEnabled:       getEnvBool("STORAGE_ENABLED", false),
		StorageEndpoint:      getEnv("STORAGE_ENDPOINT", "localhost:9000"),
		StorageAccessKey:     os.Getenv("STORAGE_ACCESS_KEY"),
		StorageSecretKey:     os.Getenv("STORAGE_SECRET_KEY"),
		StorageUseSSL:        getEnvBool("STORAGE_USE_SSL", false),
		StoragePublicBucket:  getEnv("STORAGE_PUBLIC_BUCKET", "adera-public"),
		StoragePrivateBucket: getEnv("STORAGE_PRIVATE_BUCKET", "adera-private"),
		StorageRegion:        getEnv("STORAGE_REGION", "us-east-1"),
		StoragePublicBaseURL: getEnv("STORAGE_PUBLIC_BASE_URL", "http://localhost:9000/adera-public"),

		SMTPHost:        os.Getenv("SMTP_HOST"),
		SMTPPort:        getEnvInt("SMTP_PORT", 587),
		SMTPUsername:    os.Getenv("SMTP_USERNAME"),
		SMTPPassword:    os.Getenv("SMTP_PASSWORD"),
		SMTPFromAddress: os.Getenv("SMTP_FROM_ADDRESS"),
		SMTPFromName:    getEnv("SMTP_FROM_NAME", "Adera"),
		SMTPTLS:         getEnv("SMTP_TLS", SMTPTLSStartTLS),
		SMTPTimeout:     getEnvDuration("SMTP_TIMEOUT", 10*time.Second),

		FCMCredentialsFile: os.Getenv("FCM_CREDENTIALS_FILE"),
		FCMTimeout:         getEnvDuration("FCM_TIMEOUT", 10*time.Second),

		CORSAllowedOrigins: splitAndTrim(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")),
		TrustProxyHeaders:  getEnvBool("TRUST_PROXY_HEADERS", false),

		RateLimitEnabled: getEnvBool("RATE_LIMIT_ENABLED", true),

		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogFormat: getEnv("LOG_FORMAT", "json"),

		SeedAdminEmail:    getEnv("SEED_ADMIN_EMAIL", "admin@adera.local"),
		SeedAdminPassword: os.Getenv("SEED_ADMIN_PASSWORD"),
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Validate enforces invariants that must hold before the service starts.
func (c Config) Validate() error {
	var errs []error

	switch c.Env {
	case EnvDevelopment, EnvTest, EnvProduction:
	default:
		errs = append(errs, fmt.Errorf("APP_ENV must be one of development, test, production; got %q", c.Env))
	}
	if c.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}
	if c.JWTSecret == "" {
		if c.Env == EnvProduction {
			errs = append(errs, errors.New("JWT_SECRET is required in production"))
		}
	} else if len(c.JWTSecret) < 32 {
		errs = append(errs, errors.New("JWT_SECRET must be at least 32 bytes"))
	}
	if c.AccessTokenTTL <= 0 || c.AccessTokenTTL > time.Hour {
		errs = append(errs, errors.New("ACCESS_TOKEN_TTL must be positive and at most 1h"))
	}
	if c.RefreshTokenTTL < time.Hour {
		errs = append(errs, errors.New("REFRESH_TOKEN_TTL must be at least 1h"))
	}
	if c.ArgonMemoryKiB < 19456 {
		errs = append(errs, errors.New("ARGON_MEMORY_KIB must be at least 19456 (19 MiB, OWASP minimum)"))
	}
	if c.ArgonIterations < 2 {
		errs = append(errs, errors.New("ARGON_ITERATIONS must be at least 2"))
	}
	if c.ArgonParallelism < 1 {
		errs = append(errs, errors.New("ARGON_PARALLELISM must be at least 1"))
	}
	if c.StorageEnabled && (c.StorageAccessKey == "" || c.StorageSecretKey == "") {
		errs = append(errs, errors.New("STORAGE_ACCESS_KEY and STORAGE_SECRET_KEY are required when STORAGE_ENABLED=true"))
	}
	if c.SMTPHost != "" {
		if c.SMTPPort <= 0 || c.SMTPPort > 65535 {
			errs = append(errs, errors.New("SMTP_PORT must be between 1 and 65535"))
		}
		if c.SMTPFromAddress == "" {
			errs = append(errs, errors.New("SMTP_FROM_ADDRESS is required when SMTP_HOST is set"))
		} else if _, err := mail.ParseAddress(c.SMTPFromAddress); err != nil {
			errs = append(errs, errors.New("SMTP_FROM_ADDRESS must be a valid email address"))
		}
		if c.SMTPUsername != "" && c.SMTPPassword == "" {
			errs = append(errs, errors.New("SMTP_PASSWORD is required when SMTP_USERNAME is set"))
		}
		switch c.SMTPTLS {
		case SMTPTLSStartTLS, SMTPTLSImplicit:
		case SMTPTLSNone:
			// Plaintext SMTP would send the OTP, and any SMTP password, in
			// the clear; only tolerable against a local development relay.
			if c.Env == EnvProduction {
				errs = append(errs, errors.New("SMTP_TLS must not be none in production"))
			}
		default:
			errs = append(errs, fmt.Errorf("SMTP_TLS must be one of starttls, implicit, none; got %q", c.SMTPTLS))
		}
		if c.SMTPTimeout <= 0 {
			errs = append(errs, errors.New("SMTP_TIMEOUT must be positive"))
		}
	}
	if c.FCMCredentialsFile != "" && c.FCMTimeout <= 0 {
		errs = append(errs, errors.New("FCM_TIMEOUT must be positive"))
	}
	if c.Env == EnvProduction && c.SeedAdminPassword != "" {
		errs = append(errs, errors.New("SEED_ADMIN_PASSWORD must not be set in production"))
	}
	return errors.Join(errs...)
}

// IsDev reports whether the service runs in a non-production environment.
func (c Config) IsDev() bool { return c.Env != EnvProduction }

// SMTPEnabled reports whether outbound email delivery is configured.
func (c Config) SMTPEnabled() bool { return c.SMTPHost != "" }

// PushEnabled reports whether push delivery is configured.
func (c Config) PushEnabled() bool { return c.FCMCredentialsFile != "" }

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
