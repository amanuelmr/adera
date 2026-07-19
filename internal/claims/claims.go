// Package claims implements business ownership claims: request, moderator
// decision, membership grant, and revocation. Claiming never grants
// moderation permissions — approval grants business membership and the
// business_owner role only.
package claims

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adera-platform/backend/internal/platform/database"
	"github.com/adera-platform/backend/internal/platform/web"
)

// Claim statuses.
const (
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusRejected = "rejected"
	StatusRevoked  = "revoked"
)

var claimMethods = map[string]bool{"document": true, "phone": true, "email": true, "other": true}

// Claim is an ownership request.
type Claim struct {
	ID           uuid.UUID  `json:"id"`
	BusinessID   uuid.UUID  `json:"business_id"`
	UserID       uuid.UUID  `json:"user_id"`
	Method       string     `json:"method"`
	Message      string     `json:"message,omitempty"`
	Status       string     `json:"status"`
	DecidedAt    *time.Time `json:"decided_at,omitempty"`
	DecisionNote string     `json:"decision_note,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// Service persists claims.
type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

const claimColumns = `id, business_id, user_id, method, message, status, decided_at, decision_note, created_at`

func scanClaim(row pgx.Row) (Claim, error) {
	var c Claim
	err := row.Scan(&c.ID, &c.BusinessID, &c.UserID, &c.Method, &c.Message, &c.Status,
		&c.DecidedAt, &c.DecisionNote, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Claim{}, web.ErrNotFound("claim")
	}
	if err != nil {
		return Claim{}, fmt.Errorf("scanning claim: %w", err)
	}
	return c, nil
}

// Submit files a claim for a business.
func (s *Service) Submit(ctx context.Context, businessID, userID uuid.UUID, method, message string) (Claim, error) {
	if !claimMethods[method] {
		return Claim{}, web.ErrValidation("invalid claim").WithDetail("method", "must be document, phone, email, or other")
	}
	if len([]rune(message)) > 2000 {
		return Claim{}, web.ErrValidation("invalid claim").WithDetail("message", "too long")
	}
	var exists bool
	if err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM businesses WHERE id = $1 AND status = 'active')`, businessID).Scan(&exists); err != nil {
		return Claim{}, fmt.Errorf("checking business: %w", err)
	}
	if !exists {
		return Claim{}, web.ErrNotFound("business")
	}
	var member bool
	if err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM business_members WHERE business_id = $1 AND user_id = $2)`,
		businessID, userID).Scan(&member); err != nil {
		return Claim{}, fmt.Errorf("checking membership: %w", err)
	}
	if member {
		return Claim{}, web.ErrConflict("you already manage this business")
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO business_claims (id, business_id, user_id, method, message)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+claimColumns, uuid.New(), businessID, userID, method, strings.TrimSpace(message))
	c, err := scanClaim(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Claim{}, web.ErrConflict("you already have a pending claim for this business")
		}
		return Claim{}, err
	}
	return c, nil
}

// Mine lists the caller's claims.
func (s *Service) Mine(ctx context.Context, userID uuid.UUID) ([]Claim, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+claimColumns+` FROM business_claims WHERE user_id = $1 ORDER BY created_at DESC LIMIT 50`, userID)
	if err != nil {
		return nil, fmt.Errorf("querying claims: %w", err)
	}
	defer rows.Close()
	var out []Claim
	for rows.Next() {
		c, err := scanClaim(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating claims: %w", err)
	}
	return out, nil
}

// Decide resolves a pending claim (moderator action). Approval atomically
// grants business membership, the business_owner role, marks the business
// claimed, and records the audit action.
func (s *Service) Decide(ctx context.Context, claimID, actorID uuid.UUID, decision, note string) (Claim, error) {
	if decision != StatusApproved && decision != StatusRejected {
		return Claim{}, web.ErrValidation("invalid decision").WithDetail("decision", "must be approved or rejected")
	}
	if len([]rune(note)) > 2000 {
		return Claim{}, web.ErrValidation("invalid decision").WithDetail("note", "too long")
	}
	var c Claim
	err := database.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		var err error
		c, err = scanClaim(tx.QueryRow(ctx, `
			SELECT `+claimColumns+` FROM business_claims WHERE id = $1 FOR UPDATE`, claimID))
		if err != nil {
			return err
		}
		if c.Status != StatusPending {
			return web.ErrConflict("this claim is already decided")
		}
		if err := tx.QueryRow(ctx, `
			UPDATE business_claims
			SET status = $2, decided_by = $3, decided_at = now(), decision_note = $4
			WHERE id = $1
			RETURNING `+claimColumns, claimID, decision, actorID, strings.TrimSpace(note)).
			Scan(&c.ID, &c.BusinessID, &c.UserID, &c.Method, &c.Message, &c.Status,
				&c.DecidedAt, &c.DecisionNote, &c.CreatedAt); err != nil {
			return fmt.Errorf("updating claim: %w", err)
		}
		if decision == StatusApproved {
			if _, err := tx.Exec(ctx, `
				INSERT INTO business_members (business_id, user_id, role, granted_by)
				VALUES ($1, $2, 'owner', $3) ON CONFLICT DO NOTHING`,
				c.BusinessID, c.UserID, actorID); err != nil {
				return fmt.Errorf("granting membership: %w", err)
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO user_roles (user_id, role, granted_by)
				VALUES ($1, 'business_owner', $2) ON CONFLICT DO NOTHING`,
				c.UserID, actorID); err != nil {
				return fmt.Errorf("granting business_owner role: %w", err)
			}
			if _, err := tx.Exec(ctx, `
				UPDATE businesses SET verification_status = 'claimed'
				WHERE id = $1 AND verification_status = 'unverified'`, c.BusinessID); err != nil {
				return fmt.Errorf("marking business claimed: %w", err)
			}
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO moderation_actions (id, actor_id, subject_type, subject_id, action, note, metadata)
			VALUES ($1, $2, 'claim', $3, $4, $5, $6)`,
			uuid.New(), actorID, claimID, "claim_"+decision, strings.TrimSpace(note),
			map[string]string{"business_id": c.BusinessID.String(), "user_id": c.UserID.String()}); err != nil {
			return fmt.Errorf("recording audit action: %w", err)
		}
		return nil
	})
	if err != nil {
		return Claim{}, err
	}
	return c, nil
}

// Revoke withdraws an approved claim (moderator action): membership is
// removed and the action audited. The business_owner role is kept if the user
// still manages other businesses.
func (s *Service) Revoke(ctx context.Context, claimID, actorID uuid.UUID, note string) (Claim, error) {
	var c Claim
	err := database.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		var err error
		c, err = scanClaim(tx.QueryRow(ctx, `
			SELECT `+claimColumns+` FROM business_claims WHERE id = $1 FOR UPDATE`, claimID))
		if err != nil {
			return err
		}
		if c.Status != StatusApproved {
			return web.ErrConflict("only approved claims can be revoked")
		}
		if _, err := tx.Exec(ctx, `
			UPDATE business_claims SET status = 'revoked', decided_by = $2, decided_at = now(), decision_note = $3
			WHERE id = $1`, claimID, actorID, strings.TrimSpace(note)); err != nil {
			return fmt.Errorf("revoking claim: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			DELETE FROM business_members WHERE business_id = $1 AND user_id = $2`,
			c.BusinessID, c.UserID); err != nil {
			return fmt.Errorf("removing membership: %w", err)
		}
		var stillOwner bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM business_members WHERE user_id = $1)`, c.UserID).Scan(&stillOwner); err != nil {
			return fmt.Errorf("checking remaining memberships: %w", err)
		}
		if !stillOwner {
			if _, err := tx.Exec(ctx, `
				DELETE FROM user_roles WHERE user_id = $1 AND role = 'business_owner'`, c.UserID); err != nil {
				return fmt.Errorf("removing business_owner role: %w", err)
			}
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO moderation_actions (id, actor_id, subject_type, subject_id, action, note, metadata)
			VALUES ($1, $2, 'claim', $3, 'claim_revoked', $4, $5)`,
			uuid.New(), actorID, claimID, strings.TrimSpace(note),
			map[string]string{"business_id": c.BusinessID.String(), "user_id": c.UserID.String()}); err != nil {
			return fmt.Errorf("recording audit action: %w", err)
		}
		c.Status = StatusRevoked
		return nil
	})
	if err != nil {
		return Claim{}, err
	}
	return c, nil
}

// Handler serves claim endpoints.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Routes(mux *http.ServeMux) {
	mux.Handle("POST /api/v1/businesses/{id}/claims", web.RequireAuth(http.HandlerFunc(h.submit)))
	mux.Handle("GET /api/v1/claims/mine", web.RequireAuth(http.HandlerFunc(h.mine)))
	mux.Handle("POST /api/v1/moderation/claims/{id}/decision",
		web.RequireRole(web.RoleModerator)(http.HandlerFunc(h.decide)))
	mux.Handle("POST /api/v1/moderation/claims/{id}/revoke",
		web.RequireRole(web.RoleModerator)(http.HandlerFunc(h.revoke)))
}

type submitRequest struct {
	Method  string `json:"method"`
	Message string `json:"message"`
}

func (h *Handler) submit(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	businessID, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	var req submitRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	c, err := h.svc.Submit(r.Context(), businessID, p.UserID, req.Method, req.Message)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusCreated, c)
}

func (h *Handler) mine(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	claims, err := h.svc.Mine(r.Context(), p.UserID)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, claims)
}

type decideRequest struct {
	Decision string `json:"decision"`
	Note     string `json:"note"`
}

func (h *Handler) decide(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	claimID, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	var req decideRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	c, err := h.svc.Decide(r.Context(), claimID, p.UserID, req.Decision, req.Note)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, c)
}

type revokeRequest struct {
	Note string `json:"note"`
}

func (h *Handler) revoke(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	claimID, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	var req revokeRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	c, err := h.svc.Revoke(r.Context(), claimID, p.UserID, req.Note)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, c)
}
