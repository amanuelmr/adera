package web

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Middleware is a standard net/http middleware.
type Middleware func(http.Handler) http.Handler

// Chain applies middlewares left-to-right (the first listed runs outermost).
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// RequestID attaches a random request ID to the context and response headers.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf [8]byte
		if _, err := rand.Read(buf[:]); err != nil {
			RespondError(w, r, ErrInternal(err))
			return
		}
		id := hex.EncodeToString(buf[:])
		ctx := context.WithValue(r.Context(), ctxKeyRequestID, id)
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Recover converts panics into 500 responses instead of dropped connections.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				if rec == http.ErrAbortHandler { //nolint:errorlint // sentinel comparison per net/http docs
					panic(rec)
				}
				slog.ErrorContext(r.Context(), "panic recovered",
					"panic", rec, "method", r.Method, "path", r.URL.Path)
				RespondError(w, r, newErr(http.StatusInternalServerError, CodeInternal, "an internal error occurred"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

// Logging emits one structured log line per request. Bodies, tokens, and
// query strings for auth endpoints are never logged.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		slog.InfoContext(r.Context(), "http request",
			"request_id", RequestIDFromContext(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

// SecureHeaders sets conservative security headers appropriate for a JSON API.
func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		h.Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// CORS implements a strict allowlist CORS policy.
func CORS(allowedOrigins []string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && slices.Contains(allowedOrigins, origin) {
				h := w.Header()
				h.Set("Access-Control-Allow-Origin", origin)
				h.Set("Vary", "Origin")
				h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key, If-Match")
				h.Set("Access-Control-Max-Age", "600")
			}
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Principal is the authenticated caller attached to the request context.
type Principal struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	Roles     []string
}

// Role names. Stored in user_roles and embedded in access-token claims.
const (
	RoleCustomer      = "customer"
	RoleBusinessOwner = "business_owner"
	RoleModerator     = "moderator"
	RoleAdmin         = "admin"
)

// HasRole reports whether the principal holds the role. Admins implicitly
// hold the moderator role.
func (p Principal) HasRole(role string) bool {
	if slices.Contains(p.Roles, role) {
		return true
	}
	return role == RoleModerator && slices.Contains(p.Roles, RoleAdmin)
}

// PrincipalFromContext returns the authenticated principal, if any.
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(ctxKeyPrincipal).(Principal)
	return p, ok
}

// ContextWithPrincipal is exposed for tests and the auth middleware.
func ContextWithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, ctxKeyPrincipal, p)
}

// VerifyFunc validates a bearer token and returns the principal.
type VerifyFunc func(ctx context.Context, token string) (Principal, error)

// Authenticate parses an optional bearer token. Invalid tokens are rejected;
// absent tokens pass through unauthenticated (handlers decide with RequireAuth).
func Authenticate(verify VerifyFunc) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				next.ServeHTTP(w, r)
				return
			}
			token, ok := strings.CutPrefix(header, "Bearer ")
			if !ok || token == "" {
				RespondError(w, r, ErrUnauthorized("malformed authorization header"))
				return
			}
			principal, err := verify(r.Context(), token)
			if err != nil {
				RespondError(w, r, ErrUnauthorized("invalid or expired token"))
				return
			}
			next.ServeHTTP(w, r.WithContext(ContextWithPrincipal(r.Context(), principal)))
		})
	}
}

// RequireAuth rejects unauthenticated requests.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := PrincipalFromContext(r.Context()); !ok {
			RespondError(w, r, ErrUnauthorized(""))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireRole rejects callers lacking the role (function-level authorization).
func RequireRole(role string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := PrincipalFromContext(r.Context())
			if !ok {
				RespondError(w, r, ErrUnauthorized(""))
				return
			}
			if !p.HasRole(role) {
				RespondError(w, r, ErrForbidden(""))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
