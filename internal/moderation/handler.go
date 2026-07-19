package moderation

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/adera-platform/backend/internal/media"
	"github.com/adera-platform/backend/internal/platform/ratelimit"
	"github.com/adera-platform/backend/internal/platform/web"
)

// Handler serves report and moderation endpoints.
type Handler struct {
	svc      *Service
	mediaSvc *media.Service
	limiter  ratelimit.Limiter // per-account report limiter
}

func NewHandler(svc *Service, mediaSvc *media.Service, limiter ratelimit.Limiter) *Handler {
	return &Handler{svc: svc, mediaSvc: mediaSvc, limiter: limiter}
}

func (h *Handler) Routes(mux *http.ServeMux) {
	// Reporting (any authenticated user, including business owners).
	mux.Handle("POST /api/v1/reviews/{id}/reports", web.RequireAuth(http.HandlerFunc(h.reportReview)))
	mux.Handle("POST /api/v1/targets/{id}/reports", web.RequireAuth(http.HandlerFunc(h.reportTarget)))
	mux.Handle("GET /api/v1/users/me/reports", web.RequireAuth(http.HandlerFunc(h.myReports)))

	// Moderator endpoints (function-level authorization; admin implies moderator).
	mod := web.RequireRole(web.RoleModerator)
	mux.Handle("GET /api/v1/moderation/reports", mod(http.HandlerFunc(h.queue)))
	mux.Handle("GET /api/v1/moderation/reports/{id}", mod(http.HandlerFunc(h.getReport)))
	mux.Handle("POST /api/v1/moderation/reports/{id}/resolve", mod(http.HandlerFunc(h.resolveReport)))
	mux.Handle("POST /api/v1/moderation/reviews/{id}/decision", mod(http.HandlerFunc(h.decideReview)))
	mux.Handle("POST /api/v1/moderation/targets/{id}/decision", mod(http.HandlerFunc(h.decideTarget)))
	mux.Handle("GET /api/v1/moderation/reviews/{id}/evidence", mod(http.HandlerFunc(h.reviewEvidence)))
	mux.Handle("POST /api/v1/moderation/evidence/{id}/decision", mod(http.HandlerFunc(h.decideEvidence)))
	mux.Handle("POST /api/v1/moderation/notes", mod(http.HandlerFunc(h.addNote)))
	mux.Handle("GET /api/v1/moderation/audit", mod(http.HandlerFunc(h.audit)))
}

type reportRequest struct {
	Reason  string `json:"reason"`
	Details string `json:"details"`
}

func (h *Handler) fileReport(w http.ResponseWriter, r *http.Request, reviewID, targetID *uuid.UUID) {
	p, _ := web.PrincipalFromContext(r.Context())
	if !h.limiter.Allow("report:" + p.UserID.String()) {
		web.RespondError(w, r, web.ErrRateLimited())
		return
	}
	var req reportRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	rp, err := h.svc.FileReport(r.Context(), p.UserID, reviewID, targetID, req.Reason, req.Details)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusCreated, rp)
}

func (h *Handler) reportReview(w http.ResponseWriter, r *http.Request) {
	id, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	h.fileReport(w, r, &id, nil)
}

func (h *Handler) reportTarget(w http.ResponseWriter, r *http.Request) {
	id, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	h.fileReport(w, r, nil, &id)
}

func (h *Handler) myReports(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	reports, err := h.svc.MyReports(r.Context(), p.UserID)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, reports)
}

func (h *Handler) queue(w http.ResponseWriter, r *http.Request) {
	reports, err := h.svc.Queue(r.Context(), r.URL.Query().Get("status"), web.ParseLimit(r))
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, reports)
}

func (h *Handler) getReport(w http.ResponseWriter, r *http.Request) {
	id, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	rp, err := h.svc.GetReport(r.Context(), id)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, rp)
}

type resolveRequest struct {
	Status string `json:"status"`
	Note   string `json:"note"`
}

func (h *Handler) resolveReport(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	id, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	var req resolveRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	rp, err := h.svc.ResolveReport(r.Context(), id, p.UserID, req.Status, req.Note)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, rp)
}

type decisionRequest struct {
	Action string `json:"action"`
	Note   string `json:"note"`
}

func (h *Handler) decideReview(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	id, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	var req decisionRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	status, err := h.svc.DecideReview(r.Context(), id, p.UserID, req.Action, req.Note)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, map[string]string{"moderation_status": status})
}

func (h *Handler) decideTarget(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	id, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	var req decisionRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	status, err := h.svc.DecideTarget(r.Context(), id, p.UserID, req.Action, req.Note)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, map[string]string{"moderation_status": status})
}

func (h *Handler) reviewEvidence(w http.ResponseWriter, r *http.Request) {
	id, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	items, err := h.mediaSvc.ListEvidence(r.Context(), id, true)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, items)
}

type evidenceDecisionRequest struct {
	Decision string `json:"decision"`
	Note     string `json:"note"`
}

func (h *Handler) decideEvidence(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	id, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	var req evidenceDecisionRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	if err := h.svc.DecideEvidence(r.Context(), id, p.UserID, req.Decision, req.Note); err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, map[string]string{"status": req.Decision})
}

type noteRequest struct {
	SubjectType string `json:"subject_type"`
	SubjectID   string `json:"subject_id"`
	Note        string `json:"note"`
}

func (h *Handler) addNote(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	var req noteRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	subjectID, err := uuid.Parse(req.SubjectID)
	if err != nil {
		web.RespondError(w, r, web.ErrValidation("invalid note").WithDetail("subject_id", "must be a UUID"))
		return
	}
	if err := h.svc.AddNote(r.Context(), p.UserID, req.SubjectType, subjectID, req.Note); err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusCreated, map[string]string{"status": "noted"})
}

func (h *Handler) audit(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	subjectID, err := uuid.Parse(q.Get("subject_id"))
	if err != nil {
		web.RespondError(w, r, web.ErrValidation("invalid query").WithDetail("subject_id", "must be a UUID"))
		return
	}
	entries, err := h.svc.Audit(r.Context(), q.Get("subject_type"), subjectID, web.ParseLimit(r))
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, entries)
}
