// Package admin implements administrator-only operations: account
// suspension, category and criteria management, and merging duplicate
// targets. Every action lands in the append-only audit trail.
package admin

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adera-platform/backend/internal/platform/database"
	"github.com/adera-platform/backend/internal/platform/web"
	"github.com/adera-platform/backend/internal/users"
)

// Service implements admin operations.
type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool, usersRepo *users.Repo) *Service {
	return &Service{pool: pool}
}

func auditTx(ctx context.Context, tx pgx.Tx, actorID uuid.UUID, subjectType string, subjectID uuid.UUID, action, note string, metadata map[string]string) error {
	if metadata == nil {
		metadata = map[string]string{}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO moderation_actions (id, actor_id, subject_type, subject_id, action, note, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		uuid.New(), actorID, subjectType, subjectID, action, strings.TrimSpace(note), metadata); err != nil {
		return fmt.Errorf("recording audit action: %w", err)
	}
	return nil
}

// SuspendUser suspends an account and revokes all its sessions.
func (s *Service) SuspendUser(ctx context.Context, userID, actorID uuid.UUID, note string) error {
	if userID == actorID {
		return web.ErrValidation("invalid action").WithDetail("user_id", "you cannot suspend yourself")
	}
	return database.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE users SET status = $2 WHERE id = $1`, userID, users.StatusSuspended)
		if err != nil {
			return fmt.Errorf("setting status: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return web.ErrNotFound("user")
		}
		if _, err := tx.Exec(ctx, `
			UPDATE auth_sessions SET revoked_at = now(), revoked_reason = 'account_suspended'
			WHERE user_id = $1 AND revoked_at IS NULL`, userID); err != nil {
			return fmt.Errorf("revoking sessions: %w", err)
		}
		return auditTx(ctx, tx, actorID, "user", userID, "user_suspended", note, nil)
	})
}

// ReinstateUser lifts a suspension.
func (s *Service) ReinstateUser(ctx context.Context, userID, actorID uuid.UUID, note string) error {
	return database.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE users SET status = $2 WHERE id = $1`, userID, users.StatusActive)
		if err != nil {
			return fmt.Errorf("setting status: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return web.ErrNotFound("user")
		}
		return auditTx(ctx, tx, actorID, "user", userID, "user_reinstated", note, nil)
	})
}

// GrantRole grants moderator/admin/business_owner roles.
func (s *Service) GrantRole(ctx context.Context, userID, actorID uuid.UUID, role string) error {
	switch role {
	case web.RoleModerator, web.RoleAdmin, web.RoleBusinessOwner:
	default:
		return web.ErrValidation("invalid role").WithDetail("role", "must be moderator, admin, or business_owner")
	}
	return database.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_roles (user_id, role, granted_by) VALUES ($1, $2, $3)
			ON CONFLICT DO NOTHING`, userID, role, actorID); err != nil {
			return fmt.Errorf("granting role: %w", err)
		}
		return auditTx(ctx, tx, actorID, "user", userID, "role_granted", role, map[string]string{"role": role})
	})
}

var codeRe = regexp.MustCompile(`^[a-z0-9_]{2,40}$`)

// CategoryInput is the admin category create/update payload.
type CategoryInput struct {
	Code             string            `json:"code"`
	Name             string            `json:"name"`
	NameTranslations map[string]string `json:"name_translations"`
	Description      string            `json:"description"`
	SortOrder        int               `json:"sort_order"`
	Active           *bool             `json:"active"`
}

func (in *CategoryInput) validate(forCreate bool) error {
	e := web.ErrValidation("invalid category")
	if forCreate && !codeRe.MatchString(in.Code) {
		return e.WithDetail("code", "must be 2-40 chars of a-z, 0-9, _")
	}
	if l := len([]rune(strings.TrimSpace(in.Name))); l < 2 || l > 80 {
		return e.WithDetail("name", "must be 2-80 characters")
	}
	if len([]rune(in.Description)) > 500 {
		return e.WithDetail("description", "too long")
	}
	if in.NameTranslations == nil {
		in.NameTranslations = map[string]string{}
	}
	return nil
}

// CreateCategory adds a new category (inactive categories stay hidden from
// public listings until activated).
func (s *Service) CreateCategory(ctx context.Context, actorID uuid.UUID, in CategoryInput) (uuid.UUID, error) {
	if err := in.validate(true); err != nil {
		return uuid.Nil, err
	}
	id := uuid.New()
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	err := database.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			INSERT INTO categories (id, code, name, name_translations, description, sort_order, active)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			id, in.Code, strings.TrimSpace(in.Name), in.NameTranslations, strings.TrimSpace(in.Description), in.SortOrder, active); err != nil {
			return fmt.Errorf("inserting category: %w", err)
		}
		return auditTx(ctx, tx, actorID, "target", id, "category_created", in.Code, nil)
	})
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// UpdateCategory applies the allowlisted category fields.
func (s *Service) UpdateCategory(ctx context.Context, actorID, id uuid.UUID, in CategoryInput) error {
	if err := in.validate(false); err != nil {
		return err
	}
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	return database.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE categories SET name = $2, name_translations = $3, description = $4, sort_order = $5, active = $6
			WHERE id = $1`,
			id, strings.TrimSpace(in.Name), in.NameTranslations, strings.TrimSpace(in.Description), in.SortOrder, active)
		if err != nil {
			return fmt.Errorf("updating category: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return web.ErrNotFound("category")
		}
		return auditTx(ctx, tx, actorID, "target", id, "category_updated", "", nil)
	})
}

// CriterionInput is the admin criterion create/update payload.
type CriterionInput struct {
	Code              string            `json:"code"`
	Name              string            `json:"name"`
	LabelTranslations map[string]string `json:"label_translations"`
	Description       string            `json:"description"`
	Required          bool              `json:"required"`
	ScaleMin          int               `json:"scale_min"`
	ScaleMax          int               `json:"scale_max"`
	SortOrder         int               `json:"sort_order"`
	Active            *bool             `json:"active"`
}

func (in *CriterionInput) validate(forCreate bool) error {
	e := web.ErrValidation("invalid criterion")
	if forCreate && !codeRe.MatchString(in.Code) {
		return e.WithDetail("code", "must be 2-40 chars of a-z, 0-9, _")
	}
	if l := len([]rune(strings.TrimSpace(in.Name))); l < 2 || l > 80 {
		return e.WithDetail("name", "must be 2-80 characters")
	}
	if in.ScaleMin == 0 && in.ScaleMax == 0 {
		in.ScaleMin, in.ScaleMax = 1, 5
	}
	if in.ScaleMin < 0 || in.ScaleMax <= in.ScaleMin || in.ScaleMax > 10 {
		return e.WithDetail("scale", "scale must satisfy 0 <= min < max <= 10")
	}
	if in.LabelTranslations == nil {
		in.LabelTranslations = map[string]string{}
	}
	return nil
}

// CreateCriterion adds a rating criterion to a category.
func (s *Service) CreateCriterion(ctx context.Context, actorID, categoryID uuid.UUID, in CriterionInput) (uuid.UUID, error) {
	if err := in.validate(true); err != nil {
		return uuid.Nil, err
	}
	id := uuid.New()
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	err := database.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			INSERT INTO category_criteria
				(id, category_id, code, name, label_translations, description, required, scale_min, scale_max, sort_order, active)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			id, categoryID, in.Code, strings.TrimSpace(in.Name), in.LabelTranslations,
			strings.TrimSpace(in.Description), in.Required, in.ScaleMin, in.ScaleMax, in.SortOrder, active); err != nil {
			return fmt.Errorf("inserting criterion: %w", err)
		}
		return auditTx(ctx, tx, actorID, "target", id, "criterion_created", in.Code, nil)
	})
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// UpdateCriterion applies the allowlisted criterion fields. Scale changes
// affect only future reviews; existing scores keep their original scale.
func (s *Service) UpdateCriterion(ctx context.Context, actorID, id uuid.UUID, in CriterionInput) error {
	if err := in.validate(false); err != nil {
		return err
	}
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	return database.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE category_criteria
			SET name = $2, label_translations = $3, description = $4, required = $5,
				scale_min = $6, scale_max = $7, sort_order = $8, active = $9
			WHERE id = $1`,
			id, strings.TrimSpace(in.Name), in.LabelTranslations, strings.TrimSpace(in.Description),
			in.Required, in.ScaleMin, in.ScaleMax, in.SortOrder, active)
		if err != nil {
			return fmt.Errorf("updating criterion: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return web.ErrNotFound("criterion")
		}
		return auditTx(ctx, tx, actorID, "target", id, "criterion_updated", "", nil)
	})
}

// MergeTargets folds duplicate `source` into `destination` in one
// transaction: reviews, aliases, media and reports move; the source is marked
// merged; the destination's aggregates are recomputed from scratch.
func (s *Service) MergeTargets(ctx context.Context, actorID, sourceID, destID uuid.UUID, note string) error {
	if sourceID == destID {
		return web.ErrValidation("invalid merge").WithDetail("into_id", "cannot merge a target into itself")
	}
	return database.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		// Lock both targets in a stable order to avoid deadlocks.
		first, second := sourceID, destID
		if destID.String() < sourceID.String() {
			first, second = destID, sourceID
		}
		for _, id := range []uuid.UUID{first, second} {
			var status string
			err := tx.QueryRow(ctx, `
				SELECT moderation_status FROM review_targets WHERE id = $1 FOR UPDATE`, id).Scan(&status)
			if errors.Is(err, pgx.ErrNoRows) {
				return web.ErrNotFound("target")
			}
			if err != nil {
				return fmt.Errorf("locking target: %w", err)
			}
			if id == sourceID && status == "merged" {
				return web.ErrConflict("source target is already merged")
			}
			if id == destID && status != "published" {
				return web.ErrConflict("destination target must be published")
			}
		}

		// Move content. A user who reviewed both keeps both reviews
		// (documented merge behavior).
		if _, err := tx.Exec(ctx, `UPDATE reviews SET target_id = $2 WHERE target_id = $1`, sourceID, destID); err != nil {
			return fmt.Errorf("moving reviews: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO target_aliases (id, target_id, alias, created_by)
			SELECT gen_random_uuid(), $2, alias, created_by FROM target_aliases WHERE target_id = $1
			ON CONFLICT DO NOTHING`, sourceID, destID); err != nil {
			return fmt.Errorf("copying aliases: %w", err)
		}
		// The source's own name becomes an alias of the destination.
		if _, err := tx.Exec(ctx, `
			INSERT INTO target_aliases (id, target_id, alias)
			SELECT gen_random_uuid(), $2, name FROM review_targets WHERE id = $1
			ON CONFLICT DO NOTHING`, sourceID, destID); err != nil {
			return fmt.Errorf("aliasing source name: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE reports SET target_id = $2 WHERE target_id = $1 AND status IN ('open', 'in_review')`,
			sourceID, destID); err != nil {
			return fmt.Errorf("moving open reports: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE review_targets SET moderation_status = 'merged', merged_into_id = $2 WHERE id = $1`,
			sourceID, destID); err != nil {
			return fmt.Errorf("marking source merged: %w", err)
		}

		// Recompute both stats rows from ground truth.
		for _, id := range []uuid.UUID{sourceID, destID} {
			if _, err := tx.Exec(ctx, `
				UPDATE target_rating_stats s SET
					review_count = agg.n, rating_sum = agg.sum,
					count_1 = agg.c1, count_2 = agg.c2, count_3 = agg.c3, count_4 = agg.c4, count_5 = agg.c5,
					verified_count = agg.vn, verified_sum = agg.vsum,
					recommend_yes = agg.ry, recommend_total = agg.rt,
					return_sum = agg.rets, return_count = agg.retc,
					updated_at = now()
				FROM (
					SELECT
						count(*)::int AS n,
						coalesce(sum(overall_rating), 0)::bigint AS sum,
						count(*) FILTER (WHERE overall_rating = 1)::int AS c1,
						count(*) FILTER (WHERE overall_rating = 2)::int AS c2,
						count(*) FILTER (WHERE overall_rating = 3)::int AS c3,
						count(*) FILTER (WHERE overall_rating = 4)::int AS c4,
						count(*) FILTER (WHERE overall_rating = 5)::int AS c5,
						count(*) FILTER (WHERE verification_level IN ('receipt_submitted', 'location_verified', 'partner_verified'))::int AS vn,
						coalesce(sum(overall_rating) FILTER (WHERE verification_level IN ('receipt_submitted', 'location_verified', 'partner_verified')), 0)::bigint AS vsum,
						count(*) FILTER (WHERE would_recommend)::int AS ry,
						count(would_recommend)::int AS rt,
						coalesce(sum(return_likelihood), 0)::bigint AS rets,
						count(return_likelihood)::int AS retc
					FROM reviews
					WHERE target_id = $1 AND moderation_status = 'published'
				) agg
				WHERE s.target_id = $1`, id); err != nil {
				return fmt.Errorf("recomputing stats: %w", err)
			}
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO moderation_actions (id, actor_id, subject_type, subject_id, action, note, metadata)
			VALUES ($1, $2, 'target', $3, 'target_merged', $4, $5)`,
			uuid.New(), actorID, sourceID, strings.TrimSpace(note),
			map[string]string{"merged_into": destID.String()}); err != nil {
			return fmt.Errorf("recording audit action: %w", err)
		}
		return nil
	})
}
