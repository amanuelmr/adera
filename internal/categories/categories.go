// Package categories serves the category taxonomy and category-specific
// review criteria. Criteria are data, not code: each category's rating
// dimensions, labels (with translations), scales, and required flags live in
// the database and are managed by administrators (internal/admin).
package categories

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adera-platform/backend/internal/platform/web"
)

// Category is a reviewable business category.
type Category struct {
	ID                      uuid.UUID         `json:"id"`
	Code                    string            `json:"code"`
	Name                    string            `json:"name"`
	NameTranslations        map[string]string `json:"name_translations"`
	Description             string            `json:"description"`
	DescriptionTranslations map[string]string `json:"description_translations"`
	SortOrder               int               `json:"sort_order"`
	Active                  bool              `json:"active"`
	CreatedAt               time.Time         `json:"created_at"`
	UpdatedAt               time.Time         `json:"updated_at"`
}

// Criterion is one structured rating dimension for a category.
type Criterion struct {
	ID                uuid.UUID         `json:"id"`
	CategoryID        uuid.UUID         `json:"category_id"`
	Code              string            `json:"code"`
	Name              string            `json:"name"`
	LabelTranslations map[string]string `json:"label_translations"`
	Description       string            `json:"description"`
	Required          bool              `json:"required"`
	ScaleMin          int               `json:"scale_min"`
	ScaleMax          int               `json:"scale_max"`
	SortOrder         int               `json:"sort_order"`
	Active            bool              `json:"active"`
}

// Repo provides category persistence.
type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

const categoryColumns = `id, code, name, name_translations, description, description_translations,
	sort_order, active, created_at, updated_at`

func scanCategory(row pgx.Row) (Category, error) {
	var c Category
	err := row.Scan(&c.ID, &c.Code, &c.Name, &c.NameTranslations, &c.Description,
		&c.DescriptionTranslations, &c.SortOrder, &c.Active, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Category{}, web.ErrNotFound("category")
	}
	if err != nil {
		return Category{}, fmt.Errorf("scanning category: %w", err)
	}
	return c, nil
}

// List returns categories; inactive ones only when includeInactive.
func (r *Repo) List(ctx context.Context, includeInactive bool) ([]Category, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+categoryColumns+` FROM categories
		WHERE active OR $1 ORDER BY sort_order, name`, includeInactive)
	if err != nil {
		return nil, fmt.Errorf("querying categories: %w", err)
	}
	defer rows.Close()
	var out []Category
	for rows.Next() {
		c, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating categories: %w", err)
	}
	return out, nil
}

// GetByIDOrCode resolves either a UUID or a stable code.
func (r *Repo) GetByIDOrCode(ctx context.Context, idOrCode string) (Category, error) {
	if id, err := uuid.Parse(idOrCode); err == nil {
		return scanCategory(r.pool.QueryRow(ctx, `SELECT `+categoryColumns+` FROM categories WHERE id = $1`, id))
	}
	return scanCategory(r.pool.QueryRow(ctx, `SELECT `+categoryColumns+` FROM categories WHERE code = $1`, idOrCode))
}

const criterionColumns = `id, category_id, code, name, label_translations, description,
	required, scale_min, scale_max, sort_order, active`

// Criteria returns a category's criteria, active-only unless includeInactive.
func (r *Repo) Criteria(ctx context.Context, categoryID uuid.UUID, includeInactive bool) ([]Criterion, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+criterionColumns+` FROM category_criteria
		WHERE category_id = $1 AND (active OR $2)
		ORDER BY sort_order, code`, categoryID, includeInactive)
	if err != nil {
		return nil, fmt.Errorf("querying criteria: %w", err)
	}
	defer rows.Close()
	var out []Criterion
	for rows.Next() {
		var c Criterion
		if err := rows.Scan(&c.ID, &c.CategoryID, &c.Code, &c.Name, &c.LabelTranslations,
			&c.Description, &c.Required, &c.ScaleMin, &c.ScaleMax, &c.SortOrder, &c.Active); err != nil {
			return nil, fmt.Errorf("scanning criterion: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating criteria: %w", err)
	}
	return out, nil
}

// Handler serves the public read endpoints.
type Handler struct {
	repo *Repo
}

func NewHandler(repo *Repo) *Handler { return &Handler{repo: repo} }

func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/categories", h.list)
	mux.HandleFunc("GET /api/v1/categories/{idOrCode}", h.get)
	mux.HandleFunc("GET /api/v1/categories/{idOrCode}/criteria", h.criteria)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	cats, err := h.repo.List(r.Context(), false)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, cats)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	c, err := h.repo.GetByIDOrCode(r.Context(), r.PathValue("idOrCode"))
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	if !c.Active {
		web.RespondError(w, r, web.ErrNotFound("category"))
		return
	}
	web.Respond(w, http.StatusOK, c)
}

func (h *Handler) criteria(w http.ResponseWriter, r *http.Request) {
	c, err := h.repo.GetByIDOrCode(r.Context(), r.PathValue("idOrCode"))
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	criteria, err := h.repo.Criteria(r.Context(), c.ID, false)
	if err != nil {
		web.RespondError(w, r, err)
		return
	}
	web.Respond(w, http.StatusOK, criteria)
}
