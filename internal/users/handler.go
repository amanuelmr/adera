package users

import (
	"net/http"
	"strings"
	"time"

	"github.com/adera-platform/backend/internal/platform/web"
)

// Handler serves the profile endpoints. Credential and session endpoints
// (password change, deactivation) live in internal/auth, which owns sessions.
type Handler struct {
	repo *Repo
}

func NewHandler(repo *Repo) *Handler { return &Handler{repo: repo} }

// Routes registers the profile endpoints on mux.
func (h *Handler) Routes(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/users/me", web.RequireAuth(http.HandlerFunc(h.getMe)))
	mux.Handle("PATCH /api/v1/users/me", web.RequireAuth(http.HandlerFunc(h.updateMe)))
}

// ProfileResponse is the caller's own profile. Email/phone are shown only to
// the owner; public reviewer info elsewhere exposes display name and stats only.
type ProfileResponse struct {
	ID                string    `json:"id"`
	DisplayName       string    `json:"display_name"`
	Email             string    `json:"email,omitempty"`
	Phone             string    `json:"phone,omitempty"`
	Status            string    `json:"status"`
	EmailVerified     bool      `json:"email_verified"`
	PhoneVerified     bool      `json:"phone_verified"`
	PreferredLanguage string    `json:"preferred_language"`
	Roles             []string  `json:"roles"`
	ReviewCount       int       `json:"review_count"`
	CreatedAt         time.Time `json:"created_at"`
}

func (h *Handler) profileResponse(r *http.Request, u User) (ProfileResponse, error) {
	roles, err := h.repo.Roles(r.Context(), u.ID)
	if err != nil {
		return ProfileResponse{}, err
	}
	count, err := h.repo.ReviewCount(r.Context(), u.ID)
	if err != nil {
		return ProfileResponse{}, err
	}
	return ProfileResponse{
		ID:                u.ID.String(),
		DisplayName:       u.DisplayName,
		Email:             u.Email,
		Phone:             u.Phone,
		Status:            u.Status,
		EmailVerified:     u.EmailVerifiedAt != nil,
		PhoneVerified:     u.PhoneVerifiedAt != nil,
		PreferredLanguage: u.PreferredLanguage,
		Roles:             roles,
		ReviewCount:       count,
		CreatedAt:         u.CreatedAt.UTC(),
	}, nil
}

func (h *Handler) getMe(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	u, err := h.repo.GetByID(r.Context(), p.UserID)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	resp, err := h.profileResponse(r, u)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, resp)
}

type updateMeRequest struct {
	DisplayName       *string `json:"display_name"`
	PreferredLanguage *string `json:"preferred_language"`
}

func (h *Handler) updateMe(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	var req updateMeRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	u, err := h.repo.GetByID(r.Context(), p.UserID)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	name := u.DisplayName
	if req.DisplayName != nil {
		name = strings.TrimSpace(*req.DisplayName)
		if err := ValidateDisplayName(name); err != nil {
			web.RespondError(w, r, err)
			return
		}
	}
	lang := u.PreferredLanguage
	if req.PreferredLanguage != nil {
		lang = *req.PreferredLanguage
		if !ValidLanguage(lang) {
			web.RespondError(w, r, web.ErrValidation("invalid profile").
				WithDetail("preferred_language", "must be one of: "+strings.Join(SupportedLanguages, ", ")))
			return
		}
	}
	if err := h.repo.UpdateProfile(r.Context(), p.UserID, name, lang); err != nil {
		web.RespondError(w, r, err)
		return
	}
	u.DisplayName, u.PreferredLanguage = name, lang
	resp, err := h.profileResponse(r, u)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, resp)
}
