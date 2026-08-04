package notifications

import (
	"net/http"
	"strconv"

	"github.com/adera-platform/backend/internal/platform/web"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Routes(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/users/me/notifications", web.RequireAuth(http.HandlerFunc(h.list)))
	mux.Handle("GET /api/v1/users/me/notifications/unread-count", web.RequireAuth(http.HandlerFunc(h.unreadCount)))
	mux.Handle("PUT /api/v1/users/me/notifications/read-all", web.RequireAuth(http.HandlerFunc(h.readAll)))
	mux.Handle("PUT /api/v1/users/me/notifications/{id}/read", web.RequireAuth(http.HandlerFunc(h.read)))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	unread := false
	if raw := r.URL.Query().Get("unread"); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			web.RespondError(w, r, web.ErrValidation("invalid filter").WithDetail("unread", "must be a boolean"))
			return
		}
		unread = value
	}
	cursor, err := web.ParseCursor(r)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	items, next, err := h.service.List(r.Context(), p.UserID, unread, cursor, web.ParseLimit(r))
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

func (h *Handler) unreadCount(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	count, err := h.service.UnreadCount(r.Context(), p.UserID)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, map[string]int{"unread_count": count})
}

func (h *Handler) read(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	id, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	if err := h.service.MarkRead(r.Context(), p.UserID, id); err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, map[string]string{"status": "read"})
}

func (h *Handler) readAll(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	count, err := h.service.MarkAllRead(r.Context(), p.UserID)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, map[string]any{"status": "read", "updated": count})
}
