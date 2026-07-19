// Package businesses owns the claimable business entity, membership (the
// object-level authorization source for owner actions), and public business
// responses to reviews.
package businesses

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adera-platform/backend/internal/platform/web"
)

// Business is the owning entity behind one or more review targets.
type Business struct {
	ID                 uuid.UUID `json:"id"`
	Name               string    `json:"name"`
	Slug               string    `json:"slug"`
	Description        string    `json:"description"`
	VerificationStatus string    `json:"verification_status"`
	Status             string    `json:"status"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// Response is a business's public reply to a review.
type Response struct {
	ID           uuid.UUID `json:"id"`
	ReviewID     uuid.UUID `json:"review_id"`
	BusinessID   uuid.UUID `json:"business_id"`
	AuthorUserID uuid.UUID `json:"-"`
	Body         string    `json:"body"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Repo provides business persistence and the membership authority.
type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

// Pool exposes the underlying pool to sibling modules composing transactions
// (claims approval grants membership atomically).
func (r *Repo) Pool() *pgxpool.Pool { return r.pool }

// IsMember reports whether the user has an approved membership.
func (r *Repo) IsMember(ctx context.Context, businessID, userID uuid.UUID) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM business_members WHERE business_id = $1 AND user_id = $2)`,
		businessID, userID).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("checking membership: %w", err)
	}
	return ok, nil
}

const businessColumns = `id, name, slug, description, verification_status, status, created_at, updated_at`

func scanBusiness(row pgx.Row) (Business, error) {
	var b Business
	err := row.Scan(&b.ID, &b.Name, &b.Slug, &b.Description, &b.VerificationStatus,
		&b.Status, &b.CreatedAt, &b.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Business{}, web.ErrNotFound("business")
	}
	if err != nil {
		return Business{}, fmt.Errorf("scanning business: %w", err)
	}
	return b, nil
}

// Get loads a business by ID.
func (r *Repo) Get(ctx context.Context, id uuid.UUID) (Business, error) {
	return scanBusiness(r.pool.QueryRow(ctx, `SELECT `+businessColumns+` FROM businesses WHERE id = $1`, id))
}

// Create registers a new business profile. Creating a business does NOT grant
// control: control comes only from an approved claim.
func (r *Repo) Create(ctx context.Context, name, description string, createdBy uuid.UUID) (Business, error) {
	name = strings.TrimSpace(name)
	if l := len([]rune(name)); l < 2 || l > 160 {
		return Business{}, web.ErrValidation("invalid business").WithDetail("name", "must be 2-160 characters")
	}
	if len([]rune(description)) > 2000 {
		return Business{}, web.ErrValidation("invalid business").WithDetail("description", "too long")
	}
	id := uuid.New()
	slug := Slugify(name, id)
	row := r.pool.QueryRow(ctx, `
		INSERT INTO businesses (id, name, slug, description, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+businessColumns, id, name, slug, strings.TrimSpace(description), createdBy)
	return scanBusiness(row)
}

// UpdateDescription is the only member-editable business field for now
// (explicit allowlist; names change via moderation to prevent identity swaps).
func (r *Repo) UpdateDescription(ctx context.Context, id uuid.UUID, description string) (Business, error) {
	if len([]rune(description)) > 2000 {
		return Business{}, web.ErrValidation("invalid business").WithDetail("description", "too long")
	}
	row := r.pool.QueryRow(ctx, `
		UPDATE businesses SET description = $2 WHERE id = $1
		RETURNING `+businessColumns, id, strings.TrimSpace(description))
	return scanBusiness(row)
}

// Slugify builds a URL slug from a name, falling back to the id for
// non-Latin (e.g., pure Amharic) names, and always appending an id fragment
// for uniqueness.
func Slugify(name string, id uuid.UUID) string {
	var b strings.Builder
	lastDash := true
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	base := strings.Trim(b.String(), "-")
	suffix := strings.Split(id.String(), "-")[0]
	if base == "" {
		return "t-" + suffix
	}
	if len(base) > 100 {
		base = base[:100]
	}
	return base + "-" + suffix
}
