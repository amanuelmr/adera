package auth

import (
	"encoding/hex"
	"net/http"

	"github.com/adera-platform/backend/internal/platform/ratelimit"
	"github.com/adera-platform/backend/internal/platform/security"
	"github.com/adera-platform/backend/internal/platform/web"
	"github.com/adera-platform/backend/internal/users"
)

// Handler exposes the /auth endpoints plus credential endpoints under
// /users/me (this module owns sessions and credentials).
type Handler struct {
	svc      *Service
	limits   Limits
	ipSource web.IPSource
}

// Limits are the rate limiters for the credential endpoints.
type Limits struct {
	// IP bounds sign-in, registration and reset attempts per address. It is
	// looser than Account: carrier NAT puts many phones behind one address.
	IP ratelimit.Limiter
	// Account bounds attempts per account (normalized identifier) and
	// refreshes per refresh token.
	Account ratelimit.Limiter
	// OTP guards the signed-in code and password endpoints, per user.
	OTP ratelimit.Limiter
}

func NewHandler(svc *Service, limits Limits, ipSource web.IPSource) *Handler {
	return &Handler{svc: svc, limits: limits, ipSource: ipSource}
}

func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/register", h.register)
	mux.HandleFunc("POST /api/v1/auth/login", h.login)
	mux.HandleFunc("POST /api/v1/auth/refresh", h.refresh)
	mux.Handle("POST /api/v1/auth/logout", web.RequireAuth(http.HandlerFunc(h.logout)))
	mux.Handle("POST /api/v1/auth/logout-all", web.RequireAuth(http.HandlerFunc(h.logoutAll)))
	mux.Handle("GET /api/v1/auth/sessions", web.RequireAuth(http.HandlerFunc(h.listSessions)))
	mux.Handle("DELETE /api/v1/auth/sessions/{id}", web.RequireAuth(http.HandlerFunc(h.revokeSession)))
	mux.Handle("POST /api/v1/auth/verify/request", web.RequireAuth(http.HandlerFunc(h.verifyRequest)))
	mux.Handle("POST /api/v1/auth/verify/confirm", web.RequireAuth(http.HandlerFunc(h.verifyConfirm)))
	mux.HandleFunc("POST /api/v1/auth/password-reset/request", h.passwordResetRequest)
	mux.HandleFunc("POST /api/v1/auth/password-reset/confirm", h.passwordResetConfirm)
	mux.Handle("POST /api/v1/users/me/password", web.RequireAuth(http.HandlerFunc(h.changePassword)))
	mux.Handle("DELETE /api/v1/users/me", web.RequireAuth(http.HandlerFunc(h.deactivate)))
}

// allowSensitive applies the address limit and, for each identifier given,
// the account limit. Identifiers are normalized first, so spelling a phone
// number another way doesn't buy a fresh bucket.
func (h *Handler) allowSensitive(r *http.Request, identifiers ...string) bool {
	if !h.limits.IP.Allow("ip:" + h.ipSource.ClientIP(r)) {
		return false
	}
	for _, id := range identifiers {
		if id = users.NormalizeIdentifier(id); id != "" && !h.limits.Account.Allow("id:"+id) {
			return false
		}
	}
	return true
}

type registerRequest struct {
	DisplayName       string `json:"display_name"`
	Email             string `json:"email"`
	Phone             string `json:"phone"`
	Password          string `json:"password"`
	PreferredLanguage string `json:"preferred_language"`
}

type authResponse struct {
	User   any       `json:"user"`
	Tokens TokenPair `json:"tokens"`
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	if !h.allowSensitive(r, req.Email, req.Phone) {
		web.RespondError(w, r, web.ErrRateLimited())
		return
	}
	u, pair, err := h.svc.Register(r.Context(), RegisterInput{
		DisplayName: req.DisplayName,
		Email:       req.Email,
		Phone:       req.Phone,
		Password:    req.Password,
		Language:    req.PreferredLanguage,
	}, r.UserAgent())
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusCreated, authResponse{
		User: map[string]any{
			"id":           u.ID.String(),
			"display_name": u.DisplayName,
		},
		Tokens: pair,
	})
}

type loginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	if !h.allowSensitive(r, req.Identifier) {
		web.RespondError(w, r, web.ErrRateLimited())
		return
	}
	u, pair, err := h.svc.Login(r.Context(), req.Identifier, req.Password, r.UserAgent())
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, authResponse{
		User: map[string]any{
			"id":           u.ID.String(),
			"display_name": u.DisplayName,
		},
		Tokens: pair,
	})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	if req.RefreshToken == "" {
		web.RespondError(w, r, web.ErrValidation("invalid request").WithDetail("refresh_token", "required"))
		return
	}
	// Keyed by the token, not the address: carrier NAT puts many phones
	// behind one address, and each refreshes on its own schedule. Guessing a
	// 256-bit token gains nothing from more attempts.
	if !h.limits.Account.Allow("refresh:" + hex.EncodeToString(security.HashToken(req.RefreshToken)[:16])) {
		web.RespondError(w, r, web.ErrRateLimited())
		return
	}
	pair, err := h.svc.Refresh(r.Context(), req.RefreshToken, r.UserAgent())
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, pair)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	if err := h.svc.Logout(r.Context(), p.SessionID); err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

func (h *Handler) logoutAll(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	if err := h.svc.LogoutAll(r.Context(), p.UserID); err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, map[string]string{"status": "logged_out_all"})
}

func (h *Handler) listSessions(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	sessions, err := h.svc.ListSessions(r.Context(), p.UserID, p.SessionID)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, sessions)
}

func (h *Handler) revokeSession(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	id, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	if err := h.svc.RevokeSession(r.Context(), p.UserID, id); err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, map[string]string{"status": "revoked"})
}

type verifyRequestBody struct {
	Channel string `json:"channel"`
}

func (h *Handler) verifyRequest(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	var req verifyRequestBody
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	if !h.limits.OTP.Allow("verify:" + p.UserID.String()) {
		web.RespondError(w, r, web.ErrRateLimited())
		return
	}
	if err := h.svc.RequestVerification(r.Context(), p.UserID, req.Channel); err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, map[string]string{"status": "code_sent"})
}

type verifyConfirmBody struct {
	Channel string `json:"channel"`
	Code    string `json:"code"`
}

func (h *Handler) verifyConfirm(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	var req verifyConfirmBody
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	if !h.limits.OTP.Allow("confirm:" + p.UserID.String()) {
		web.RespondError(w, r, web.ErrRateLimited())
		return
	}
	if err := h.svc.ConfirmVerification(r.Context(), p.UserID, req.Channel, req.Code); err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, map[string]string{"status": "verified"})
}

type passwordResetRequestBody struct {
	Identifier string `json:"identifier"`
}

func (h *Handler) passwordResetRequest(w http.ResponseWriter, r *http.Request) {
	var req passwordResetRequestBody
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	if !h.allowSensitive(r, req.Identifier) {
		web.RespondError(w, r, web.ErrRateLimited())
		return
	}
	if err := h.svc.RequestPasswordReset(r.Context(), req.Identifier); err != nil {
		web.RespondError(w, r, err)
		return
	}
	// Uniform response regardless of account existence.
	web.Respond(w, http.StatusOK, map[string]string{"status": "if_account_exists_code_sent"})
}

type passwordResetConfirmBody struct {
	Identifier  string `json:"identifier"`
	Code        string `json:"code"`
	NewPassword string `json:"new_password"`
}

func (h *Handler) passwordResetConfirm(w http.ResponseWriter, r *http.Request) {
	var req passwordResetConfirmBody
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	if !h.allowSensitive(r, req.Identifier) {
		web.RespondError(w, r, web.ErrRateLimited())
		return
	}
	if err := h.svc.ConfirmPasswordReset(r.Context(), req.Identifier, req.Code, req.NewPassword); err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, map[string]string{"status": "password_reset"})
}

type changePasswordBody struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	var req changePasswordBody
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	if !h.limits.OTP.Allow("chpw:" + p.UserID.String()) {
		web.RespondError(w, r, web.ErrRateLimited())
		return
	}
	if err := h.svc.ChangePassword(r.Context(), p.UserID, p.SessionID, req.CurrentPassword, req.NewPassword); err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, map[string]string{"status": "password_changed"})
}

func (h *Handler) deactivate(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	if err := h.svc.Deactivate(r.Context(), p.UserID); err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, map[string]string{"status": "deactivated"})
}
