package media

import (
	"net/http"

	"github.com/adera-platform/backend/internal/platform/ratelimit"
	"github.com/adera-platform/backend/internal/platform/web"
)

// Handler serves upload endpoints.
type Handler struct {
	svc     *Service
	limiter ratelimit.Limiter // per-account presign limiter
}

func NewHandler(svc *Service, limiter ratelimit.Limiter) *Handler {
	return &Handler{svc: svc, limiter: limiter}
}

func (h *Handler) Routes(mux *http.ServeMux) {
	mux.Handle("POST /api/v1/reviews/{id}/media", web.RequireAuth(http.HandlerFunc(h.presignMedia)))
	mux.Handle("POST /api/v1/media/uploads/{id}/finalize", web.RequireAuth(http.HandlerFunc(h.finalizeMedia)))
	mux.Handle("POST /api/v1/reviews/{id}/evidence", web.RequireAuth(http.HandlerFunc(h.presignEvidence)))
	mux.Handle("POST /api/v1/evidence/{id}/finalize", web.RequireAuth(http.HandlerFunc(h.finalizeEvidence)))
	mux.Handle("GET /api/v1/reviews/{id}/evidence", web.RequireAuth(http.HandlerFunc(h.listOwnEvidence)))
}

type presignRequest struct {
	ContentType string `json:"content_type"`
	Kind        string `json:"kind,omitempty"` // evidence only
}

func (h *Handler) presignMedia(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	reviewID, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	if !h.limiter.Allow("presign:" + p.UserID.String()) {
		web.RespondError(w, r, web.ErrRateLimited())
		return
	}
	var req presignRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	ticket, err := h.svc.PresignReviewMedia(r.Context(), reviewID, p.UserID, req.ContentType)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusCreated, ticket)
}

func (h *Handler) finalizeMedia(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	uploadID, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	id, key, err := h.svc.FinalizeReviewMedia(r.Context(), uploadID, p.UserID)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, map[string]string{
		"id":         id.String(),
		"object_key": key,
		"status":     "ready",
	})
}

func (h *Handler) presignEvidence(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	reviewID, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	if !h.limiter.Allow("presign:" + p.UserID.String()) {
		web.RespondError(w, r, web.ErrRateLimited())
		return
	}
	var req presignRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	ticket, err := h.svc.PresignEvidence(r.Context(), reviewID, p.UserID, req.Kind, req.ContentType)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusCreated, ticket)
}

func (h *Handler) finalizeEvidence(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	evidenceID, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	if err := h.svc.FinalizeEvidence(r.Context(), evidenceID, p.UserID); err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, map[string]string{"status": "submitted"})
}

// listOwnEvidence lets the review author see their submitted evidence and its
// verification status (with access URLs). Moderators use the moderation
// endpoint instead.
func (h *Handler) listOwnEvidence(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	reviewID, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	if err := h.svc.requireOwnReview(r.Context(), reviewID, p.UserID); err != nil {
		web.RespondError(w, r, err)
		return
	}
	items, err := h.svc.ListEvidence(r.Context(), reviewID, true)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, items)
}
