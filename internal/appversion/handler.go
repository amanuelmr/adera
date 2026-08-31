package appversion

import (
	"net/http"

	"github.com/adera-platform/backend/internal/platform/web"
)

// Handler serves the client version gate.
type Handler struct {
	policy Policy
}

func NewHandler(policy Policy) *Handler { return &Handler{policy: policy} }

func (h *Handler) Routes(mux *http.ServeMux) {
	// Public on purpose: a client that is too old to authenticate still has
	// to be able to discover that it must update.
	mux.HandleFunc("GET /api/v1/app/version", h.get)
}

// Response describes the gate for one platform. update_required and
// update_available are only present when the caller supplies its own
// version, so absent never reads as "you are up to date".
type Response struct {
	Platform         string `json:"platform"`
	MinimumSupported string `json:"minimum_supported,omitempty"`
	Latest           string `json:"latest,omitempty"`
	StoreURL         string `json:"store_url,omitempty"`
	UpdateRequired   *bool  `json:"update_required,omitempty"`
	UpdateAvailable  *bool  `json:"update_available,omitempty"`
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	platform := q.Get("platform")
	if platform == "" {
		platform = PlatformAndroid
	}
	gate, ok := h.policy.Gate(platform)
	if !ok {
		web.RespondError(w, r, web.ErrValidation("invalid request").
			WithDetail("platform", "must be android or ios"))
		return
	}

	resp := Response{
		Platform:         platform,
		MinimumSupported: gate.MinimumSupported,
		Latest:           gate.Latest,
		StoreURL:         gate.StoreURL,
	}

	if current := q.Get("version"); current != "" {
		if _, err := Parse(current); err != nil {
			web.RespondError(w, r, web.ErrValidation("invalid request").
				WithDetail("version", "must be a dotted numeric version such as 1.4.2"))
			return
		}
		if gate.MinimumSupported != "" {
			cmp, err := Compare(current, gate.MinimumSupported)
			if err != nil {
				web.RespondError(w, r, err)
				return
			}
			required := cmp < 0
			resp.UpdateRequired = &required
		}
		if gate.Latest != "" {
			cmp, err := Compare(current, gate.Latest)
			if err != nil {
				web.RespondError(w, r, err)
				return
			}
			available := cmp < 0
			resp.UpdateAvailable = &available
		}
	}
	web.Respond(w, http.StatusOK, resp)
}
