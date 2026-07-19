package users

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when no user matches.
var ErrNotFound = errors.New("user not found")

// ErrDuplicate is returned when email or phone is already registered.
var ErrDuplicate = errors.New("email or phone already registered")

// Repo provides user persistence. All methods use parameterized SQL.
type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

const userColumns = `id, display_name, coalesce(email, ''), coalesce(phone, ''), coalesce(password_hash, ''),
	status, email_verified_at, phone_verified_at, preferred_language, created_at, updated_at`

func scanUser(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.DisplayName, &u.Email, &u.Phone, &u.PasswordHash,
		&u.Status, &u.EmailVerifiedAt, &u.PhoneVerifiedAt, &u.PreferredLanguage, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("scanning user: %w", err)
	}
	return u, nil
}

// Create inserts a new user and grants the customer role.
func (r *Repo) Create(ctx context.Context, u User) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO users (id, display_name, email, phone, password_hash, status, preferred_language)
		VALUES ($1, $2, nullif($3, ''), nullif($4, ''), nullif($5, ''), $6, $7)`,
		u.ID, u.DisplayName, u.Email, u.Phone, u.PasswordHash, u.Status, u.PreferredLanguage)
	if isUniqueViolation(err) {
		return ErrDuplicate
	}
	if err != nil {
		return fmt.Errorf("inserting user: %w", err)
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO user_roles (user_id, role) VALUES ($1, 'customer')
		ON CONFLICT DO NOTHING`, u.ID)
	if err != nil {
		return fmt.Errorf("granting customer role: %w", err)
	}
	return nil
}

func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (User, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id)
	return scanUser(row)
}

// GetByIdentifier looks up by email (case-insensitive) or E.164 phone.
func (r *Repo) GetByIdentifier(ctx context.Context, identifier string) (User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT `+userColumns+` FROM users WHERE email = $1 OR phone = $1`, identifier)
	return scanUser(row)
}

// Roles returns the user's role names.
func (r *Repo) Roles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT role FROM user_roles WHERE user_id = $1 ORDER BY role`, userID)
	if err != nil {
		return nil, fmt.Errorf("querying roles: %w", err)
	}
	defer rows.Close()
	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, fmt.Errorf("scanning role: %w", err)
		}
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating roles: %w", err)
	}
	return roles, nil
}

// GrantRole grants a role idempotently.
func (r *Repo) GrantRole(ctx context.Context, userID uuid.UUID, role string, grantedBy uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_roles (user_id, role, granted_by) VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING`, userID, role, grantedBy)
	if err != nil {
		return fmt.Errorf("granting role: %w", err)
	}
	return nil
}

// UpdateProfile applies an explicit field allowlist (display name, language).
func (r *Repo) UpdateProfile(ctx context.Context, id uuid.UUID, displayName, language string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE users SET display_name = $2, preferred_language = $3 WHERE id = $1`,
		id, displayName, language)
	if err != nil {
		return fmt.Errorf("updating profile: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdatePassword replaces the password hash.
func (r *Repo) UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE users SET password_hash = $2 WHERE id = $1`, id, hash)
	if err != nil {
		return fmt.Errorf("updating password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// MarkVerified records a successful email or phone verification.
func (r *Repo) MarkVerified(ctx context.Context, id uuid.UUID, channel string, at time.Time) error {
	var col string
	switch channel {
	case "email":
		col = "email_verified_at"
	case "phone":
		col = "phone_verified_at"
	default:
		return fmt.Errorf("unknown verification channel %q", channel)
	}
	// col comes from the switch above, never from input.
	_, err := r.pool.Exec(ctx, `UPDATE users SET `+col+` = $2 WHERE id = $1`, id, at)
	if err != nil {
		return fmt.Errorf("marking %s verified: %w", channel, err)
	}
	return nil
}

// SetStatus transitions the account status.
func (r *Repo) SetStatus(ctx context.Context, id uuid.UUID, status string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE users SET status = $2 WHERE id = $1`, id, status)
	if err != nil {
		return fmt.Errorf("setting status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ReviewCount returns the user's published-review count (credibility signal).
func (r *Repo) ReviewCount(ctx context.Context, id uuid.UUID) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT count(*) FROM reviews WHERE user_id = $1 AND moderation_status = 'published'`, id).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("counting reviews: %w", err)
	}
	return n, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
