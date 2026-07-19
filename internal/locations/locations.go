// Package locations serves cities and areas (neighborhoods). Addis Ababa is
// seeded with its common areas; the model supports other Ethiopian cities.
package locations

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adera-platform/backend/internal/platform/web"
)

// City is a supported city.
type City struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	NameAm    string    `json:"name_am"`
	Country   string    `json:"country"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
}

// Area is a neighborhood within a city.
type Area struct {
	ID     uuid.UUID `json:"id"`
	CityID uuid.UUID `json:"city_id"`
	Name   string    `json:"name"`
	NameAm string    `json:"name_am"`
}

// Repo provides location reads.
type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

// Cities lists active cities.
func (r *Repo) Cities(ctx context.Context) ([]City, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, name_am, country, sort_order, created_at
		FROM cities WHERE active ORDER BY sort_order, name`)
	if err != nil {
		return nil, fmt.Errorf("querying cities: %w", err)
	}
	defer rows.Close()
	var out []City
	for rows.Next() {
		var c City
		if err := rows.Scan(&c.ID, &c.Name, &c.NameAm, &c.Country, &c.SortOrder, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning city: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating cities: %w", err)
	}
	return out, nil
}

// Areas lists a city's active areas.
func (r *Repo) Areas(ctx context.Context, cityID uuid.UUID) ([]Area, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, city_id, name, name_am FROM areas
		WHERE city_id = $1 AND active ORDER BY sort_order, name`, cityID)
	if err != nil {
		return nil, fmt.Errorf("querying areas: %w", err)
	}
	defer rows.Close()
	var out []Area
	for rows.Next() {
		var a Area
		if err := rows.Scan(&a.ID, &a.CityID, &a.Name, &a.NameAm); err != nil {
			return nil, fmt.Errorf("scanning area: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating areas: %w", err)
	}
	return out, nil
}

// Handler serves location endpoints.
type Handler struct {
	repo *Repo
}

func NewHandler(repo *Repo) *Handler { return &Handler{repo: repo} }

func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/locations/cities", h.cities)
	mux.HandleFunc("GET /api/v1/locations/cities/{id}/areas", h.areas)
}

func (h *Handler) cities(w http.ResponseWriter, r *http.Request) {
	cities, err := h.repo.Cities(r.Context())
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, cities)
}

func (h *Handler) areas(w http.ResponseWriter, r *http.Request) {
	id, err := web.ParseUUID(r, "id")
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	areas, err := h.repo.Areas(r.Context(), id)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, areas)
}
