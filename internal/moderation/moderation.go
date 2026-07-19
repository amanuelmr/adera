// Package moderation implements reports, the moderation queue, review and
// target decisions, evidence verification decisions, and the append-only
// audit trail. Reports never auto-hide content: only a moderator decision
// changes visibility (documented in docs/moderation-policy.md).
package moderation

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
	"github.com/adera-platform/backend/internal/reviews"
	"github.com/adera-platform/backend/internal/targets"
)

// Report reasons mirror the schema constraint.
var ReportReasons = map[string]bool{
	"spam": true, "fake_experience": true, "conflict_of_interest": true,
	"harassment": true, "hate_speech": true, "personal_information": true,
	"unsupported_accusation": true, "irrelevant": true, "duplicate": true,
	"manipulated_evidence": true,
}

// Report statuses.
const (
	ReportOpen      = "open"
	ReportInReview  = "in_review"
	ReportResolved  = "resolved"
	ReportDismissed = "dismissed"
)

// Report is a user/business report against a review or a target.
type Report struct {
	ID             uuid.UUID  `json:"id"`
	ReporterID     uuid.UUID  `json:"reporter_id"`
	ReviewID       *uuid.UUID `json:"review_id,omitempty"`
	TargetID       *uuid.UUID `json:"target_id,omitempty"`
	Reason         string     `json:"reason"`
	Details        string     `json:"details,omitempty"`
	Status         string     `json:"status"`
	ResolutionNote string     `json:"resolution_note,omitempty"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// AuditEntry is one append-only moderation action.
type AuditEntry struct {
	ID          uuid.UUID  `json:"id"`
	ActorID     *uuid.UUID `json:"actor_id,omitempty"`
	SubjectType string     `json:"subject_type"`
	SubjectID   uuid.UUID  `json:"subject_id"`
	Action      string     `json:"action"`
	Note        string     `json:"note,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// Service implements moderation operations.
type Service struct {
	pool        *pgxpool.Pool
	reviewsRepo *reviews.Repo
	targetsRepo *targets.Repo
}

func NewService(pool *pgxpool.Pool, reviewsRepo *reviews.Repo, targetsRepo *targets.Repo) *Service {
	return &Service{pool: pool, reviewsRepo: reviewsRepo, targetsRepo: targetsRepo}
}

const reportColumns = `id, reporter_id, review_id, target_id, reason, details, status,
	resolution_note, resolved_at, created_at`

func scanReport(row pgx.Row) (Report, error) {
	var rp Report
	err := row.Scan(&rp.ID, &rp.ReporterID, &rp.ReviewID, &rp.TargetID, &rp.Reason, &rp.Details,
		&rp.Status, &rp.ResolutionNote, &rp.ResolvedAt, &rp.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Report{}, web.ErrNotFound("report")
	}
	if err != nil {
		return Report{}, fmt.Errorf("scanning report: %w", err)
	}
	return rp, nil
}

// FileReport creates a report against a review or a target (exactly one).
func (s *Service) FileReport(ctx context.Context, reporterID uuid.UUID, reviewID, targetID *uuid.UUID, reason, details string) (Report, error) {
	if !ReportReasons[reason] {
		return Report{}, web.ErrValidation("invalid report").WithDetail("reason", "unsupported reason")
	}
	if len([]rune(details)) > 2000 {
		return Report{}, web.ErrValidation("invalid report").WithDetail("details", "too long")
	}
	if (reviewID == nil) == (targetID == nil) {
		return Report{}, web.ErrValidation("invalid report").WithDetail("subject", "exactly one of review or target")
	}
	// Subject must exist and be visible.
	if reviewID != nil {
		var status string
		err := s.pool.QueryRow(ctx, `SELECT moderation_status FROM reviews WHERE id = $1`, *reviewID).Scan(&status)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && (status == reviews.StatusRemoved || status == reviews.StatusRejected)) {
			return Report{}, web.ErrNotFound("review")
		}
		if err != nil {
			return Report{}, fmt.Errorf("checking review: %w", err)
		}
	} else {
		var status string
		err := s.pool.QueryRow(ctx, `SELECT moderation_status FROM review_targets WHERE id = $1`, *targetID).Scan(&status)
		if errors.Is(err, pgx.ErrNoRows) {
			return Report{}, web.ErrNotFound("target")
		}
		if err != nil {
			return Report{}, fmt.Errorf("checking target: %w", err)
		}
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO reports (id, reporter_id, review_id, target_id, reason, details)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT DO NOTHING
		RETURNING `+reportColumns,
		uuid.New(), reporterID, reviewID, targetID, reason, strings.TrimSpace(details))
	rp, err := scanReport(row)
	if err != nil {
		var appErr *web.Error
		if errors.As(err, &appErr) && appErr.Code == web.CodeNotFound {
			// ON CONFLICT DO NOTHING returned no row: duplicate open report.
			return Report{}, web.ErrConflict("you already have an open report for this")
		}
		return Report{}, err
	}
	return rp, nil
}

// MyReports lists the caller's reports (status transparency).
func (s *Service) MyReports(ctx context.Context, reporterID uuid.UUID) ([]Report, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+reportColumns+` FROM reports WHERE reporter_id = $1 ORDER BY created_at DESC LIMIT 100`, reporterID)
	if err != nil {
		return nil, fmt.Errorf("querying own reports: %w", err)
	}
	defer rows.Close()
	return collectReports(rows)
}

// Queue lists reports for moderators, oldest first (takedown SLA ordering).
func (s *Service) Queue(ctx context.Context, status string, limit int) ([]Report, error) {
	if status == "" {
		status = ReportOpen
	}
	switch status {
	case ReportOpen, ReportInReview, ReportResolved, ReportDismissed:
	default:
		return nil, web.ErrValidation("invalid filter").WithDetail("status", "unsupported status")
	}
	rows, err := s.pool.Query(ctx, `
		SELECT `+reportColumns+` FROM reports WHERE status = $1 ORDER BY created_at ASC LIMIT $2`, status, limit)
	if err != nil {
		return nil, fmt.Errorf("querying report queue: %w", err)
	}
	defer rows.Close()
	return collectReports(rows)
}

func collectReports(rows pgx.Rows) ([]Report, error) {
	var out []Report
	for rows.Next() {
		rp, err := scanReport(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rp)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating reports: %w", err)
	}
	return out, nil
}

// GetReport loads one report (moderator).
func (s *Service) GetReport(ctx context.Context, id uuid.UUID) (Report, error) {
	return scanReport(s.pool.QueryRow(ctx, `SELECT `+reportColumns+` FROM reports WHERE id = $1`, id))
}

// ResolveReport transitions a report's status and records the audit action.
func (s *Service) ResolveReport(ctx context.Context, reportID, actorID uuid.UUID, status, note string) (Report, error) {
	switch status {
	case ReportInReview, ReportResolved, ReportDismissed:
	default:
		return Report{}, web.ErrValidation("invalid decision").WithDetail("status", "must be in_review, resolved, or dismissed")
	}
	row := s.pool.QueryRow(ctx, `
		UPDATE reports SET status = $2, resolved_by = $3,
			resolved_at = CASE WHEN $2 IN ('resolved', 'dismissed') THEN now() END,
			resolution_note = $4
		WHERE id = $1
		RETURNING `+reportColumns, reportID, status, actorID, strings.TrimSpace(note))
	rp, err := scanReport(row)
	if err != nil {
		return Report{}, err
	}
	if err := s.audit(ctx, actorID, "report", reportID, "report_"+status, note, nil); err != nil {
		return Report{}, err
	}
	return rp, nil
}

// Review decision actions → moderation statuses.
var reviewActions = map[string]string{
	"approve":      reviews.StatusPublished,
	"restore":      reviews.StatusPublished,
	"under_review": reviews.StatusUnderReview,
	"hide":         reviews.StatusHidden,
	"reject":       reviews.StatusRejected,
	"remove":       reviews.StatusRemoved,
}

// DecideReview applies a moderator decision to a review. Aggregates adjust
// transactionally inside reviews.SetModerationStatus.
func (s *Service) DecideReview(ctx context.Context, reviewID, actorID uuid.UUID, action, note string) (string, error) {
	newStatus, ok := reviewActions[action]
	if !ok {
		return "", web.ErrValidation("invalid decision").
			WithDetail("action", "must be approve, restore, under_review, hide, reject, or remove")
	}
	oldStatus, err := s.reviewsRepo.SetModerationStatus(ctx, reviewID, newStatus)
	if err != nil {
		return "", err
	}
	if err := s.audit(ctx, actorID, "review", reviewID, "review_"+action, note,
		map[string]string{"from": oldStatus, "to": newStatus}); err != nil {
		return "", err
	}
	return newStatus, nil
}

// Target decision actions → target moderation statuses.
var targetActions = map[string]string{
	"approve": targets.ModerationPublished,
	"restore": targets.ModerationPublished,
	"hide":    targets.ModerationHidden,
	"remove":  targets.ModerationRemoved,
}

// DecideTarget applies a moderator decision to a target submission.
func (s *Service) DecideTarget(ctx context.Context, targetID, actorID uuid.UUID, action, note string) (string, error) {
	newStatus, ok := targetActions[action]
	if !ok {
		return "", web.ErrValidation("invalid decision").
			WithDetail("action", "must be approve, restore, hide, or remove")
	}
	if err := s.targetsRepo.SetModerationStatus(ctx, targetID, newStatus, nil); err != nil {
		return "", err
	}
	if err := s.audit(ctx, actorID, "target", targetID, "target_"+action, note, nil); err != nil {
		return "", err
	}
	return newStatus, nil
}

// evidenceLevelByKind maps accepted evidence kinds to the verification level
// they grant (see docs/verification-model.md).
var evidenceLevelByKind = map[string]string{
	"receipt":          reviews.VerifyReceipt,
	"order_screenshot": reviews.VerifyReceipt,
	"product_photo":    reviews.VerifyReceipt,
	"service_result":   reviews.VerifyReceipt,
	"location_qr":      reviews.VerifyLocation,
}

// DecideEvidence accepts or rejects submitted evidence. Acceptance upgrades
// the review's verification level; a review is never labeled verified before
// this step succeeds.
func (s *Service) DecideEvidence(ctx context.Context, evidenceID, actorID uuid.UUID, decision, note string) error {
	if decision != "accepted" && decision != "rejected" {
		return web.ErrValidation("invalid decision").WithDetail("decision", "must be accepted or rejected")
	}
	var reviewID uuid.UUID
	var kind, status string
	err := s.pool.QueryRow(ctx, `
		SELECT review_id, kind, status FROM review_evidence WHERE id = $1`, evidenceID).
		Scan(&reviewID, &kind, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return web.ErrNotFound("evidence")
	}
	if err != nil {
		return fmt.Errorf("loading evidence: %w", err)
	}
	if status != "submitted" {
		return web.ErrConflict("this evidence is already decided")
	}
	if _, err := s.pool.Exec(ctx, `
		UPDATE review_evidence SET status = $2, reviewed_by = $3, reviewed_at = now(), note = $4
		WHERE id = $1`, evidenceID, decision, actorID, strings.TrimSpace(note)); err != nil {
		return fmt.Errorf("updating evidence: %w", err)
	}
	if decision == "accepted" {
		if err := s.reviewsRepo.UpgradeVerification(ctx, reviewID, evidenceLevelByKind[kind]); err != nil {
			return err
		}
	}
	return s.audit(ctx, actorID, "evidence", evidenceID, "evidence_"+decision, note,
		map[string]string{"review_id": reviewID.String(), "kind": kind})
}

// AddNote records a free-form moderation note on any subject.
func (s *Service) AddNote(ctx context.Context, actorID uuid.UUID, subjectType string, subjectID uuid.UUID, note string) error {
	switch subjectType {
	case "review", "target", "claim", "user", "report", "response", "evidence":
	default:
		return web.ErrValidation("invalid note").WithDetail("subject_type", "unsupported subject type")
	}
	if n := len([]rune(strings.TrimSpace(note))); n < 1 || n > 2000 {
		return web.ErrValidation("invalid note").WithDetail("note", "must be 1-2000 characters")
	}
	return s.audit(ctx, actorID, subjectType, subjectID, "note", note, nil)
}

// Audit lists the audit trail for a subject.
func (s *Service) Audit(ctx context.Context, subjectType string, subjectID uuid.UUID, limit int) ([]AuditEntry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, actor_id, subject_type, subject_id, action, note, created_at
		FROM moderation_actions
		WHERE subject_type = $1 AND subject_id = $2
		ORDER BY created_at DESC LIMIT $3`, subjectType, subjectID, limit)
	if err != nil {
		return nil, fmt.Errorf("querying audit trail: %w", err)
	}
	defer rows.Close()
	var out []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.ActorID, &e.SubjectType, &e.SubjectID, &e.Action, &e.Note, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning audit entry: %w", err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating audit entries: %w", err)
	}
	return out, nil
}

func (s *Service) audit(ctx context.Context, actorID uuid.UUID, subjectType string, subjectID uuid.UUID, action, note string, metadata map[string]string) error {
	if metadata == nil {
		metadata = map[string]string{}
	}
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO moderation_actions (id, actor_id, subject_type, subject_id, action, note, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		uuid.New(), actorID, subjectType, subjectID, action, strings.TrimSpace(note), metadata); err != nil {
		return fmt.Errorf("recording audit action: %w", err)
	}
	return nil
}
