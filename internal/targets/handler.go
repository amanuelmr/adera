package targets

import (
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/adera-platform/backend/internal/businesses"
	"github.com/adera-platform/backend/internal/platform/web"
	"github.com/adera-platform/backend/internal/users"
)

// Handler serves target endpoints.
type Handler struct {
	repo      *Repo
	bizRepo   *businesses.Repo
	usersRepo *users.Repo
}

func NewHandler(repo *Repo, bizRepo *businesses.Repo, usersRepo *users.Repo) *Handler {
	return &Handler{repo: repo, bizRepo: bizRepo, usersRepo: usersRepo}
}

func (h *Handler) Routes(mux *http.ServeMux) {
	mux.Handle("POST /api/v1/targets", web.RequireAuth(http.HandlerFunc(h.create)))
	mux.HandleFunc("GET /api/v1/targets", h.browse)
	mux.HandleFunc("GET /api/v1/targets/top-rated", h.topRated)
	mux.HandleFunc("GET /api/v1/targets/trending", h.trending)
	mux.HandleFunc("GET /api/v1/targets/nearby", h.nearby)
	mux.HandleFunc("GET /api/v1/targets/{idOrSlug}", h.get)
	mux.Handle("PATCH /api/v1/targets/{idOrSlug}", web.RequireAuth(http.HandlerFunc(h.update)))
	mux.Handle("POST /api/v1/targets/{idOrSlug}/edit-suggestions", web.RequireAuth(http.HandlerFunc(h.suggestEdit)))
}

type createTargetRequest struct {
	TargetType  string            `json:"target_type"`
	CategoryID  string            `json:"category_id"`
	BusinessID  *string           `json:"business_id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	CityID      *string           `json:"city_id"`
	AreaID      *string           `json:"area_id"`
	AddressText string            `json:"address_text"`
	Latitude    *float64          `json:"latitude"`
	Longitude   *float64          `json:"longitude"`
	OnlineOnly  bool              `json:"online_only"`
	Phone       string            `json:"phone"`
	Website     string            `json:"website"`
	SocialLinks map[string]string `json:"social_links"`
	Aliases     []string          `json:"aliases"`
}

func parseOptionalUUID(s *string, field string) (*uuid.UUID, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	id, err := uuid.Parse(*s)
	if err != nil {
		return nil, web.ErrValidation("invalid request").WithDetail(field, "must be a UUID")
	}
	return &id, nil
}

func validateWebsite(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if len(raw) > 300 {
		return web.ErrValidation("invalid target").WithDetail("website", "URL too long")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return web.ErrValidation("invalid target").WithDetail("website", "must be a valid https URL")
	}
	return nil
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	var req createTargetRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if l := len([]rune(req.Name)); l < 2 || l > 160 {
		web.RespondError(w, r, web.ErrValidation("invalid target").WithDetail("name", "must be 2-160 characters"))
		return
	}
	if !ValidTargetType(req.TargetType) {
		web.RespondError(w, r, web.ErrValidation("invalid target").WithDetail("target_type", "unsupported type"))
		return
	}
	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		web.RespondError(w, r, web.ErrValidation("invalid target").WithDetail("category_id", "must be a UUID"))
		return
	}
	if len([]rune(req.Description)) > 2000 {
		web.RespondError(w, r, web.ErrValidation("invalid target").WithDetail("description", "too long"))
		return
	}
	if err := ValidateSocialLinks(req.SocialLinks); err != nil {
		web.RespondError(w, r, err)
		return
	}
	if req.Phone != "" {
		normalized := users.NormalizePhone(req.Phone)
		if normalized == "" {
			web.RespondError(w, r, web.ErrValidation("invalid target").WithDetail("phone", "not a valid phone number"))
			return
		}
		req.Phone = normalized
	}
	if err := validateWebsite(req.Website); err != nil {
		web.RespondError(w, r, err)
		return
	}
	if len(req.Aliases) > 5 {
		web.RespondError(w, r, web.ErrValidation("invalid target").WithDetail("aliases", "at most 5 aliases"))
		return
	}
	aliases := make([]string, 0, len(req.Aliases))
	for _, a := range req.Aliases {
		a = strings.TrimSpace(a)
		if l := len([]rune(a)); l < 2 || l > 160 {
			web.RespondError(w, r, web.ErrValidation("invalid target").WithDetail("aliases", "each alias must be 2-160 characters"))
			return
		}
		aliases = append(aliases, a)
	}
	businessID, err := parseOptionalUUID(req.BusinessID, "business_id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	cityID, err := parseOptionalUUID(req.CityID, "city_id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	areaID, err := parseOptionalUUID(req.AreaID, "area_id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	if (req.Latitude == nil) != (req.Longitude == nil) {
		web.RespondError(w, r, web.ErrValidation("invalid target").WithDetail("latitude", "latitude and longitude must be provided together"))
		return
	}

	t, err := h.repo.Create(r.Context(), CreateInput{
		TargetType:  req.TargetType,
		CategoryID:  categoryID,
		BusinessID:  businessID,
		Name:        req.Name,
		Description: strings.TrimSpace(req.Description),
		CityID:      cityID,
		AreaID:      areaID,
		AddressText: strings.TrimSpace(req.AddressText),
		Latitude:    req.Latitude,
		Longitude:   req.Longitude,
		OnlineOnly:  req.OnlineOnly,
		Phone:       req.Phone,
		Website:     strings.TrimSpace(req.Website),
		SocialLinks: req.SocialLinks,
		Aliases:     aliases,
		CreatedBy:   p.UserID,
		AutoPublish: p.HasRole(web.RoleModerator),
	})
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusCreated, t)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	t, err := h.repo.GetByIDOrSlug(r.Context(), r.PathValue("idOrSlug"))
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	if t.ModerationStatus != ModerationPublished {
		p, ok := web.PrincipalFromContext(r.Context())
		if !ok || !p.HasRole(web.RoleModerator) {
			web.RespondError(w, r, web.ErrNotFound("target"))
			return
		}
	}
	web.Respond(w, http.StatusOK, t)
}

type updateTargetRequest struct {
	Description *string           `json:"description"`
	AddressText *string           `json:"address_text"`
	Phone       *string           `json:"phone"`
	Website     *string           `json:"website"`
	CityID      *string           `json:"city_id"`
	AreaID      *string           `json:"area_id"`
	OnlineOnly  *bool             `json:"online_only"`
	SocialLinks map[string]string `json:"social_links"`
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	t, err := h.repo.GetByIDOrSlug(r.Context(), r.PathValue("idOrSlug"))
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	// Object-level authorization: members of the owning business or moderators.
	if !p.HasRole(web.RoleModerator) {
		if t.BusinessID == nil {
			web.RespondError(w, r, web.ErrForbidden("this target has no claimed business; suggest an edit instead"))
			return
		}
		member, err := h.bizRepo.IsMember(r.Context(), *t.BusinessID, p.UserID)
		if err != nil {
			web.RespondError(w, r, err)
			return
		}
		if !member {
			web.RespondError(w, r, web.ErrForbidden("you do not manage this target"))
			return
		}
	}

	var req updateTargetRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	if req.Description != nil && len([]rune(*req.Description)) > 2000 {
		web.RespondError(w, r, web.ErrValidation("invalid target").WithDetail("description", "too long"))
		return
	}
	if req.SocialLinks != nil {
		if err := ValidateSocialLinks(req.SocialLinks); err != nil {
			web.RespondError(w, r, err)
			return
		}
	}
	if req.Phone != nil && *req.Phone != "" {
		normalized := users.NormalizePhone(*req.Phone)
		if normalized == "" {
			web.RespondError(w, r, web.ErrValidation("invalid target").WithDetail("phone", "not a valid phone number"))
			return
		}
		req.Phone = &normalized
	}
	if req.Website != nil {
		if err := validateWebsite(*req.Website); err != nil {
			web.RespondError(w, r, err)
			return
		}
		trimmed := strings.TrimSpace(*req.Website)
		req.Website = &trimmed
	}
	cityID, err := parseOptionalUUID(req.CityID, "city_id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	areaID, err := parseOptionalUUID(req.AreaID, "area_id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}

	updated, err := h.repo.Update(r.Context(), t.ID, UpdateInput{
		Description: req.Description,
		AddressText: req.AddressText,
		Phone:       req.Phone,
		Website:     req.Website,
		CityID:      cityID,
		AreaID:      areaID,
		OnlineOnly:  req.OnlineOnly,
		SocialLinks: req.SocialLinks,
	})
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, updated)
}

// parseBrowseFilter reads shared listing filters from the query string.
func ParseBrowseFilter(r *http.Request) (BrowseFilter, error) {
	q := r.URL.Query()
	var f BrowseFilter
	parse := func(name string) (*uuid.UUID, error) {
		v := q.Get(name)
		if v == "" {
			return nil, nil
		}
		id, err := uuid.Parse(v)
		if err != nil {
			return nil, web.ErrValidation("invalid filter").WithDetail(name, "must be a UUID")
		}
		return &id, nil
	}
	var err error
	if f.CategoryID, err = parse("category"); err != nil {
		return f, err
	}
	if f.CityID, err = parse("city"); err != nil {
		return f, err
	}
	if f.AreaID, err = parse("area"); err != nil {
		return f, err
	}
	if f.BusinessID, err = parse("business"); err != nil {
		return f, err
	}
	if t := q.Get("type"); t != "" {
		if !ValidTargetType(t) {
			return f, web.ErrValidation("invalid filter").WithDetail("type", "unsupported type")
		}
		f.TargetType = t
	}
	if v := q.Get("min_rating"); v != "" {
		mr, err := strconv.ParseFloat(v, 64)
		if err != nil || mr < 1 || mr > 5 {
			return f, web.ErrValidation("invalid filter").WithDetail("min_rating", "must be between 1 and 5")
		}
		f.MinRating = mr
	}
	if v := q.Get("verified"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return f, web.ErrValidation("invalid filter").WithDetail("verified", "must be a boolean")
		}
		f.Verified = &b
	}
	if v := q.Get("online_only"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return f, web.ErrValidation("invalid filter").WithDetail("online_only", "must be a boolean")
		}
		f.OnlineOnly = &b
	}
	return f, nil
}

func (h *Handler) browse(w http.ResponseWriter, r *http.Request) {
	f, err := ParseBrowseFilter(r)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	cursor, err := web.ParseCursor(r)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	limit := web.ParseLimit(r)
	items, next, err := h.repo.Browse(r.Context(), f, cursor, limit)
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

func (h *Handler) topRated(w http.ResponseWriter, r *http.Request) {
	f, err := ParseBrowseFilter(r)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	items, err := h.repo.TopRated(r.Context(), f, web.ParseLimit(r))
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, items)
}

func (h *Handler) trending(w http.ResponseWriter, r *http.Request) {
	f, err := ParseBrowseFilter(r)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	items, err := h.repo.Trending(r.Context(), f, web.ParseLimit(r))
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, items)
}

// Nearby search bounds. The maximum keeps a single query from scanning the
// whole table, and matches what a phone can usefully show in a list.
const (
	DefaultNearbyRadiusKm = 5.0
	MaxNearbyRadiusKm     = 50.0
)

// ParseNearbyPoint reads the required lat/lng pair and the optional radius.
func ParseNearbyPoint(r *http.Request) (lat, lng, radiusKm float64, err error) {
	q := r.URL.Query()
	coord := func(name string, limit float64) (float64, error) {
		raw := q.Get(name)
		if raw == "" {
			return 0, web.ErrValidation("invalid filter").WithDetail(name, "is required")
		}
		v, parseErr := strconv.ParseFloat(raw, 64)
		if parseErr != nil || math.IsNaN(v) || v < -limit || v > limit {
			return 0, web.ErrValidation("invalid filter").
				WithDetail(name, fmt.Sprintf("must be a number between %g and %g", -limit, limit))
		}
		return v, nil
	}
	if lat, err = coord("lat", 90); err != nil {
		return 0, 0, 0, err
	}
	if lng, err = coord("lng", 180); err != nil {
		return 0, 0, 0, err
	}
	radiusKm = DefaultNearbyRadiusKm
	if raw := q.Get("radius_km"); raw != "" {
		v, parseErr := strconv.ParseFloat(raw, 64)
		if parseErr != nil || !(v > 0) || v > MaxNearbyRadiusKm {
			return 0, 0, 0, web.ErrValidation("invalid filter").
				WithDetail("radius_km", fmt.Sprintf("must be greater than 0 and at most %g", MaxNearbyRadiusKm))
		}
		radiusKm = v
	}
	return lat, lng, radiusKm, nil
}

func (h *Handler) nearby(w http.ResponseWriter, r *http.Request) {
	f, err := ParseBrowseFilter(r)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	lat, lng, radiusKm, err := ParseNearbyPoint(r)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	items, err := h.repo.Nearby(r.Context(), f, lat, lng, radiusKm, web.ParseLimit(r))
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, items)
}

type suggestEditRequest struct {
	Changes map[string]any `json:"changes"`
	Note    string         `json:"note"`
}

func (h *Handler) suggestEdit(w http.ResponseWriter, r *http.Request) {
	p, _ := web.PrincipalFromContext(r.Context())
	t, err := h.repo.GetByIDOrSlug(r.Context(), r.PathValue("idOrSlug"))
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	var req suggestEditRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		web.RespondError(w, r, err)
		return
	}
	if len([]rune(req.Note)) > 1000 {
		web.RespondError(w, r, web.ErrValidation("invalid suggestion").WithDetail("note", "too long"))
		return
	}
	id, err := h.repo.CreateEditSuggestion(r.Context(), t.ID, p.UserID, req.Changes, strings.TrimSpace(req.Note))
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusCreated, map[string]string{"id": id.String(), "status": "open"})
}
