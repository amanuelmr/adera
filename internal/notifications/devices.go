package notifications

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/adera-platform/backend/internal/platform/web"
)

// Push platforms a client may register.
const (
	PlatformAndroid = "android"
	PlatformIOS     = "ios"
	PlatformWeb     = "web"
)

// maxDevicesPerUser bounds how many installs one account can register, so a
// misbehaving client cannot grow the table without limit. The oldest
// registration is dropped once the cap is reached.
const maxDevicesPerUser = 10

// DeviceToken describes one registered install. The token itself is never
// returned: it is a send credential, and echoing it would let any reader of a
// response push to that device.
type DeviceToken struct {
	ID         uuid.UUID `json:"id"`
	Platform   string    `json:"platform"`
	AppVersion string    `json:"app_version,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
}

// ValidPlatform reports whether p is a supported push platform.
func ValidPlatform(p string) bool {
	return p == PlatformAndroid || p == PlatformIOS || p == PlatformWeb
}

// RegisterDevice records or refreshes a push token for the user, tied to the
// sign-in (session family) it was sent under. Re-posting the same token is
// the normal case — clients send it on every launch — so the operation is an
// idempotent upsert that also reassigns ownership when the token belonged to
// a different account.
func (s *Service) RegisterDevice(ctx context.Context, userID, sessionID uuid.UUID, token, platform, appVersion string) (DeviceToken, error) {
	if !ValidPlatform(platform) {
		return DeviceToken{}, web.ErrValidation("invalid device").
			WithDetail("platform", "must be android, ios, or web")
	}
	if n := len(token); n < 32 || n > 4096 {
		return DeviceToken{}, web.ErrValidation("invalid device").
			WithDetail("token", "must be between 32 and 4096 characters")
	}
	if len(appVersion) > 40 {
		return DeviceToken{}, web.ErrValidation("invalid device").
			WithDetail("app_version", "must be at most 40 characters")
	}

	var d DeviceToken
	err := s.pool.QueryRow(ctx, `
		INSERT INTO device_tokens (id, user_id, token, platform, app_version, session_family_id)
		VALUES ($1, $2, $3, $4, $5,
			(SELECT family_id FROM auth_sessions WHERE id = $6 AND user_id = $2))
		ON CONFLICT (token) DO UPDATE
		   SET user_id = excluded.user_id,
		       platform = excluded.platform,
		       app_version = excluded.app_version,
		       session_family_id = excluded.session_family_id,
		       last_seen_at = now()
		RETURNING id, platform, app_version, created_at, last_seen_at`,
		uuid.New(), userID, token, platform, appVersion, sessionID,
	).Scan(&d.ID, &d.Platform, &d.AppVersion, &d.CreatedAt, &d.LastSeenAt)
	if err != nil {
		return DeviceToken{}, fmt.Errorf("registering device token: %w", err)
	}
	if err := s.pruneDeviceTokens(ctx, userID); err != nil {
		return DeviceToken{}, err
	}
	d.CreatedAt, d.LastSeenAt = d.CreatedAt.UTC(), d.LastSeenAt.UTC()
	return d, nil
}

// pruneDeviceTokens keeps only the most recently seen registrations.
func (s *Service) pruneDeviceTokens(ctx context.Context, userID uuid.UUID) error {
	if _, err := s.pool.Exec(ctx, `
		DELETE FROM device_tokens
		WHERE user_id = $1 AND id NOT IN (
			SELECT id FROM device_tokens
			WHERE user_id = $1
			ORDER BY last_seen_at DESC, id
			LIMIT $2
		)`, userID, maxDevicesPerUser); err != nil {
		return fmt.Errorf("pruning device tokens: %w", err)
	}
	return nil
}

// UnregisterDevice removes one of the caller's push tokens, which is what a
// client does on logout.
func (s *Service) UnregisterDevice(ctx context.Context, userID uuid.UUID, token string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM device_tokens WHERE user_id = $1 AND token = $2`, userID, token)
	if err != nil {
		return fmt.Errorf("unregistering device token: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return web.ErrNotFound("device token")
	}
	return nil
}

// ListDevices returns the caller's registered installs, most recent first.
func (s *Service) ListDevices(ctx context.Context, userID uuid.UUID) ([]DeviceToken, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, platform, app_version, created_at, last_seen_at
		FROM device_tokens WHERE user_id = $1
		ORDER BY last_seen_at DESC, id`, userID)
	if err != nil {
		return nil, fmt.Errorf("listing device tokens: %w", err)
	}
	defer rows.Close()
	var out []DeviceToken
	for rows.Next() {
		var d DeviceToken
		if err := rows.Scan(&d.ID, &d.Platform, &d.AppVersion, &d.CreatedAt, &d.LastSeenAt); err != nil {
			return nil, fmt.Errorf("scanning device token: %w", err)
		}
		d.CreatedAt, d.LastSeenAt = d.CreatedAt.UTC(), d.LastSeenAt.UTC()
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating device tokens: %w", err)
	}
	return out, nil
}

// DeviceTokensFor returns the raw push tokens for a user, for a delivery
// provider to send to: only for an active account, and only tokens whose
// sign-in is still live, so a phone signed out while offline (and so never
// able to unregister) stops receiving.
func (s *Service) DeviceTokensFor(ctx context.Context, userID uuid.UUID) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT d.token FROM device_tokens d JOIN users u ON u.id = d.user_id
		WHERE d.user_id = $1 AND u.status = 'active'
		  AND (d.session_family_id IS NULL OR EXISTS (
			SELECT 1 FROM auth_sessions s
			WHERE s.family_id = d.session_family_id AND s.revoked_at IS NULL
			  AND s.replaced_by_id IS NULL AND s.expires_at > now()))
		ORDER BY d.last_seen_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("loading device tokens: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			return nil, fmt.Errorf("scanning device token: %w", err)
		}
		out = append(out, token)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating device tokens: %w", err)
	}
	return out, nil
}

// DeleteDeviceTokens removes tokens the push service has rejected as no
// longer valid, so dead installs stop being retried forever.
func (s *Service) DeleteDeviceTokens(ctx context.Context, tokens []string) error {
	if len(tokens) == 0 {
		return nil
	}
	if _, err := s.pool.Exec(ctx, `DELETE FROM device_tokens WHERE token = ANY($1)`, tokens); err != nil {
		return fmt.Errorf("deleting stale device tokens: %w", err)
	}
	return nil
}
