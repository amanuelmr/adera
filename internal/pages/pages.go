// Package pages is Adera's thin server-rendered web layer
// (docs/mobile-plan.md §1): target profiles, review permalinks, and the
// Trust Center, so a link shared on Telegram lands somewhere readable
// without the app; plus a home page, category pages and a sitemap, so the
// places can be found from a search engine too (discovery.go). Plain HTML with one small stylesheet and no JavaScript,
// sized for the slow connections docs/frontend-handoff.md §5 budgets for.
package pages

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/adera-platform/backend/internal/categories"
	"github.com/adera-platform/backend/internal/platform/web"
	"github.com/adera-platform/backend/internal/ratings"
	"github.com/adera-platform/backend/internal/reviews"
	"github.com/adera-platform/backend/internal/targets"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

const (
	reviewsPerPage = 20
	bannerCookie   = "adera_banner"
)

// Dependencies are the read methods the pages reuse from the API modules,
// so both surfaces apply the same publication and aggregation rules.
type Dependencies struct {
	Targets    *targets.Repo
	Ratings    *ratings.Repo
	Reviews    *reviews.Repo
	Categories *categories.Repo
}

type Config struct {
	// BaseURL is the public origin used for canonical and Open Graph URLs.
	BaseURL string
	// StoragePublicBaseURL is where review photos are served from; its
	// origin is allowed by the page CSP.
	StoragePublicBaseURL string
	// AndroidStoreURL enables the install banner when set.
	AndroidStoreURL    string
	AndroidPackage     string
	AndroidCertSHA256s []string
}

type Handler struct {
	deps      Dependencies
	cfg       Config
	dicts     map[string]dictionary
	templates map[string]*template.Template
	csp       string
}

func NewHandler(deps Dependencies, cfg Config) (*Handler, error) {
	dicts, err := loadDictionaries()
	if err != nil {
		return nil, err
	}
	h := &Handler{deps: deps, cfg: cfg, dicts: dicts, templates: map[string]*template.Template{}}
	for _, name := range []string{"home", "category", "target", "review", "trust", "notfound"} {
		t, err := template.New("layout.html").Funcs(template.FuncMap{"stars": stars}).
			ParseFS(templateFS, "templates/layout.html", "templates/partials.html", "templates/"+name+".html")
		if err != nil {
			return nil, fmt.Errorf("parsing %s template: %w", name, err)
		}
		h.templates[name] = t
	}
	img := "'self'"
	if u, err := url.Parse(cfg.StoragePublicBaseURL); err == nil && u.Scheme != "" && u.Host != "" {
		img += " " + u.Scheme + "://" + u.Host
	}
	h.csp = "default-src 'none'; style-src 'self'; img-src " + img +
		"; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"
	return h, nil
}

func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /{$}", h.home)
	mux.HandleFunc("GET /c/{code}", h.category)
	mux.HandleFunc("GET /t/{slug}", h.target)
	mux.HandleFunc("GET /r/{id}", h.review)
	mux.HandleFunc("GET /trust", h.trust)
	mux.HandleFunc("GET /banner/dismiss", h.dismissBanner)
	mux.HandleFunc("GET /.well-known/assetlinks.json", h.assetLinks)
	mux.HandleFunc("GET /robots.txt", h.robots)
	mux.HandleFunc("GET /sitemap.xml", h.sitemap)
	mux.HandleFunc("GET /favicon.ico", h.favicon)
	static, _ := fs.Sub(staticFS, "static")
	files := http.StripPrefix("/static/", http.FileServerFS(static))
	mux.Handle("GET /static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=86400")
		files.ServeHTTP(w, r)
	}))
}

// base is the data every template gets: language, strings, and the shared
// head/banner/footer fields.
type base struct {
	Lang        string
	Title       string
	Description string
	Canonical   string
	// Alternates are the hreflang URLs: each language explicitly, and the
	// bare path (language picked from the browser) as x-default.
	AltEN, AltAM, AltDefault string
	OGImage                  string
	LangSwitchURL            string
	// AppLink is the app's own deep link. html/template rejects non-http
	// schemes, so it's built here from an escaped slug and marked trusted.
	AppLink      template.URL
	StoreURL     string
	DismissURL   string
	OGType       string
	dicts        map[string]dictionary
	explicitLang bool
}

// T renders a localized string; args are name/value pairs (see translate).
func (b base) T(key string, args ...any) string { return translate(b.dicts, b.Lang, key, args...) }

// Link adds ?lang= to internal links when the reader chose a language
// explicitly, so it sticks while they browse.
func (b base) Link(path string, query url.Values) string {
	if query == nil {
		query = url.Values{}
	}
	if b.explicitLang {
		query.Set("lang", b.Lang)
	}
	if len(query) == 0 {
		return path
	}
	return path + "?" + query.Encode()
}

func (h *Handler) newBase(r *http.Request, path string) base {
	lang, explicit := pickLanguage(r)
	other := "am"
	if lang == "am" {
		other = "en"
	}
	switchQuery := r.URL.Query()
	switchQuery.Set("lang", other)
	abs := h.absURL(path)
	b := base{
		Lang: lang,
		// A page chosen by ?lang= is its own language version; the bare URL
		// negotiates and is the x-default.
		Canonical:     abs,
		AltEN:         abs + "?lang=en",
		AltAM:         abs + "?lang=am",
		AltDefault:    abs,
		OGImage:       h.absURL("/static/og.png"),
		LangSwitchURL: path + "?" + switchQuery.Encode(),
		OGType:        "website",
		dicts:         h.dicts,
		explicitLang:  explicit,
	}
	if explicit {
		b.Canonical = abs + "?lang=" + lang
	}
	if h.cfg.AndroidStoreURL != "" {
		if c, err := r.Cookie(bannerCookie); err != nil || c.Value != "off" {
			b.StoreURL = h.cfg.AndroidStoreURL
			b.DismissURL = "/banner/dismiss?" + url.Values{"next": {r.URL.RequestURI()}}.Encode()
		}
	}
	return b
}

// ---- target profile ----

type histogramRow struct {
	Stars int
	Count int
	// Width is the bar length rounded to 5% steps: the page CSP forbids
	// inline styles, so widths are predefined classes in site.css.
	Width  int
	URL    string
	Active bool
}

type responseView struct {
	Body string
	Date string
}

type reviewView struct {
	// B gives the shared "review" partial the page's language and strings.
	B                 base
	Reviewer          string
	Title             string
	Body              string
	Rating            int
	Date              string
	Edited            bool
	Verified          bool
	Helpful           int
	Incentive         string
	Connection        string
	DisclosureDetails string
	Media             []reviews.MediaRef
	Response          *responseView
	Permalink         string
}

type targetPage struct {
	base
	Target          targets.Target
	Category        string
	ShowAggregate   bool
	Average         string
	ReviewCount     int
	VerifiedAverage string
	Histogram       []histogramRow
	Rating          int
	Reviews         []reviewView
	NextURL         string
	AllRatingsURL   string
	JSONLD          *structuredData
}

func (h *Handler) target(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	t, err := h.deps.Targets.GetByIDOrSlug(ctx, r.PathValue("slug"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if t.ModerationStatus == targets.ModerationMerged {
		h.redirectMerged(w, r, t.ID)
		return
	}
	if t.ModerationStatus != targets.ModerationPublished {
		h.notFound(w, r)
		return
	}

	path := "/t/" + t.Slug
	p := targetPage{base: h.newBase(r, path), Target: t}
	p.Title = t.Name
	p.AppLink = appLink(t.Slug)

	if cat, err := h.deps.Categories.GetByIDOrCode(ctx, t.CategoryID.String()); err == nil {
		p.Category = cat.Name
		if name := cat.NameTranslations[p.Lang]; name != "" {
			p.Category = name
		}
	}

	stats, err := h.deps.Ratings.Stats(ctx, t.ID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	p.ReviewCount = stats.ReviewCount
	// Same rule as the app: below 3 reviews there's no aggregate to show.
	p.ShowAggregate = stats.Confidence != "none" && stats.Average != nil
	if p.ShowAggregate {
		p.Average = strconv.FormatFloat(*stats.Average, 'f', 1, 64)
		if stats.VerifiedCount > 0 && stats.VerifiedAverage != nil {
			p.VerifiedAverage = strconv.FormatFloat(*stats.VerifiedAverage, 'f', 1, 64)
		}
		p.Title = fmt.Sprintf("%s — %s★ (%d)", t.Name, p.Average, stats.ReviewCount)
	}
	p.Description = strings.Join(nonEmpty(p.Category, t.AddressText, p.T("tagline")), " · ")
	p.JSONLD = h.targetStructuredData(t, stats, p.ShowAggregate)

	if n, err := strconv.Atoi(r.URL.Query().Get("rating")); err == nil && n >= 1 && n <= 5 {
		p.Rating = n
	}
	p.AllRatingsURL = p.Link(path, nil)
	for stars := 5; stars >= 1; stars-- {
		count := stats.Distribution[stars-1]
		row := histogramRow{Stars: stars, Count: count, Active: p.Rating == stars,
			URL: p.Link(path, url.Values{"rating": {strconv.Itoa(stars)}})}
		if stats.ReviewCount > 0 {
			row.Width = (count*100/stats.ReviewCount + 2) / 5 * 5
		}
		p.Histogram = append(p.Histogram, row)
	}

	cursor, err := web.ParseCursor(r)
	if err != nil {
		cursor = nil // a mangled ?cursor= just means the first page
	}
	items, next, err := h.deps.Reviews.ListForTarget(ctx, t.ID, uuid.Nil,
		reviews.ListFilter{Rating: p.Rating}, reviews.SortNewest, cursor, reviewsPerPage)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	for _, rv := range items {
		p.Reviews = append(p.Reviews, h.reviewView(p.base, rv))
	}
	if next != nil {
		q := url.Values{"cursor": {next.Encode()}}
		if p.Rating > 0 {
			q.Set("rating", strconv.Itoa(p.Rating))
		}
		p.NextURL = p.Link(path, q)
	}
	h.render(w, r, "target", http.StatusOK, p)
}

func (h *Handler) redirectMerged(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	into, err := h.deps.Targets.MergedInto(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if into == nil {
		h.notFound(w, r)
		return
	}
	survivor, err := h.deps.Targets.GetByIDOrSlug(r.Context(), into.String())
	if err != nil || survivor.ModerationStatus != targets.ModerationPublished {
		h.notFound(w, r)
		return
	}
	http.Redirect(w, r, "/t/"+survivor.Slug, http.StatusMovedPermanently)
}

// ---- review permalink ----

type reviewPage struct {
	base
	Target   targets.Target
	Review   reviewView
	AllURL   string
	Category string
}

func (h *Handler) review(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		h.notFound(w, r)
		return
	}
	rv, err := h.deps.Reviews.GetListed(ctx, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	t, err := h.deps.Targets.GetByIDOrSlug(ctx, rv.TargetID.String())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if t.ModerationStatus != targets.ModerationPublished {
		h.notFound(w, r)
		return
	}
	p := reviewPage{base: h.newBase(r, "/r/"+id.String()), Target: t}
	p.Review = h.reviewView(p.base, rv)
	p.AllURL = p.Link("/t/"+t.Slug, nil)
	p.AppLink = appLink(t.Slug)
	p.OGType = "article"
	p.Title = stars(rv.OverallRating) + " " + p.T("review.reviewOf", "name", t.Name)
	p.Description = excerpt(firstNonEmpty(rv.Title, rv.Body), 200)
	h.render(w, r, "review", http.StatusOK, p)
}

func (h *Handler) reviewView(b base, rv reviews.ListedReview) reviewView {
	v := reviewView{
		B:                 b,
		Reviewer:          rv.ReviewerName,
		Title:             rv.Title,
		Body:              rv.Body,
		Rating:            rv.OverallRating,
		Date:              formatDate(b.Lang, rv.CreatedAt),
		Edited:            rv.EditCount > 0,
		Helpful:           rv.HelpfulCount,
		DisclosureDetails: rv.DisclosureDetails,
		Media:             rv.Media,
		Permalink:         b.Link("/r/"+rv.ID.String(), nil),
	}
	switch rv.VerificationLevel {
	case "receipt_submitted", "location_verified", "partner_verified":
		v.Verified = true
	}
	// docs/moderation-policy.md §6: any disclosure other than none is labeled.
	if rv.IncentiveType != "" && rv.IncentiveType != "none" {
		v.Incentive = b.T("disclosure.incentive", "value", b.T("disclosure.incentiveValue."+rv.IncentiveType))
	}
	if rv.MaterialConnection != "" && rv.MaterialConnection != "none" {
		v.Connection = b.T("disclosure.connection", "value", b.T("disclosure.connectionValue."+rv.MaterialConnection))
	}
	if rv.Response != nil {
		v.Response = &responseView{Body: rv.Response.Body, Date: formatDate(b.Lang, rv.Response.CreatedAt)}
	}
	return v
}

// ---- trust center ----

type trustSection struct {
	Title      string   `json:"title"`
	Paragraphs []string `json:"paragraphs"`
}

type trustPage struct {
	base
	Sections []trustSection
}

func (h *Handler) trust(w http.ResponseWriter, r *http.Request) {
	p := trustPage{base: h.newBase(r, "/trust")}
	p.Title = p.T("trust.title")
	p.Description = p.T("trust.intro")
	raw, _ := json.Marshal(h.dicts[p.Lang].lookup("trust.sections"))
	if err := json.Unmarshal(raw, &p.Sections); err != nil {
		h.fail(w, r, err)
		return
	}
	h.render(w, r, "trust", http.StatusOK, p)
}

// ---- banner, app links ----

func (h *Handler) dismissBanner(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name: bannerCookie, Value: "off", Path: "/", MaxAge: 30 * 24 * 3600,
		HttpOnly: true, SameSite: http.SameSiteLaxMode,
		Secure: strings.HasPrefix(h.cfg.BaseURL, "https://"),
	})
	http.Redirect(w, r, safeNext(r.URL.Query().Get("next")), http.StatusSeeOther)
}

// safeNext keeps the post-dismiss redirect on this site: only a local
// absolute path, never "//host" or a full URL (an open redirect otherwise).
func safeNext(next string) string {
	if !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") || strings.HasPrefix(next, "/\\") {
		return "/trust"
	}
	return next
}

func (h *Handler) assetLinks(w http.ResponseWriter, r *http.Request) {
	if h.cfg.AndroidPackage == "" || len(h.cfg.AndroidCertSHA256s) == 0 {
		http.NotFound(w, r)
		return
	}
	body, _ := json.Marshal([]map[string]any{{
		"relation": []string{"delegate_permission/common.handle_all_urls"},
		"target": map[string]any{
			"namespace":                "android_app",
			"package_name":             h.cfg.AndroidPackage,
			"sha256_cert_fingerprints": h.cfg.AndroidCertSHA256s,
		},
	}})
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write(body)
}

// favicon: browsers and crawlers request /favicon.ico regardless of the
// <link rel="icon"> tag; point them at the SVG instead of a 404.
func (h *Handler) favicon(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/static/favicon.svg", http.StatusMovedPermanently)
}

// ---- rendering ----

func (h *Handler) notFound(w http.ResponseWriter, r *http.Request) {
	p := struct{ base }{h.newBase(r, r.URL.Path)}
	p.Title = p.T("notFound.title")
	h.render(w, r, "notfound", http.StatusNotFound, p)
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var we *web.Error
	if errors.As(err, &we) && we.Status == http.StatusNotFound {
		h.notFound(w, r)
		return
	}
	slog.ErrorContext(r.Context(), "rendering page", "path", r.URL.Path, "error", err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

// render overrides the API's JSON-oriented headers (no-store, CSP
// default-src 'none') with page-appropriate ones, and serves an ETag so a
// revisit over a slow connection costs a 304, not the page again.
func (h *Handler) render(w http.ResponseWriter, r *http.Request, name string, status int, data any) {
	var buf bytes.Buffer
	if err := h.templates[name].Execute(&buf, data); err != nil {
		h.fail(w, r, err)
		return
	}
	sum := sha256.Sum256(buf.Bytes())
	etag := `"` + hex.EncodeToString(sum[:8]) + `"`

	hd := w.Header()
	hd.Set("Content-Type", "text/html; charset=utf-8")
	hd.Set("Content-Security-Policy", h.csp)
	hd.Set("Referrer-Policy", "strict-origin-when-cross-origin")
	hd.Set("Vary", "Accept-Language, Cookie")
	if status == http.StatusOK {
		hd.Set("Cache-Control", "public, max-age=300")
		hd.Set("ETag", etag)
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	} else {
		hd.Set("Cache-Control", "no-store")
	}
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

// ---- helpers ----

func appLink(slug string) template.URL {
	return template.URL("adera://target/" + url.PathEscape(slug)) //nolint:gosec // fixed scheme, escaped server-generated slug
}

func stars(n int) string {
	if n < 0 {
		n = 0
	}
	if n > 5 {
		n = 5
	}
	return strings.Repeat("★", n) + strings.Repeat("☆", 5-n)
}

// Gregorian, Western digits for everyone (docs/frontend-handoff.md §7);
// Amharic avoids month names that would need translating.
func formatDate(lang string, t time.Time) string {
	if lang == "am" {
		return t.Format("2006-01-02")
	}
	return t.Format("Jan 2, 2006")
}

func excerpt(s string, max int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= max {
		return string(r)
	}
	return strings.TrimSpace(string(r[:max])) + "…"
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func nonEmpty(values ...string) []string {
	out := values[:0:0]
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			out = append(out, v)
		}
	}
	return out
}
