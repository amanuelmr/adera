package privacy

import (
	"net/http"

	"github.com/adera-platform/backend/internal/platform/web"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Routes(mux *http.ServeMux) {
	auth := web.RequireAuth
	admin := web.RequireRole(web.RoleAdmin)
	mux.Handle("GET /api/v1/users/me/data-export", auth(http.HandlerFunc(h.export)))
	mux.Handle("POST /api/v1/users/me/erasure-requests", auth(http.HandlerFunc(h.createRequest)))
	mux.Handle("GET /api/v1/users/me/erasure-requests", auth(http.HandlerFunc(h.listMine)))
	mux.Handle("DELETE /api/v1/users/me/erasure-requests/{id}", auth(http.HandlerFunc(h.cancelRequest)))
	mux.Handle("GET /api/v1/admin/privacy/erasure-requests", admin(http.HandlerFunc(h.listAdmin)))
	mux.Handle("POST /api/v1/admin/privacy/erasure-requests/{id}/decision", admin(http.HandlerFunc(h.decide)))
}

func (h *Handler) export(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	export, err := h.service.Export(r.Context(), p.UserID)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="adera-data-export.json"`)
	web.Respond(w, http.StatusOK, export)
}

type createRequestBody struct {
	Reason string `json:"reason"`
}

func (h *Handler) createRequest(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	var req createRequestBody
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	item, err := h.service.CreateRequest(r.Context(), p.UserID, req.Reason)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusCreated, item)
}

func pageMeta(next *web.Cursor) web.Meta {
	meta := web.Meta{HasMore: next != nil}
	if next != nil {
		meta.NextCursor = next.Encode()
	}
	return meta
}

func (h *Handler) listMine(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	cursor, err := web.ParseCursor(r)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	items, next, err := h.service.ListMine(r.Context(), p.UserID, cursor, web.ParseLimit(r))
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.RespondPage(w, http.StatusOK, items, pageMeta(next))
}

func (h *Handler) cancelRequest(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	id, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	item, err := h.service.CancelRequest(r.Context(), p.UserID, id)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, item)
}

func (h *Handler) listAdmin(w http.ResponseWriter, r *http.Request) {
	cursor, err := web.ParseCursor(r)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	items, next, err := h.service.ListForAdmin(r.Context(), r.URL.Query().Get("status"), cursor, web.ParseLimit(r))
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.RespondPage(w, http.StatusOK, items, pageMeta(next))
}

type decisionBody struct {
	Status string `json:"status"`
	Note   string `json:"note"`
}

func (h *Handler) decide(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	id, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	var req decisionBody
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	item, err := h.service.DecideRequest(r.Context(), id, p.UserID, req.Status, req.Note)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, item)
}
