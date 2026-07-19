package admin

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/adera-platform/backend/internal/platform/web"
)

// Handler serves administrator endpoints (function-level authorization:
// admin role required for every route).
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Routes(mux *http.ServeMux) {
	adm := web.RequireRole(web.RoleAdmin)
	mux.Handle("POST /api/v1/admin/users/{id}/suspend", adm(http.HandlerFunc(h.suspend)))
	mux.Handle("POST /api/v1/admin/users/{id}/reinstate", adm(http.HandlerFunc(h.reinstate)))
	mux.Handle("POST /api/v1/admin/users/{id}/roles", adm(http.HandlerFunc(h.grantRole)))
	mux.Handle("POST /api/v1/admin/categories", adm(http.HandlerFunc(h.createCategory)))
	mux.Handle("PATCH /api/v1/admin/categories/{id}", adm(http.HandlerFunc(h.updateCategory)))
	mux.Handle("POST /api/v1/admin/categories/{id}/criteria", adm(http.HandlerFunc(h.createCriterion)))
	mux.Handle("PATCH /api/v1/admin/criteria/{id}", adm(http.HandlerFunc(h.updateCriterion)))
	mux.Handle("POST /api/v1/admin/targets/{id}/merge", adm(http.HandlerFunc(h.mergeTargets)))
}

type noteBody struct {
	Note string `json:"note"`
}

func (h *Handler) suspend(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	id, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	var req noteBody
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	if err := h.svc.SuspendUser(r.Context(), id, p.UserID, req.Note); err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, map[string]string{"status": "suspended"})
}

func (h *Handler) reinstate(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	id, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	var req noteBody
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	if err := h.svc.ReinstateUser(r.Context(), id, p.UserID, req.Note); err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, map[string]string{"status": "active"})
}

type roleBody struct {
	Role string `json:"role"`
}

func (h *Handler) grantRole(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	id, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	var req roleBody
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	if err := h.svc.GrantRole(r.Context(), id, p.UserID, req.Role); err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, map[string]string{"status": "granted"})
}

func (h *Handler) createCategory(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	var req CategoryInput
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	id, err := h.svc.CreateCategory(r.Context(), p.UserID, req)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusCreated, map[string]string{"id": id.String()})
}

func (h *Handler) updateCategory(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	id, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	var req CategoryInput
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	if err := h.svc.UpdateCategory(r.Context(), p.UserID, id, req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) createCriterion(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	categoryID, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	var req CriterionInput
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	id, err := h.svc.CreateCriterion(r.Context(), p.UserID, categoryID, req)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusCreated, map[string]string{"id": id.String()})
}

func (h *Handler) updateCriterion(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	id, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	var req CriterionInput
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	if err := h.svc.UpdateCriterion(r.Context(), p.UserID, id, req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, map[string]string{"status": "updated"})
}

type mergeBody struct {
	IntoID string `json:"into_id"`
	Note   string `json:"note"`
}

func (h *Handler) mergeTargets(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	sourceID, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	var req mergeBody
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	destID, err := uuid.Parse(req.IntoID)
	if err != nil {
		web.RespondError(w, r, web.ErrValidation("invalid merge").WithDetail("into_id", "must be a UUID"))
		return
	}
	if err := h.svc.MergeTargets(r.Context(), p.UserID, sourceID, destID, req.Note); err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, map[string]string{"status": "merged", "merged_into": destID.String()})
}
