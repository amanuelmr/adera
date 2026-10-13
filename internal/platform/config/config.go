// Package config loads and validates all runtime configuration from the
// environment. Configuration is read once at startup; the rest of the
// application receives an immutable Config value.
package config

import (
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/adera-platform/backend/internal/appversion"
	"github.com/adera-platform/backend/internal/platform/web"
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
	// SessionMaxLifetime ends a session this long after sign-in, however
	// often it is refreshed (SESSION_MAX_LIFETIME).
	SessionMaxLifetime time.Duration

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

	// Mobile client version gate, served by GET /api/v1/app/version. An empty
	// minimum disables the check for that platform.
	AndroidMinVersion    string
	AndroidLatestVersion string
	AndroidStoreURL      string
	IOSMinVersion        string
	IOSLatestVersion     string
	IOSStoreURL          string

	// Android App Links: served at /.well-known/assetlinks.json so https
	// links to the web pages open in the installed app. The SHA-256 values
	// are the app signing certificate fingerprints (comma-separated).
	AndroidPackage     string
	AndroidCertSHA256s []string

	CORSAllowedOrigins []string
	// Where the caller's address comes from (CLIENT_IP_SOURCE): the TCP
	// peer, an X-Forwarded-For hop, or one proxy-written header. Rate limits
	// and fraud signals key on it. See web.IPSource.
	ClientIPSource web.IPSource

	// Key for the HMAC that hashes review fraud signals (network prefix and
	// app install ID). Signals are not recorded when it is empty. Changing it
	// stops new reviews matching older signals, so set it once per environment.
	SignalHashKey string

	// Address for privacy and data requests, shown on /privacy
	// (PRIVACY_CONTACT_EMAIL). Optional: without it the page points to the
	// app's own export and delete controls.
	PrivacyContactEmail string

	// Bearer token for scraping /metrics (METRICS_TOKEN). In production the
	// endpoint isn't served without one; elsewhere it is open.
	MetricsToken string

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
	r := &envReader{}
	cfg := Config{
		Env:      getEnv("APP_ENV", EnvDevelopment),
		HTTPAddr: getEnv("HTTP_ADDR", ":8080"),
		BaseURL:  getEnv("BASE_URL", "http://localhost:8080"),

		DatabaseURL:          os.Getenv("DATABASE_URL"),
		DatabaseMaxConns:     int32(r.int("DATABASE_MAX_CONNS", 10)),
		DatabaseQueryTimeout: r.duration("DATABASE_QUERY_TIMEOUT", 5*time.Second),

		JWTSecret:       os.Getenv("JWT_SECRET"),
		AccessTokenTTL:  r.duration("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL: r.duration("REFRESH_TOKEN_TTL", 30*24*time.Hour),
		// Half a year: rare enough not to nag, short enough that a stolen
		// token doesn't last indefinitely.
		SessionMaxLifetime: r.duration("SESSION_MAX_LIFETIME", 180*24*time.Hour),

		ArgonMemoryKiB:   uint32(r.int("ARGON_MEMORY_KIB", 19456)),
		ArgonIterations:  uint32(r.int("ARGON_ITERATIONS", 2)),
		ArgonParallelism: uint8(r.int("ARGON_PARALLELISM", 1)),

		StorageEnabled:       r.bool("STORAGE_ENABLED", false),
		StorageEndpoint:      getEnv("STORAGE_ENDPOINT", "localhost:9000"),
		StorageAccessKey:     os.Getenv("STORAGE_ACCESS_KEY"),
		StorageSecretKey:     os.Getenv("STORAGE_SECRET_KEY"),
		StorageUseSSL:        r.bool("STORAGE_USE_SSL", false),
		StoragePublicBucket:  getEnv("STORAGE_PUBLIC_BUCKET", "adera-public"),
		StoragePrivateBucket: getEnv("STORAGE_PRIVATE_BUCKET", "adera-private"),
		StorageRegion:        getEnv("STORAGE_REGION", "us-east-1"),
		StoragePublicBaseURL: getEnv("STORAGE_PUBLIC_BASE_URL", "http://localhost:9000/adera-public"),

		SMTPHost:        os.Getenv("SMTP_HOST"),
		SMTPPort:        r.int("SMTP_PORT", 587),
		SMTPUsername:    os.Getenv("SMTP_USERNAME"),
		SMTPPassword:    os.Getenv("SMTP_PASSWORD"),
		SMTPFromAddress: os.Getenv("SMTP_FROM_ADDRESS"),
		SMTPFromName:    getEnv("SMTP_FROM_NAME", "Adera"),
		SMTPTLS:         getEnv("SMTP_TLS", SMTPTLSStartTLS),
		SMTPTimeout:     r.duration("SMTP_TIMEOUT", 10*time.Second),

		FCMCredentialsFile: os.Getenv("FCM_CREDENTIALS_FILE"),
		FCMTimeout:         r.duration("FCM_TIMEOUT", 10*time.Second),

		AndroidMinVersion:    os.Getenv("APP_ANDROID_MIN_VERSION"),
		AndroidLatestVersion: os.Getenv("APP_ANDROID_LATEST_VERSION"),
		AndroidStoreURL:      os.Getenv("APP_ANDROID_STORE_URL"),
		IOSMinVersion:        os.Getenv("APP_IOS_MIN_VERSION"),
		IOSLatestVersion:     os.Getenv("APP_IOS_LATEST_VERSION"),
		IOSStoreURL:          os.Getenv("APP_IOS_STORE_URL"),

		AndroidPackage:     getEnv("ANDROID_PACKAGE", "work.amanuel.adera"),
		AndroidCertSHA256s: splitAndTrim(os.Getenv("ANDROID_CERT_SHA256")),

		CORSAllowedOrigins:  splitAndTrim(getEnv("CORS_ALLOWED_ORIGINS", defaultCORSOrigins())),
		SignalHashKey:       os.Getenv("SIGNAL_HASH_KEY"),
		MetricsToken:        os.Getenv("METRICS_TOKEN"),
		PrivacyContactEmail: os.Getenv("PRIVACY_CONTACT_EMAIL"),

		RateLimitEnabled: r.bool("RATE_LIMIT_ENABLED", true),

		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogFormat: getEnv("LOG_FORMAT", "json"),

		SeedAdminEmail:    getEnv("SEED_ADMIN_EMAIL", "admin@adera.local"),
		SeedAdminPassword: os.Getenv("SEED_ADMIN_PASSWORD"),
	}
	src, err := clientIPSource(r)
	if err != nil {
		return Config{}, err
	}
	cfg.ClientIPSource = src
	// A value that doesn't parse is a typo, and silently using the default
	// instead (rate limiting on when it was meant off, say) hides it.
	if len(r.errs) > 0 {
		return Config{}, errors.Join(r.errs...)
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
	if c.SignalHashKey != "" && len(c.SignalHashKey) < 32 {
		errs = append(errs, errors.New("SIGNAL_HASH_KEY must be at least 32 bytes"))
	}
	if c.PrivacyContactEmail != "" {
		if _, err := mail.ParseAddress(c.PrivacyContactEmail); err != nil {
			errs = append(errs, errors.New("PRIVACY_CONTACT_EMAIL must be a valid email address"))
		}
	}
	if c.MetricsToken != "" && len(c.MetricsToken) < 32 {
		errs = append(errs, errors.New("METRICS_TOKEN must be at least 32 bytes"))
	}
	if c.AccessTokenTTL <= 0 || c.AccessTokenTTL > time.Hour {
		errs = append(errs, errors.New("ACCESS_TOKEN_TTL must be positive and at most 1h"))
	}
	if c.RefreshTokenTTL < time.Hour {
		errs = append(errs, errors.New("REFRESH_TOKEN_TTL must be at least 1h"))
	}
	if c.SessionMaxLifetime < c.RefreshTokenTTL || c.SessionMaxLifetime > 2*365*24*time.Hour {
		errs = append(errs, errors.New("SESSION_MAX_LIFETIME must be at least REFRESH_TOKEN_TTL and at most 2 years"))
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
	// A malformed version (e.g. "v1.2.0") would make every version check
	// fail with a 500 — exactly when an operator is trying to retire a
	// broken client — so it's refused at startup instead.
	for name, v := range map[string]string{
		"APP_ANDROID_MIN_VERSION": c.AndroidMinVersion, "APP_ANDROID_LATEST_VERSION": c.AndroidLatestVersion,
		"APP_IOS_MIN_VERSION": c.IOSMinVersion, "APP_IOS_LATEST_VERSION": c.IOSLatestVersion,
	} {
		if v == "" {
			continue
		}
		if _, err := appversion.Parse(v); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w (use plain numbers like 1.2.0)", name, err))
		}
	}
	if c.FCMCredentialsFile != "" && c.FCMTimeout <= 0 {
		errs = append(errs, errors.New("FCM_TIMEOUT must be positive"))
	}
	if c.Env == EnvProduction && c.SeedAdminPassword != "" {
		errs = append(errs, errors.New("SEED_ADMIN_PASSWORD must not be set in production"))
	}
	if c.Env == EnvProduction {
		errs = append(errs, c.productionErrors()...)
	}
	return errors.Join(errs...)
}

// productionErrors refuses settings that are fine on a laptop but unsafe
// with real users: everything leaving the process must be encrypted, and
// the brakes must be on.
func (c Config) productionErrors() []error {
	var errs []error
	if !c.RateLimitEnabled {
		errs = append(errs, errors.New("RATE_LIMIT_ENABLED must not be false in production"))
	}
	if c.DatabaseURL != "" {
		switch databaseSSLMode(c.DatabaseURL) {
		case "require", "verify-ca", "verify-full":
		default:
			// Unset means "prefer", which quietly falls back to plaintext.
			errs = append(errs, errors.New("DATABASE_URL must set sslmode=require (or verify-ca, verify-full) in production"))
		}
	}
	if !strings.HasPrefix(c.BaseURL, "https://") {
		errs = append(errs, errors.New("BASE_URL must be https in production"))
	}
	if c.StorageEnabled {
		if !c.StorageUseSSL {
			errs = append(errs, errors.New("STORAGE_USE_SSL must be true in production"))
		}
		if !strings.HasPrefix(c.StoragePublicBaseURL, "https://") {
			errs = append(errs, errors.New("STORAGE_PUBLIC_BASE_URL must be https in production"))
		}
	}
	return errs
}

// databaseSSLMode returns the sslmode of a postgres URL or key=value DSN, or
// "" when it isn't set.
func databaseSSLMode(dsn string) string {
	if strings.Contains(dsn, "://") {
		u, err := url.Parse(dsn)
		if err != nil {
			return ""
		}
		return u.Query().Get("sslmode")
	}
	for _, field := range strings.Fields(dsn) {
		if v, ok := strings.CutPrefix(field, "sslmode="); ok {
			return strings.Trim(v, "'")
		}
	}
	return ""
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

// defaultCORSOrigins allows a local web client in development. Production
// allows no cross-origin callers unless they are listed: the app isn't a
// browser, and the web pages are served from the API's own origin.
func defaultCORSOrigins() string {
	if os.Getenv("APP_ENV") == EnvProduction {
		return ""
	}
	return "http://localhost:3000"
}

// envReader reads typed environment values, collecting any that don't
// parse so Load can refuse to start rather than fall back to a default.
type envReader struct {
	errs []error
}

func (r *envReader) int(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		r.errs = append(r.errs, fmt.Errorf("%s must be an integer; got %q", key, v))
		return fallback
	}
	return n
}

func (r *envReader) bool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		r.errs = append(r.errs, fmt.Errorf("%s must be true or false; got %q", key, v))
		return fallback
	}
	return b
}

func (r *envReader) duration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		r.errs = append(r.errs, fmt.Errorf("%s must be a duration such as 15m or 720h; got %q", key, v))
		return fallback
	}
	return d
}

// clientIPSource reads CLIENT_IP_SOURCE. The older TRUST_PROXY_HEADERS=true
// still means the rightmost X-Forwarded-For hop when the new one is unset.
func clientIPSource(r *envReader) (web.IPSource, error) {
	if v := os.Getenv("CLIENT_IP_SOURCE"); v != "" {
		return web.ParseIPSource(v)
	}
	if r.bool("TRUST_PROXY_HEADERS", false) {
		return web.IPSource{Hops: 1}, nil
	}
	return web.IPSource{}, nil
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
