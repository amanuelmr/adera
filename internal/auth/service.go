package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adera-platform/backend/internal/platform/database"
	"github.com/adera-platform/backend/internal/platform/security"
	"github.com/adera-platform/backend/internal/platform/web"
	"github.com/adera-platform/backend/internal/users"
)

const (
	otpLength     = 6
	otpTTL        = 15 * time.Minute
	otpMaxAttempt = 5
)

// Service implements the authentication domain.
type Service struct {
	pool       *pgxpool.Pool
	usersRepo  *users.Repo
	hasher     *security.Hasher
	tokens     *security.TokenManager
	provider   Provider
	refreshTTL time.Duration
}

func NewService(pool *pgxpool.Pool, usersRepo *users.Repo, hasher *security.Hasher,
	tokens *security.TokenManager, provider Provider, refreshTTL time.Duration) *Service {
	return &Service{
		pool: pool, usersRepo: usersRepo, hasher: hasher,
		tokens: tokens, provider: provider, refreshTTL: refreshTTL,
	}
}

// TokenPair is returned by register/login/refresh.
type TokenPair struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	ExpiresInSeconds int    `json:"expires_in"`
}

// VerifyAccess adapts token verification for the HTTP middleware.
func (s *Service) VerifyAccess(_ context.Context, token string) (web.Principal, error) {
	userID, sessionID, roles, err := s.tokens.Verify(token)
	if err != nil {
		return web.Principal{}, err
	}
	return web.Principal{UserID: userID, SessionID: sessionID, Roles: roles}, nil
}

// RegisterInput is the validated registration payload.
type RegisterInput struct {
	DisplayName string
	Email       string
	Phone       string // any accepted format; normalized here
	Password    string
	Language    string
}

// Register creates the account, sends a verification code when a provider is
// available, and returns an authenticated session.
func (s *Service) Register(ctx context.Context, in RegisterInput, deviceInfo string) (users.User, TokenPair, error) {
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	if err := users.ValidateDisplayName(in.DisplayName); err != nil {
		return users.User{}, TokenPair{}, err
	}
	if err := users.ValidatePassword(in.Password); err != nil {
		return users.User{}, TokenPair{}, err
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if in.Email != "" && !users.ValidEmail(in.Email) {
		return users.User{}, TokenPair{}, web.ErrValidation("invalid registration").WithDetail("email", "not a valid email address")
	}
	if in.Phone != "" {
		normalized := users.NormalizePhone(in.Phone)
		if normalized == "" {
			return users.User{}, TokenPair{}, web.ErrValidation("invalid registration").WithDetail("phone", "not a valid phone number")
		}
		in.Phone = normalized
	}
	if in.Email == "" && in.Phone == "" {
		return users.User{}, TokenPair{}, web.ErrValidation("invalid registration").WithDetail("identifier", "email or phone is required")
	}
	if in.Language == "" {
		in.Language = "en"
	}
	if !users.ValidLanguage(in.Language) {
		return users.User{}, TokenPair{}, web.ErrValidation("invalid registration").WithDetail("preferred_language", "unsupported language")
	}

	hash, err := s.hasher.Hash(in.Password)
	if err != nil {
		return users.User{}, TokenPair{}, fmt.Errorf("hashing password: %w", err)
	}
	u := users.User{
		ID:                uuid.New(),
		DisplayName:       in.DisplayName,
		Email:             in.Email,
		Phone:             in.Phone,
		PasswordHash:      hash,
		Status:            users.StatusActive,
		PreferredLanguage: in.Language,
	}
	if err := s.usersRepo.Create(ctx, u); err != nil {
		if errors.Is(err, users.ErrDuplicate) {
			// Deliberately vague: do not confirm which identifier exists.
			return users.User{}, TokenPair{}, web.ErrConflict("an account with this email or phone may already exist")
		}
		return users.User{}, TokenPair{}, err
	}

	// Verification code delivery is best-effort at registration; the user can
	// re-request from the verify endpoint.
	channel, destination := "email", u.Email
	if destination == "" {
		channel, destination = "phone", u.Phone
	}
	if err := s.sendVerificationCode(ctx, u.ID, channel, destination); err != nil && !errors.Is(err, ErrProviderUnavailable) {
		return users.User{}, TokenPair{}, err
	}

	pair, err := s.issuePair(ctx, u.ID, deviceInfo)
	if err != nil {
		return users.User{}, TokenPair{}, err
	}
	return u, pair, nil
}

// Login authenticates by email/phone + password. Errors are uniform to
// prevent user enumeration; a deactivated account is reactivated on
// successful login (documented retention behavior).
func (s *Service) Login(ctx context.Context, identifier, password, deviceInfo string) (users.User, TokenPair, error) {
	identifier = strings.TrimSpace(identifier)
	if p := users.NormalizePhone(identifier); p != "" {
		identifier = p
	} else {
		identifier = strings.ToLower(identifier)
	}

	u, err := s.usersRepo.GetByIdentifier(ctx, identifier)
	if err != nil {
		if errors.Is(err, users.ErrNotFound) {
			// Burn comparable time so absent users are indistinguishable.
			_, _ = s.hasher.Hash(password)
			return users.User{}, TokenPair{}, web.ErrInvalidCredentials()
		}
		return users.User{}, TokenPair{}, err
	}
	if u.PasswordHash == "" {
		return users.User{}, TokenPair{}, web.ErrInvalidCredentials()
	}
	if err := s.hasher.Verify(password, u.PasswordHash); err != nil {
		if errors.Is(err, security.ErrHashMismatch) {
			return users.User{}, TokenPair{}, web.ErrInvalidCredentials()
		}
		return users.User{}, TokenPair{}, fmt.Errorf("verifying password: %w", err)
	}
	switch u.Status {
	case users.StatusSuspended:
		return users.User{}, TokenPair{}, &web.Error{
			Status: http.StatusForbidden, Code: web.CodeAccountSuspended, Message: "this account is suspended",
		}
	case users.StatusDeactivated:
		if err := s.usersRepo.SetStatus(ctx, u.ID, users.StatusActive); err != nil {
			return users.User{}, TokenPair{}, err
		}
		u.Status = users.StatusActive
	}

	pair, err := s.issuePair(ctx, u.ID, deviceInfo)
	if err != nil {
		return users.User{}, TokenPair{}, err
	}
	return u, pair, nil
}

// issuePair creates a new session family and signs an access token.
func (s *Service) issuePair(ctx context.Context, userID uuid.UUID, deviceInfo string) (TokenPair, error) {
	return s.insertSession(ctx, userID, uuid.New(), deviceInfo)
}

func (s *Service) insertSession(ctx context.Context, userID, familyID uuid.UUID, deviceInfo string) (TokenPair, error) {
	refresh, err := security.RandomToken()
	if err != nil {
		return TokenPair{}, err
	}
	sessionID := uuid.New()
	now := time.Now().UTC()
	if len(deviceInfo) > 300 {
		deviceInfo = deviceInfo[:300]
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO auth_sessions (id, user_id, family_id, refresh_token_hash, device_info, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		sessionID, userID, familyID, security.HashToken(refresh), deviceInfo, now.Add(s.refreshTTL))
	if err != nil {
		return TokenPair{}, fmt.Errorf("inserting session: %w", err)
	}
	roles, err := s.usersRepo.Roles(ctx, userID)
	if err != nil {
		return TokenPair{}, err
	}
	access, err := s.tokens.Sign(userID, sessionID, roles, now)
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{
		AccessToken:      access,
		RefreshToken:     refresh,
		TokenType:        "Bearer",
		ExpiresInSeconds: int(s.tokens.AccessTokenTTL().Seconds()),
	}, nil
}

// Refresh rotates a refresh token. Presenting an already-rotated or revoked
// token is treated as theft: the entire token family is revoked.
func (s *Service) Refresh(ctx context.Context, refreshToken, deviceInfo string) (TokenPair, error) {
	hash := security.HashToken(refreshToken)
	var (
		pair       TokenPair
		reuse      bool
		unauthorized = web.ErrUnauthorized("invalid or expired refresh token")
	)
	err := database.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		var (
			id, userID, familyID uuid.UUID
			expiresAt            time.Time
			revokedAt            *time.Time
			replacedBy           *uuid.UUID
			oldDevice            string
			userStatus           string
		)
		err := tx.QueryRow(ctx, `
			SELECT s.id, s.user_id, s.family_id, s.expires_at, s.revoked_at, s.replaced_by_id, s.device_info, u.status
			FROM auth_sessions s JOIN users u ON u.id = s.user_id
			WHERE s.refresh_token_hash = $1
			FOR UPDATE OF s`, hash).
			Scan(&id, &userID, &familyID, &expiresAt, &revokedAt, &replacedBy, &oldDevice, &userStatus)
		if errors.Is(err, pgx.ErrNoRows) {
			return unauthorized
		}
		if err != nil {
			return fmt.Errorf("loading session: %w", err)
		}

		if revokedAt != nil || replacedBy != nil {
			// Token reuse detected: revoke the whole family. Return nil so
			// the transaction COMMITS the revocation; the caller converts
			// the reuse flag into an authentication error.
			reuse = true
			if _, err := tx.Exec(ctx, `
				UPDATE auth_sessions
				SET revoked_at = now(), revoked_reason = 'refresh_token_reuse'
				WHERE family_id = $1 AND revoked_at IS NULL`, familyID); err != nil {
				return fmt.Errorf("revoking token family: %w", err)
			}
			return nil
		}
		if time.Now().After(expiresAt) || userStatus != users.StatusActive {
			return unauthorized
		}

		newRefresh, err := security.RandomToken()
		if err != nil {
			return err
		}
		newID := uuid.New()
		now := time.Now().UTC()
		if deviceInfo == "" {
			deviceInfo = oldDevice
		}
		if len(deviceInfo) > 300 {
			deviceInfo = deviceInfo[:300]
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO auth_sessions (id, user_id, family_id, refresh_token_hash, device_info, expires_at)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			newID, userID, familyID, security.HashToken(newRefresh), deviceInfo, now.Add(s.refreshTTL)); err != nil {
			return fmt.Errorf("inserting rotated session: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE auth_sessions SET replaced_by_id = $2, last_used_at = now() WHERE id = $1`,
			id, newID); err != nil {
			return fmt.Errorf("linking rotated session: %w", err)
		}

		var roles []string
		rows, err := tx.Query(ctx, `SELECT role FROM user_roles WHERE user_id = $1 ORDER BY role`, userID)
		if err != nil {
			return fmt.Errorf("querying roles: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var role string
			if err := rows.Scan(&role); err != nil {
				return fmt.Errorf("scanning role: %w", err)
			}
			roles = append(roles, role)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterating roles: %w", err)
		}

		access, err := s.tokens.Sign(userID, newID, roles, now)
		if err != nil {
			return err
		}
		pair = TokenPair{
			AccessToken:      access,
			RefreshToken:     newRefresh,
			TokenType:        "Bearer",
			ExpiresInSeconds: int(s.tokens.AccessTokenTTL().Seconds()),
		}
		return nil
	})
	if reuse {
		// The family is already revoked inside the transaction; surface a
		// uniform error.
		return TokenPair{}, unauthorized
	}
	if err != nil {
		return TokenPair{}, err
	}
	return pair, nil
}

// Logout revokes the family of the calling session.
func (s *Service) Logout(ctx context.Context, sessionID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE auth_sessions SET revoked_at = now(), revoked_reason = 'logout'
		WHERE family_id = (SELECT family_id FROM auth_sessions WHERE id = $1)
		  AND revoked_at IS NULL`, sessionID)
	if err != nil {
		return fmt.Errorf("revoking session family: %w", err)
	}
	return nil
}

// LogoutAll revokes every session of the user.
func (s *Service) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE auth_sessions SET revoked_at = now(), revoked_reason = 'logout_all'
		WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	if err != nil {
		return fmt.Errorf("revoking all sessions: %w", err)
	}
	return nil
}

// Session is one active session (family head) for display.
type Session struct {
	ID         uuid.UUID `json:"id"`
	DeviceInfo string    `json:"device_info"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	Current    bool      `json:"current"`
}

// ListSessions returns the user's active sessions (the newest token of each
// unexpired, unrevoked family).
func (s *Service) ListSessions(ctx context.Context, userID, currentSessionID uuid.UUID) ([]Session, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, device_info, created_at, last_used_at, expires_at, family_id
		FROM auth_sessions
		WHERE user_id = $1 AND revoked_at IS NULL AND replaced_by_id IS NULL AND expires_at > now()
		ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("querying sessions: %w", err)
	}
	defer rows.Close()

	var currentFamily uuid.UUID
	type rowT struct {
		s      Session
		family uuid.UUID
	}
	var items []rowT
	for rows.Next() {
		var it rowT
		if err := rows.Scan(&it.s.ID, &it.s.DeviceInfo, &it.s.CreatedAt, &it.s.LastUsedAt, &it.s.ExpiresAt, &it.family); err != nil {
			return nil, fmt.Errorf("scanning session: %w", err)
		}
		if it.s.ID == currentSessionID {
			currentFamily = it.family
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating sessions: %w", err)
	}
	sessions := make([]Session, 0, len(items))
	for _, it := range items {
		it.s.Current = it.family == currentFamily && currentFamily != uuid.Nil
		it.s.CreatedAt = it.s.CreatedAt.UTC()
		it.s.LastUsedAt = it.s.LastUsedAt.UTC()
		it.s.ExpiresAt = it.s.ExpiresAt.UTC()
		sessions = append(sessions, it.s)
	}
	return sessions, nil
}

// RevokeSession revokes one session family owned by the user (object-level
// authorization: the WHERE clause binds the session to the caller).
func (s *Service) RevokeSession(ctx context.Context, userID, sessionID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE auth_sessions SET revoked_at = now(), revoked_reason = 'user_revoked'
		WHERE family_id = (SELECT family_id FROM auth_sessions WHERE id = $1 AND user_id = $2)
		  AND user_id = $2 AND revoked_at IS NULL`, sessionID, userID)
	if err != nil {
		return fmt.Errorf("revoking session: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return web.ErrNotFound("session")
	}
	return nil
}

// sendVerificationCode creates and delivers an OTP for the channel.
func (s *Service) sendVerificationCode(ctx context.Context, userID uuid.UUID, channel, destination string) error {
	purpose := channel + "_verify"
	code, err := security.RandomDigits(otpLength)
	if err != nil {
		return err
	}
	// Invalidate previous outstanding codes for this purpose.
	if _, err := s.pool.Exec(ctx, `
		UPDATE verification_tokens SET consumed_at = now()
		WHERE user_id = $1 AND purpose = $2 AND consumed_at IS NULL`, userID, purpose); err != nil {
		return fmt.Errorf("invalidating previous codes: %w", err)
	}
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO verification_tokens (id, user_id, purpose, code_hash, expires_at)
		VALUES ($1, $2, $3, $4, $5)`,
		uuid.New(), userID, purpose, security.HashToken(code), time.Now().UTC().Add(otpTTL)); err != nil {
		return fmt.Errorf("storing verification code: %w", err)
	}
	if err := s.provider.SendCode(ctx, channel, destination, purpose, code); err != nil {
		return err
	}
	return nil
}

// RequestVerification sends a code to the user's own email or phone.
func (s *Service) RequestVerification(ctx context.Context, userID uuid.UUID, channel string) error {
	u, err := s.usersRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	var destination string
	switch channel {
	case "email":
		destination = u.Email
	case "phone":
		destination = u.Phone
	default:
		return web.ErrValidation("invalid request").WithDetail("channel", "must be email or phone")
	}
	if destination == "" {
		return web.ErrValidation("invalid request").WithDetail("channel", "no "+channel+" on this account")
	}
	if err := s.sendVerificationCode(ctx, userID, channel, destination); err != nil {
		if errors.Is(err, ErrProviderUnavailable) {
			return &web.Error{Status: http.StatusServiceUnavailable, Code: "verification_unavailable",
				Message: "verification delivery is not configured yet"}
		}
		return err
	}
	return nil
}

// consumeCode validates an OTP for the purpose, enforcing expiry and attempt
// limits, and consumes it on success.
func (s *Service) consumeCode(ctx context.Context, userID uuid.UUID, purpose, code string) error {
	invalid := web.ErrValidation("invalid or expired code").WithDetail("code", "invalid or expired")
	return database.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		var (
			id       uuid.UUID
			codeHash []byte
			attempts int
		)
		err := tx.QueryRow(ctx, `
			SELECT id, code_hash, attempts FROM verification_tokens
			WHERE user_id = $1 AND purpose = $2 AND consumed_at IS NULL AND expires_at > now()
			ORDER BY created_at DESC LIMIT 1
			FOR UPDATE`, userID, purpose).Scan(&id, &codeHash, &attempts)
		if errors.Is(err, pgx.ErrNoRows) {
			return invalid
		}
		if err != nil {
			return fmt.Errorf("loading verification code: %w", err)
		}
		if attempts >= otpMaxAttempt {
			return invalid
		}
		if !security.ConstantTimeEqual(security.HashToken(code), codeHash) {
			if _, err := tx.Exec(ctx, `UPDATE verification_tokens SET attempts = attempts + 1 WHERE id = $1`, id); err != nil {
				return fmt.Errorf("recording failed attempt: %w", err)
			}
			return invalid
		}
		if _, err := tx.Exec(ctx, `UPDATE verification_tokens SET consumed_at = now() WHERE id = $1`, id); err != nil {
			return fmt.Errorf("consuming code: %w", err)
		}
		return nil
	})
}

// ConfirmVerification validates the OTP and marks the channel verified.
func (s *Service) ConfirmVerification(ctx context.Context, userID uuid.UUID, channel, code string) error {
	if channel != "email" && channel != "phone" {
		return web.ErrValidation("invalid request").WithDetail("channel", "must be email or phone")
	}
	if err := s.consumeCode(ctx, userID, channel+"_verify", code); err != nil {
		return err
	}
	return s.usersRepo.MarkVerified(ctx, userID, channel, time.Now().UTC())
}

// RequestPasswordReset always succeeds from the caller's perspective (no user
// enumeration); a code is sent only when the account exists.
func (s *Service) RequestPasswordReset(ctx context.Context, identifier string) error {
	identifier = strings.TrimSpace(identifier)
	if p := users.NormalizePhone(identifier); p != "" {
		identifier = p
	} else {
		identifier = strings.ToLower(identifier)
	}
	u, err := s.usersRepo.GetByIdentifier(ctx, identifier)
	if err != nil {
		if errors.Is(err, users.ErrNotFound) {
			return nil
		}
		return err
	}
	channel, destination := "email", u.Email
	if identifier == u.Phone {
		channel, destination = "phone", u.Phone
	}
	code, err := security.RandomDigits(otpLength)
	if err != nil {
		return err
	}
	if _, err := s.pool.Exec(ctx, `
		UPDATE verification_tokens SET consumed_at = now()
		WHERE user_id = $1 AND purpose = 'password_reset' AND consumed_at IS NULL`, u.ID); err != nil {
		return fmt.Errorf("invalidating previous reset codes: %w", err)
	}
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO verification_tokens (id, user_id, purpose, code_hash, expires_at)
		VALUES ($1, $2, 'password_reset', $3, $4)`,
		uuid.New(), u.ID, security.HashToken(code), time.Now().UTC().Add(otpTTL)); err != nil {
		return fmt.Errorf("storing reset code: %w", err)
	}
	if err := s.provider.SendCode(ctx, channel, destination, "password_reset", code); err != nil {
		if errors.Is(err, ErrProviderUnavailable) {
			// Still 200: do not reveal whether the account exists. The code
			// simply cannot be delivered until a provider is configured.
			return nil
		}
		return err
	}
	return nil
}

// ConfirmPasswordReset sets a new password and revokes all sessions.
func (s *Service) ConfirmPasswordReset(ctx context.Context, identifier, code, newPassword string) error {
	if err := users.ValidatePassword(newPassword); err != nil {
		return err
	}
	identifier = strings.TrimSpace(identifier)
	if p := users.NormalizePhone(identifier); p != "" {
		identifier = p
	} else {
		identifier = strings.ToLower(identifier)
	}
	u, err := s.usersRepo.GetByIdentifier(ctx, identifier)
	if err != nil {
		if errors.Is(err, users.ErrNotFound) {
			return web.ErrValidation("invalid or expired code").WithDetail("code", "invalid or expired")
		}
		return err
	}
	if err := s.consumeCode(ctx, u.ID, "password_reset", code); err != nil {
		return err
	}
	hash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}
	if err := s.usersRepo.UpdatePassword(ctx, u.ID, hash); err != nil {
		return err
	}
	return s.LogoutAll(ctx, u.ID)
}

// ChangePassword verifies the current password, sets the new one, and revokes
// every other session family.
func (s *Service) ChangePassword(ctx context.Context, userID, currentSessionID uuid.UUID, current, newPassword string) error {
	if err := users.ValidatePassword(newPassword); err != nil {
		return err
	}
	u, err := s.usersRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if u.PasswordHash == "" {
		return web.ErrInvalidCredentials()
	}
	if err := s.hasher.Verify(current, u.PasswordHash); err != nil {
		if errors.Is(err, security.ErrHashMismatch) {
			return web.ErrInvalidCredentials()
		}
		return fmt.Errorf("verifying password: %w", err)
	}
	hash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}
	if err := s.usersRepo.UpdatePassword(ctx, userID, hash); err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		UPDATE auth_sessions SET revoked_at = now(), revoked_reason = 'password_changed'
		WHERE user_id = $1 AND revoked_at IS NULL
		  AND family_id <> (SELECT family_id FROM auth_sessions WHERE id = $2)`,
		userID, currentSessionID)
	if err != nil {
		return fmt.Errorf("revoking other sessions: %w", err)
	}
	return nil
}

// Deactivate sets the account to deactivated and revokes all sessions.
// Reviews remain published under the retention policy documented in
// docs/moderation-policy.md; logging in again reactivates the account.
func (s *Service) Deactivate(ctx context.Context, userID uuid.UUID) error {
	if err := s.usersRepo.SetStatus(ctx, userID, users.StatusDeactivated); err != nil {
		return err
	}
	return s.LogoutAll(ctx, userID)
}
