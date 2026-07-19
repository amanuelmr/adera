package reviews

import (
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/adera-platform/backend/internal/platform/ratelimit"
	"github.com/adera-platform/backend/internal/platform/web"
)

// Handler serves review endpoints.
type Handler struct {
	repo    *Repo
	limiter ratelimit.Limiter // per-account write limiter
}

func NewHandler(repo *Repo, limiter ratelimit.Limiter) *Handler {
	return &Handler{repo: repo, limiter: limiter}
}

func (h *Handler) Routes(mux *http.ServeMux) {
	mux.Handle("POST /api/v1/reviews", web.RequireAuth(http.HandlerFunc(h.create)))
	mux.HandleFunc("GET /api/v1/reviews/{id}", h.get)
	mux.Handle("PUT /api/v1/reviews/{id}", web.RequireAuth(http.HandlerFunc(h.update)))
	mux.Handle("DELETE /api/v1/reviews/{id}", web.RequireAuth(http.HandlerFunc(h.remove)))
	mux.HandleFunc("GET /api/v1/targets/{id}/reviews", h.listForTarget)
	mux.Handle("PUT /api/v1/reviews/{id}/helpful", web.RequireAuth(http.HandlerFunc(h.vote)))
	mux.Handle("DELETE /api/v1/reviews/{id}/helpful", web.RequireAuth(http.HandlerFunc(h.unvote)))
	mux.Handle("GET /api/v1/users/me/reviews", web.RequireAuth(http.HandlerFunc(h.listMine)))
}

type reviewRequest struct {
	TargetID         string         `json:"target_id"`
	OverallRating    int            `json:"overall_rating"`
	Title            string         `json:"title"`
	Body             string         `json:"body"`
	Language         string         `json:"language"`
	ExperienceDate   string         `json:"experience_date"` // YYYY-MM-DD
	PricePaid        *float64       `json:"price_paid"`
	Currency         string         `json:"currency"`
	WouldRecommend   *bool          `json:"would_recommend"`
	ReturnLikelihood *int           `json:"return_likelihood"`
	DiscoverySource  string         `json:"discovery_source"`
	ExpectationMatch string         `json:"expectation_match"`
	SocialMediaURL   string         `json:"social_media_url"`
	CriterionScores  map[string]int `json:"criterion_scores"`
	// Version is required on update (optimistic concurrency), ignored on create.
	Version int `json:"version,omitempty"`
}

func (req reviewRequest) toInput(requireTarget bool) (Input, error) {
	in := Input{
		OverallRating:    req.OverallRating,
		Title:            req.Title,
		Body:             req.Body,
		Language:         req.Language,
		PricePaid:        req.PricePaid,
		Currency:         req.Currency,
		WouldRecommend:   req.WouldRecommend,
		ReturnLikelihood: req.ReturnLikelihood,
		DiscoverySource:  req.DiscoverySource,
		ExpectationMatch: req.ExpectationMatch,
		SocialMediaURL:   req.SocialMediaURL,
		CriterionScores:  req.CriterionScores,
	}
	if in.CriterionScores == nil {
		in.CriterionScores = map[string]int{}
	}
	if req.TargetID != "" {
		id, err := uuid.Parse(req.TargetID)
		if err != nil {
			return in, web.ErrValidation("invalid review").WithDetail("target_id", "must be a UUID")
		}
		in.TargetID = id
	} else if requireTarget {
		return in, web.ErrValidation("invalid review").WithDetail("target_id", "required")
	}
	if req.ExperienceDate != "" {
		d, err := time.Parse("2006-01-02", req.ExperienceDate)
		if err != nil {
			return in, web.ErrValidation("invalid review").WithDetail("experience_date", "must be YYYY-MM-DD")
		}
		in.ExperienceDate = &d
	}
	return in, nil
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	var req reviewRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	in, err := req.toInput(true)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	if !h.limiter.Allow("review:" + p.UserID.String()) {
		web.RespondError(w, r, web.ErrRateLimited())
		return
	}

	const endpoint = "POST /reviews"
	idemKey := r.Header.Get("Idempotency-Key")
	if idemKey != "" {
		if len(idemKey) > 200 {
			web.RespondError(w, r, web.ErrValidation("invalid request").WithDetail("Idempotency-Key", "too long"))
			return
		}
		stored, status, done, err := h.repo.BeginIdempotent(r.Context(), p.UserID, endpoint, idemKey, HashRequest(req))
		if err != nil {
			web.RespondError(w, r, err)
			return
		}
		if done {
			w.Header().Set("Idempotent-Replay", "true")
			web.Respond(w, status, stored)
			return
		}
	}

	rv, err := h.repo.Create(r.Context(), p.UserID, in)
	if err != nil {
		if idemKey != "" {
			// Release the key so an honest retry can succeed.
			if abortErr := h.repo.AbortIdempotent(r.Context(), p.UserID, endpoint, idemKey); abortErr != nil {
				web.RespondError(w, r, abortErr)
				return
			}
		}
		web.RespondError(w, r, err)
		return
	}
	if idemKey != "" {
		if err := h.repo.CompleteIdempotent(r.Context(), p.UserID, endpoint, idemKey, http.StatusCreated, rv); err != nil {
			web.RespondError(w, r, err)
			return
		}
	}
	web.Respond(w, http.StatusCreated, rv)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	rv, err := h.repo.Get(r.Context(), id)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	if rv.ModerationStatus != StatusPublished {
		p, ok := web.PrincipalFromContext(r.Context())
		if !ok || (p.UserID != rv.UserID && !p.HasRole(web.RoleModerator)) {
			web.RespondError(w, r, web.ErrNotFound("review"))
			return
		}
	}
	web.Respond(w, http.StatusOK, rv)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	id, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	var req reviewRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	if req.Version < 1 {
		web.RespondError(w, r, web.ErrValidation("invalid review").WithDetail("version", "current version required"))
		return
	}
	in, err := req.toInput(false)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	rv, err := h.repo.Update(r.Context(), p.UserID, id, req.Version, in)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, rv)
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	id, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	if err := h.repo.Remove(r.Context(), p.UserID, id); err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, map[string]string{"status": "removed"})
}

func (h *Handler) listForTarget(w http.ResponseWriter, r *http.Request) {
	targetID, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	q := r.URL.Query()
	var f ListFilter
	f.Language = q.Get("language")
	if v := q.Get("rating"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 5 {
			web.RespondError(w, r, web.ErrValidation("invalid filter").WithDetail("rating", "must be 1-5"))
			return
		}
		f.Rating = n
	}
	if v := q.Get("min_rating"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 5 {
			web.RespondError(w, r, web.ErrValidation("invalid filter").WithDetail("min_rating", "must be 1-5"))
			return
		}
		f.MinRating = n
	}
	if v := q.Get("verified"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			web.RespondError(w, r, web.ErrValidation("invalid filter").WithDetail("verified", "must be a boolean"))
			return
		}
		f.VerifiedOnly = b
	}
	if v := q.Get("discovery_source"); v != "" {
		if !contains(DiscoverySources, v) {
			web.RespondError(w, r, web.ErrValidation("invalid filter").WithDetail("discovery_source", "unsupported source"))
			return
		}
		f.DiscoverySource = v
	}
	if v := q.Get("expectation_match"); v != "" {
		if !contains(ExpectationMatches, v) {
			web.RespondError(w, r, web.ErrValidation("invalid filter").WithDetail("expectation_match", "unsupported value"))
			return
		}
		f.ExpectationMatch = v
	}
	if v := q.Get("since"); v != "" {
		d, err := time.Parse("2006-01-02", v)
		if err != nil {
			web.RespondError(w, r, web.ErrValidation("invalid filter").WithDetail("since", "must be YYYY-MM-DD"))
			return
		}
		f.Since = &d
	}
	cursor, err := web.ParseCursor(r)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	var viewer uuid.UUID
	if p, ok := web.PrincipalFromContext(r.Context()); ok {
		viewer = p.UserID
	}
	items, next, err := h.repo.ListForTarget(r.Context(), targetID, viewer, f, q.Get("sort"), cursor, web.ParseLimit(r))
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	meta := web.Meta{HasMore: next != nil}
	if next != nil {
		meta.NextCursor = next.Encode()
	}
	web.RespondPage(w, http.StatusOK, items, meta)
}

func (h *Handler) vote(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	id, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	if !h.limiter.Allow("vote:" + p.UserID.String()) {
		web.RespondError(w, r, web.ErrRateLimited())
		return
	}
	n, err := h.repo.Vote(r.Context(), id, p.UserID)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, map[string]any{"helpful_count": n, "voted": true})
}

func (h *Handler) unvote(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	id, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	n, err := h.repo.Unvote(r.Context(), id, p.UserID)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, map[string]any{"helpful_count": n, "voted": false})
}

func (h *Handler) listMine(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	cursor, err := web.ParseCursor(r)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	items, next, err := h.repo.ListForUser(r.Context(), p.UserID, cursor, web.ParseLimit(r))
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	meta := web.Meta{HasMore: next != nil}
	if next != nil {
		meta.NextCursor = next.Encode()
	}
	web.RespondPage(w, http.StatusOK, items, meta)
}
