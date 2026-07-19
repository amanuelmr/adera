// Package auth implements authentication: registration, login, rotating
// refresh-token sessions with reuse detection, contact verification, and
// password reset.
package auth

import (
	"context"
	"fmt"
	"io"
)

// Provider delivers one-time codes for contact verification and password
// reset. Implementations must never log or store the raw code.
type Provider interface {
	// SendCode delivers code to destination (an email address or E.164 phone).
	// channel is "email" or "phone"; purpose is the verification purpose.
	SendCode(ctx context.Context, channel, destination, purpose, code string) error
}

// ConsoleProvider writes codes to a writer (stdout) for local development.
// It must never be constructed in production; cmd/api enforces that. There is
// deliberately no fake "SMS sent" behavior — the console IS the delivery
// channel in development.
type ConsoleProvider struct {
	W io.Writer
}

func (p ConsoleProvider) SendCode(_ context.Context, channel, destination, purpose, code string) error {
	_, err := fmt.Fprintf(p.W, "[DEV VERIFICATION] channel=%s destination=%s purpose=%s code=%s\n",
		channel, destination, purpose, code)
	if err != nil {
		return fmt.Errorf("writing dev code: %w", err)
	}
	return nil
}

// NoopProvider is used when no real provider is configured (e.g., production
// before an SMS/email integration exists). Sends fail loudly so the API can
// report verification as unavailable instead of pretending it happened.
type NoopProvider struct{}

// ErrProviderUnavailable signals that no delivery integration is configured.
var ErrProviderUnavailable = fmt.Errorf("verification provider not configured")

func (NoopProvider) SendCode(context.Context, string, string, string, string) error {
	return ErrProviderUnavailable
}
