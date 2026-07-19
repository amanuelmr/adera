package businesses

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/adera-platform/backend/internal/platform/web"
)

// Handler serves business endpoints.
type Handler struct {
	repo *Repo
}

func NewHandler(repo *Repo) *Handler { return &Handler{repo: repo} }

func (h *Handler) Routes(mux *http.ServeMux) {
	mux.Handle("POST /api/v1/businesses", web.RequireAuth(http.HandlerFunc(h.create)))
	mux.HandleFunc("GET /api/v1/businesses/{id}", h.get)
	mux.Handle("PATCH /api/v1/businesses/{id}", web.RequireAuth(http.HandlerFunc(h.update)))
	mux.Handle("GET /api/v1/businesses/{id}/stats", web.RequireAuth(http.HandlerFunc(h.stats)))
	mux.Handle("POST /api/v1/reviews/{id}/response", web.RequireAuth(http.HandlerFunc(h.respond)))
	mux.Handle("PUT /api/v1/responses/{id}", web.RequireAuth(http.HandlerFunc(h.updateResponse)))
}

// requireMemberOrModerator enforces object-level authorization for business
// management endpoints.
func (h *Handler) requireMemberOrModerator(r *http.Request, businessID uuid.UUID) error {
	p, _ := web.PrincipalFromContext(r.Context())
	if p.HasRole(web.RoleModerator) {
		return nil
	}
	member, err := h.repo.IsMember(r.Context(), businessID, p.UserID)
	if err != nil {
		return err
	}
	if !member {
		return web.ErrForbidden("you do not manage this business")
	}
	return nil
}

type createBusinessRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	var req createBusinessRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	b, err := h.repo.Create(r.Context(), req.Name, req.Description, p.UserID)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusCreated, b)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	b, err := h.repo.Get(r.Context(), id)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, b)
}

type updateBusinessRequest struct {
	Description string `json:"description"`
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	if err := h.requireMemberOrModerator(r, id); err != nil {
		web.RespondError(w, r, err)
		return
	}
	var req updateBusinessRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	b, err := h.repo.UpdateDescription(r.Context(), id, req.Description)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, b)
}

func (h *Handler) stats(w http.ResponseWriter, r *http.Request) {
	id, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	if err := h.requireMemberOrModerator(r, id); err != nil {
		web.RespondError(w, r, err)
		return
	}
	s, err := h.repo.Stats(r.Context(), id)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, s)
}

type responseRequest struct {
	Body string `json:"body"`
}

func (h *Handler) respond(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	reviewID, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	var req responseRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	resp, err := h.repo.CreateResponse(r.Context(), reviewID, p.UserID, req.Body)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusCreated, resp)
}

func (h *Handler) updateResponse(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	id, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	var req responseRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	resp, err := h.repo.UpdateResponse(r.Context(), id, p.UserID, req.Body)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, resp)
}
