package pages

import (
	"bytes"
	"encoding/xml"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/adera-platform/backend/internal/categories"
	"github.com/adera-platform/backend/internal/platform/web"
	"github.com/adera-platform/backend/internal/ratings"
	"github.com/adera-platform/backend/internal/targets"
)

// Discovery pages: a home page and one page per category, so the places
// people share links to can also be found from a search engine or by
// browsing, without the app (docs/mobile-plan.md §1).

const (
	homeListSize      = 10
	categoryPageSize  = 20
	sitemapMaxTargets = 50000 - 100 // the sitemap limit, less room for the fixed pages
)

// placeItem is one row of a place list.
type placeItem struct {
	Name string
	URL  string
	Meta string
}

type categoryLink struct {
	Name string
	URL  string
}

// newPlaceItem builds a list row. Like the place page, it shows no average
// below the minimum sample: a list mustn't show a rating the place page
// would withhold. extra is appended to the meta line (e.g. recent activity).
func newPlaceItem(b base, t targets.Target, extra ...string) placeItem {
	var parts []string
	if t.ReviewCount >= ratings.MinSampleForTrend && t.AverageRating != nil {
		parts = append(parts, b.T("place.average", "rating", strconv.FormatFloat(*t.AverageRating, 'f', 1, 64)))
	}
	parts = append(parts, b.T("target.reviews", "count", t.ReviewCount))
	parts = append(parts, extra...)
	return placeItem{Name: t.Name, URL: b.Link("/t/"+t.Slug, nil), Meta: strings.Join(parts, " · ")}
}

func categoryName(c categories.Category, lang string) string {
	if name := c.NameTranslations[lang]; name != "" {
		return name
	}
	return c.Name
}

// ---- home ----

type homePage struct {
	base
	Categories []categoryLink
	TopRated   []placeItem
	Trending   []placeItem
}

func (h *Handler) home(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	p := homePage{base: h.newBase(r, "/")}
	p.Title = p.T("tagline")
	p.Description = p.T("home.description")

	cats, err := h.deps.Categories.List(ctx, false)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	for _, c := range cats {
		p.Categories = append(p.Categories, categoryLink{Name: categoryName(c, p.Lang), URL: p.Link("/c/"+url.PathEscape(c.Code), nil)})
	}

	top, err := h.deps.Targets.TopRated(ctx, targets.BrowseFilter{}, homeListSize)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	for _, t := range top {
		p.TopRated = append(p.TopRated, newPlaceItem(p.base, t.Target))
	}

	trending, err := h.deps.Targets.Trending(ctx, targets.BrowseFilter{}, homeListSize)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	for _, t := range trending {
		p.Trending = append(p.Trending, newPlaceItem(p.base, t.Target, p.T("home.thisMonth", "count", t.RecentReviewCount)))
	}
	h.render(w, r, "home", http.StatusOK, p)
}

// ---- category ----

type categoryPage struct {
	base
	Name     string
	Intro    string
	TopRated []placeItem
	Places   []placeItem
	NextURL  string
}

func (h *Handler) category(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	c, err := h.deps.Categories.GetByIDOrCode(ctx, r.PathValue("code"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	// Only the code form is a page; a UUID would be a second URL for it.
	if !c.Active || c.Code != r.PathValue("code") {
		h.notFound(w, r)
		return
	}

	path := "/c/" + url.PathEscape(c.Code)
	p := categoryPage{base: h.newBase(r, path)}
	p.Name = categoryName(c, p.Lang)
	p.Intro = c.DescriptionTranslations[p.Lang]
	if p.Intro == "" {
		p.Intro = c.Description
	}
	p.Title = p.Name
	p.Description = firstNonEmpty(p.Intro, p.T("category.description", "name", p.Name))

	filter := targets.BrowseFilter{CategoryID: &c.ID}
	cursor, err := web.ParseCursor(r)
	if err != nil {
		cursor = nil // a mangled ?cursor= just means the first page
	}
	// The ranking leads the first page only; later pages continue the
	// full list.
	if cursor == nil {
		top, err := h.deps.Targets.TopRated(ctx, filter, homeListSize)
		if err != nil {
			h.fail(w, r, err)
			return
		}
		for _, t := range top {
			p.TopRated = append(p.TopRated, newPlaceItem(p.base, t.Target))
		}
	}

	places, next, err := h.deps.Targets.Browse(ctx, filter, cursor, categoryPageSize)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	for _, t := range places {
		p.Places = append(p.Places, newPlaceItem(p.base, t))
	}
	if next != nil {
		p.NextURL = p.Link(path, url.Values{"cursor": {next.Encode()}})
	}
	h.render(w, r, "category", http.StatusOK, p)
}

// ---- structured data ----

// structuredData is the schema.org JSON-LD for a place page. Rendered inside
// <script type="application/ld+json">, which html/template escapes as JSON.
type structuredData struct {
	Context         string           `json:"@context"`
	Type            string           `json:"@type"`
	Name            string           `json:"name"`
	URL             string           `json:"url"`
	Description     string           `json:"description,omitempty"`
	Telephone       string           `json:"telephone,omitempty"`
	Address         *postalAddress   `json:"address,omitempty"`
	AggregateRating *aggregateRating `json:"aggregateRating,omitempty"`
}

type postalAddress struct {
	Type           string `json:"@type"`
	StreetAddress  string `json:"streetAddress,omitempty"`
	AddressCountry string `json:"addressCountry"`
}

type aggregateRating struct {
	Type        string `json:"@type"`
	RatingValue string `json:"ratingValue"`
	ReviewCount int    `json:"reviewCount"`
	BestRating  int    `json:"bestRating"`
	WorstRating int    `json:"worstRating"`
}

// targetStructuredData describes the place for search engines. The rating
// is included only when the page itself shows one: structured data must not
// claim an aggregate the reader can't see.
func (h *Handler) targetStructuredData(t targets.Target, stats ratings.TargetStats, showAggregate bool) *structuredData {
	d := &structuredData{
		Context:     "https://schema.org",
		Type:        "LocalBusiness",
		Name:        t.Name,
		URL:         h.absURL("/t/" + t.Slug),
		Description: excerpt(t.Description, 300),
		Telephone:   t.Phone,
	}
	if t.OnlineOnly {
		d.Type = "Organization"
	} else {
		d.Address = &postalAddress{Type: "PostalAddress", StreetAddress: t.AddressText, AddressCountry: "ET"}
	}
	if showAggregate && stats.Average != nil {
		d.AggregateRating = &aggregateRating{
			Type:        "AggregateRating",
			RatingValue: strconv.FormatFloat(*stats.Average, 'f', 1, 64),
			ReviewCount: stats.ReviewCount,
			BestRating:  5,
			WorstRating: 1,
		}
	}
	return d
}

// ---- sitemap, robots ----

type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

type urlSet struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

func (h *Handler) sitemap(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	set := urlSet{XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	set.URLs = append(set.URLs, sitemapURL{Loc: h.absURL("/")}, sitemapURL{Loc: h.absURL("/trust")})

	cats, err := h.deps.Categories.List(ctx, false)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	for _, c := range cats {
		set.URLs = append(set.URLs, sitemapURL{Loc: h.absURL("/c/" + url.PathEscape(c.Code))})
	}
	entries, err := h.deps.Targets.SitemapEntries(ctx, sitemapMaxTargets)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	for _, e := range entries {
		set.URLs = append(set.URLs, sitemapURL{
			Loc:     h.absURL("/t/" + url.PathEscape(e.Slug)),
			LastMod: e.UpdatedAt.UTC().Format(time.DateOnly),
		})
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	if err := xml.NewEncoder(&buf).Encode(set); err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write(buf.Bytes())
}

// robots.txt: the public pages are meant to be found; the JSON API and the
// banner-dismiss redirect aren't pages.
func (h *Handler) robots(w http.ResponseWriter, _ *http.Request) {
	var b strings.Builder
	b.WriteString("User-agent: *\nAllow: /$\nAllow: /c/\nAllow: /t/\nAllow: /r/\nAllow: /trust\n" +
		"Disallow: /api/\nDisallow: /banner/\nDisallow: /docs\n")
	if h.cfg.BaseURL != "" {
		b.WriteString("\nSitemap: " + h.absURL("/sitemap.xml") + "\n")
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write([]byte(b.String()))
}

func (h *Handler) absURL(path string) string {
	return strings.TrimRight(h.cfg.BaseURL, "/") + path
}
