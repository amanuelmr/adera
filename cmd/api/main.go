// Command api is the Adera backend service.
//
// Usage:
//
//	api serve     start the HTTP API (default)
//	api migrate   apply pending database migrations and exit
//	api seed      apply migrations, then load development seed data and exit
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adera-platform/backend/internal/app"
	"github.com/adera-platform/backend/internal/auth"
	"github.com/adera-platform/backend/internal/notifications"
	"github.com/adera-platform/backend/internal/platform/config"
	"github.com/adera-platform/backend/internal/platform/database"
	"github.com/adera-platform/backend/internal/platform/logging"
	"github.com/adera-platform/backend/internal/platform/storage"
	"github.com/adera-platform/backend/migrations"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "error", err.Error())
		os.Exit(1)
	}
}

func run() error {
	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading configuration: %w", err)
	}
	logging.Setup(cfg.LogLevel, cfg.LogFormat)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := database.Connect(ctx, cfg.DatabaseURL, cfg.DatabaseMaxConns)
	if err != nil {
		return err
	}
	defer pool.Close()

	switch cmd {
	case "migrate":
		return database.Migrate(ctx, pool, migrations.FS)
	case "seed":
		if err := database.Migrate(ctx, pool, migrations.FS); err != nil {
			return err
		}
		return seed(ctx, cfg, pool)
	case "serve":
		return serve(ctx, cfg, pool)
	default:
		return fmt.Errorf("unknown command %q (want serve, migrate, or seed)", cmd)
	}
}

// verificationProvider selects how one-time codes reach users: SMTP when a
// relay is configured, the console in development, and otherwise a provider
// that reports unavailable so endpoints return 503 instead of silently
// dropping codes.
//
// Only the email channel has a real delivery path; phone verification stays
// unavailable until an SMS or Telegram provider is added.
func verificationProvider(cfg config.Config) (auth.Provider, error) {
	if !cfg.SMTPEnabled() {
		if cfg.IsDev() {
			slog.Info("using console verification provider (development only)")
			return auth.ConsoleProvider{W: os.Stdout}, nil
		}
		slog.Warn("no verification provider configured; verification endpoints will return 503")
		return auth.NoopProvider{}, nil
	}

	opts := auth.SMTPOptions{
		Host:        cfg.SMTPHost,
		Port:        cfg.SMTPPort,
		Username:    cfg.SMTPUsername,
		Password:    cfg.SMTPPassword,
		FromAddress: cfg.SMTPFromAddress,
		FromName:    cfg.SMTPFromName,
		Timeout:     cfg.SMTPTimeout,
	}
	switch cfg.SMTPTLS {
	case config.SMTPTLSImplicit:
		opts.ImplicitTLS = true
	case config.SMTPTLSNone:
		opts.AllowPlaintext = true
	}

	provider, err := auth.NewEmailProvider(opts)
	if err != nil {
		return nil, fmt.Errorf("configuring smtp verification provider: %w", err)
	}
	slog.Info("using smtp verification provider",
		"host", cfg.SMTPHost, "port", cfg.SMTPPort, "tls", cfg.SMTPTLS, "from", cfg.SMTPFromAddress)
	return provider, nil
}

func serve(ctx context.Context, cfg config.Config, pool *pgxpool.Pool) error {
	var store storage.Store
	if cfg.StorageEnabled {
		minioStore, err := storage.NewMinio(cfg.StorageEndpoint, cfg.StorageAccessKey, cfg.StorageSecretKey,
			cfg.StorageUseSSL, cfg.StoragePublicBucket, cfg.StoragePrivateBucket)
		if err != nil {
			return err
		}
		if cfg.IsDev() {
			if err := minioStore.EnsureBuckets(ctx); err != nil {
				return fmt.Errorf("ensuring buckets: %w", err)
			}
		}
		store = minioStore
	} else {
		slog.Warn("object storage disabled; media endpoints will return 503")
	}

	provider, err := verificationProvider(cfg)
	if err != nil {
		return err
	}

	var notifyProvider notifications.Provider
	if cfg.IsDev() {
		notifyProvider = notifications.ConsoleProvider{W: os.Stdout}
		slog.Info("using console notification delivery provider (development only)")
	} else {
		// No real SMS/email integration is configured yet; events accumulate
		// in the outbox for later replay rather than being dropped.
		notifyProvider = notifications.NoopProvider{}
		slog.Warn("no notification delivery provider configured; outbox events will not be delivered")
	}
	dispatcher := notifications.NewDispatcher(notifications.NewService(pool), notifyProvider)
	go dispatcher.Run(ctx)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           app.BuildAPI(cfg, pool, store, provider),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      35 * time.Second, // > request context timeout
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("http server listening", "addr", cfg.HTTPAddr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
	}

	slog.Info("shutting down gracefully")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	return nil
}
