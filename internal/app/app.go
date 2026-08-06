// Package app wires every module into the HTTP API. It is shared by cmd/api
// and the integration test harness.
package app

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/time/rate"

	"github.com/adera-platform/backend/internal/admin"
	"github.com/adera-platform/backend/internal/auth"
	"github.com/adera-platform/backend/internal/businesses"
	"github.com/adera-platform/backend/internal/categories"
	"github.com/adera-platform/backend/internal/claims"
	"github.com/adera-platform/backend/internal/locations"
	"github.com/adera-platform/backend/internal/media"
	"github.com/adera-platform/backend/internal/moderation"
	"github.com/adera-platform/backend/internal/notifications"
	"github.com/adera-platform/backend/internal/platform/config"
	"github.com/adera-platform/backend/internal/platform/ratelimit"
	"github.com/adera-platform/backend/internal/platform/security"
	"github.com/adera-platform/backend/internal/platform/storage"
	"github.com/adera-platform/backend/internal/platform/web"
	"github.com/adera-platform/backend/internal/privacy"
	"github.com/adera-platform/backend/internal/ratings"
	"github.com/adera-platform/backend/internal/reviews"
	"github.com/adera-platform/backend/internal/search"
	"github.com/adera-platform/backend/internal/targets"
	"github.com/adera-platform/backend/internal/users"
)

// BuildAPI constructs the fully-wired HTTP handler.
func BuildAPI(cfg config.Config, pool *pgxpool.Pool, store storage.Store, provider auth.Provider) http.Handler {
	hasher := security.NewHasher(security.Argon2Params{
		MemoryKiB:   cfg.ArgonMemoryKiB,
		Iterations:  cfg.ArgonIterations,
		Parallelism: cfg.ArgonParallelism,
	})
	tokens := security.NewTokenManager(cfg.JWTSecret, cfg.AccessTokenTTL)

	limiter := func(k *ratelimit.Keyed) ratelimit.Limiter {
		if cfg.RateLimitEnabled {
			return k
		}
		k.Close()
		return ratelimit.Unlimited{}
	}
	loginLimiter := limiter(ratelimit.PerMinute(5))
	otpLimiter := limiter(ratelimit.PerMinute(5))
	reviewLimiter := limiter(ratelimit.PerMinute(10))
	reportLimiter := limiter(ratelimit.PerMinute(10))
	presignLimiter := limiter(ratelimit.NewKeyed(rate.Limit(20.0/3600.0), 20))
	searchLimiter := limiter(ratelimit.NewKeyed(2, 10))

	usersRepo := users.NewRepo(pool)
	notificationSvc := notifications.NewService(pool)
	privacySvc := privacy.NewService(pool, notificationSvc)
	authSvc := auth.NewService(pool, usersRepo, hasher, tokens, provider, cfg.RefreshTokenTTL)
	categoriesRepo := categories.NewRepo(pool)
	locationsRepo := locations.NewRepo(pool)
	bizRepo := businesses.NewRepo(pool, notificationSvc)
	targetsRepo := targets.NewRepo(pool)
	reviewsRepo := reviews.NewRepo(pool, cfg.StoragePublicBaseURL)
	ratingsRepo := ratings.NewRepo(pool)
	searchRepo := search.NewRepo(pool)
	mediaSvc := media.NewService(pool, store, reviewsRepo, cfg.StoragePrivateBucket, cfg.StoragePublicBucket)
	claimsSvc := claims.NewService(pool, notificationSvc)
	moderationSvc := moderation.NewService(pool, reviewsRepo, targetsRepo, notificationSvc)
	adminSvc := admin.NewService(pool, usersRepo)

	mux := http.NewServeMux()
	auth.NewHandler(authSvc, loginLimiter, otpLimiter, cfg.TrustProxyHeaders).Routes(mux)
	users.NewHandler(usersRepo).Routes(mux)
	notifications.NewHandler(notificationSvc).Routes(mux)
	privacy.NewHandler(privacySvc).Routes(mux)
	categories.NewHandler(categoriesRepo).Routes(mux)
	locations.NewHandler(locationsRepo).Routes(mux)
	businesses.NewHandler(bizRepo).Routes(mux)
	targets.NewHandler(targetsRepo, bizRepo, usersRepo).Routes(mux)
	reviews.NewHandler(reviewsRepo, reviewLimiter).Routes(mux)
	ratings.NewHandler(ratingsRepo).Routes(mux)
	search.NewHandler(searchRepo, searchLimiter, cfg.TrustProxyHeaders).Routes(mux)
	media.NewHandler(mediaSvc, presignLimiter).Routes(mux)
	claims.NewHandler(claimsSvc).Routes(mux)
	moderation.NewHandler(moderationSvc, mediaSvc, reportLimiter).Routes(mux)
	admin.NewHandler(adminSvc).Routes(mux)
	registerDocs(mux)

	// Operational endpoints (no auth; restrict /metrics at the network layer
	// in production deployments).
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		web.Respond(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		pingCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(pingCtx); err != nil {
			web.RespondError(w, r, &web.Error{Status: http.StatusServiceUnavailable,
				Code: "not_ready", Message: "database unavailable", Internal: err})
			return
		}
		web.Respond(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	mux.Handle("GET /metrics", web.MetricsHandler())

	return web.Chain(mux,
		web.Recover,
		web.RequestID,
		web.Metrics,
		web.Logging,
		web.SecureHeaders,
		web.CORS(cfg.CORSAllowedOrigins),
		timeoutMiddleware(cfg.DatabaseQueryTimeout),
		web.Authenticate(authSvc.VerifyAccess),
		web.PublicCache,
	)
}

// timeoutMiddleware bounds each request's context so downstream database
// calls inherit a deadline.
func timeoutMiddleware(d time.Duration) web.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
