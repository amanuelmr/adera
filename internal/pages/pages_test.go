package pages

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestHandler(t *testing.T, cfg Config) (*Handler, *http.ServeMux) {
	t.Helper()
	h, err := NewHandler(Dependencies{}, cfg)
	require.NoError(t, err)
	mux := http.NewServeMux()
	h.Routes(mux)
	return h, mux
}

func get(mux *http.ServeMux, target string, headers ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestTranslatePluralsAndFallback(t *testing.T) {
	dicts, err := loadDictionaries()
	require.NoError(t, err)

	assert.Equal(t, "1 review", translate(dicts, "en", "target.reviews", "count", 1))
	assert.Equal(t, "12 reviews", translate(dicts, "en", "target.reviews", "count", 12))
	assert.Equal(t, "12 ግምገማዎች", translate(dicts, "am", "target.reviews", "count", 12))
	assert.Equal(t, "no.such.key", translate(dicts, "am", "no.such.key"), "missing keys stay visible rather than blank")
}

func TestDictionariesHaveMatchingKeys(t *testing.T) {
	dicts, err := loadDictionaries()
	require.NoError(t, err)
	var walk func(prefix string, node any, out map[string]bool)
	walk = func(prefix string, node any, out map[string]bool) {
		switch v := node.(type) {
		case map[string]any:
			for k, child := range v {
				walk(prefix+"."+k, child, out)
			}
		case []any:
			out[prefix] = true // trust sections: compared by presence, content differs by language
		default:
			out[prefix] = true
		}
	}
	en, am := map[string]bool{}, map[string]bool{}
	walk("", map[string]any(dicts["en"]), en)
	walk("", map[string]any(dicts["am"]), am)
	assert.Equal(t, en, am)
}

func TestPickLanguage(t *testing.T) {
	cases := []struct {
		url, accept, want string
		explicit          bool
	}{
		{"/trust?lang=am", "en-US", "am", true},
		{"/trust", "am-ET,am;q=0.9,en;q=0.8", "am", false},
		{"/trust", "fr-FR, en;q=0.5", "en", false},
		{"/trust?lang=xx", "", "en", false},
	}
	for _, tc := range cases {
		r := httptest.NewRequest(http.MethodGet, tc.url, nil)
		r.Header.Set("Accept-Language", tc.accept)
		lang, explicit := pickLanguage(r)
		assert.Equal(t, tc.want, lang, tc.url+" "+tc.accept)
		assert.Equal(t, tc.explicit, explicit)
	}
}

func TestTrustCenterRendersInBothLanguages(t *testing.T) {
	_, mux := newTestHandler(t, Config{BaseURL: "https://adera.example", StoragePublicBaseURL: "https://cdn.example/public"})

	en := get(mux, "/trust")
	require.Equal(t, http.StatusOK, en.Code)
	assert.Contains(t, en.Body.String(), `<html lang="en">`)
	assert.Contains(t, en.Body.String(), "The rating you see is never adjusted")
	assert.Contains(t, en.Body.String(), `<link rel="canonical" href="https://adera.example/trust">`)

	am := get(mux, "/trust", "Accept-Language", "am")
	require.Equal(t, http.StatusOK, am.Code)
	assert.Contains(t, am.Body.String(), `<html lang="am">`)
	assert.Contains(t, am.Body.String(), "የሚያዩት ደረጃ በጭራሽ አይስተካከልም")
}

func TestPageHeadersReplaceTheAPIDefaults(t *testing.T) {
	_, mux := newTestHandler(t, Config{StoragePublicBaseURL: "https://cdn.example/public"})

	rec := get(mux, "/trust")
	csp := rec.Header().Get("Content-Security-Policy")
	assert.Contains(t, csp, "style-src 'self'")
	assert.Contains(t, csp, "img-src 'self' https://cdn.example", "review photos load from storage")
	assert.NotContains(t, csp, "script-src", "pages ship no JavaScript")
	assert.Equal(t, "public, max-age=300", rec.Header().Get("Cache-Control"))
	assert.Equal(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"))

	etag := rec.Header().Get("ETag")
	require.NotEmpty(t, etag)
	assert.Equal(t, http.StatusNotModified, get(mux, "/trust", "If-None-Match", etag).Code)
}

func TestNoInlineStylesOrScripts(t *testing.T) {
	// The CSP blocks both; this catches a template reintroducing them. The
	// one script tag allowed is JSON-LD, a data block browsers never run.
	for _, name := range []string{"layout", "partials", "home", "category", "target", "review", "trust", "notfound"} {
		raw, err := templateFS.ReadFile("templates/" + name + ".html")
		require.NoError(t, err)
		assert.NotContains(t, string(raw), "style=", name)
		withoutData := strings.ReplaceAll(string(raw), `<script type="application/ld+json">`, "")
		assert.NotContains(t, withoutData, "<script", name)
	}
}

func TestLanguageAlternatesAndCanonical(t *testing.T) {
	_, mux := newTestHandler(t, Config{BaseURL: "https://adera.example"})

	body := get(mux, "/trust").Body.String()
	assert.Contains(t, body, `<link rel="canonical" href="https://adera.example/trust">`)
	assert.Contains(t, body, `<link rel="alternate" hreflang="en" href="https://adera.example/trust?lang=en">`)
	assert.Contains(t, body, `<link rel="alternate" hreflang="am" href="https://adera.example/trust?lang=am">`)
	assert.Contains(t, body, `<link rel="alternate" hreflang="x-default" href="https://adera.example/trust">`)
	assert.Contains(t, body, `<meta property="og:image" content="https://adera.example/static/og.png">`)

	am := get(mux, "/trust?lang=am").Body.String()
	assert.Contains(t, am, `<link rel="canonical" href="https://adera.example/trust?lang=am">`,
		"a chosen language is its own version")
}

func TestShareImage(t *testing.T) {
	_, mux := newTestHandler(t, Config{})
	rec := get(mux, "/static/og.png")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "image/png", rec.Header().Get("Content-Type"))
	assert.Less(t, rec.Body.Len(), 100<<10)
}

func TestInstallBannerAndDismissal(t *testing.T) {
	_, mux := newTestHandler(t, Config{BaseURL: "https://adera.example", AndroidStoreURL: "https://play.example/adera"})

	assert.Contains(t, get(mux, "/trust").Body.String(), `href="https://play.example/adera"`)

	dismiss := get(mux, "/banner/dismiss?next=%2Ftrust%3Flang%3Dam")
	assert.Equal(t, http.StatusSeeOther, dismiss.Code)
	assert.Equal(t, "/trust?lang=am", dismiss.Header().Get("Location"))
	cookie := dismiss.Header().Get("Set-Cookie")
	assert.Contains(t, cookie, bannerCookie+"=off")
	assert.Contains(t, cookie, "Secure")

	after := get(mux, "/trust", "Cookie", bannerCookie+"=off")
	assert.NotContains(t, after.Body.String(), "play.example")
}

func TestDismissRedirectStaysOnSite(t *testing.T) {
	for _, next := range []string{"https://evil.example", "//evil.example", `/\evil.example`, "", "javascript:alert(1)"} {
		assert.Equal(t, "/trust", safeNext(next), next)
	}
	assert.Equal(t, "/t/tomoca", safeNext("/t/tomoca"))
}

func TestAssetLinks(t *testing.T) {
	_, off := newTestHandler(t, Config{AndroidPackage: "work.amanuel.adera"})
	assert.Equal(t, http.StatusNotFound, get(off, "/.well-known/assetlinks.json").Code, "no cert configured yet")

	_, mux := newTestHandler(t, Config{AndroidPackage: "work.amanuel.adera", AndroidCertSHA256s: []string{"AA:BB"}})
	rec := get(mux, "/.well-known/assetlinks.json")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	var body []struct {
		Relation []string `json:"relation"`
		Target   struct {
			Namespace   string   `json:"namespace"`
			PackageName string   `json:"package_name"`
			Certs       []string `json:"sha256_cert_fingerprints"`
		} `json:"target"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body, 1)
	assert.Equal(t, []string{"delegate_permission/common.handle_all_urls"}, body[0].Relation)
	assert.Equal(t, "android_app", body[0].Target.Namespace)
	assert.Equal(t, "work.amanuel.adera", body[0].Target.PackageName)
	assert.Equal(t, []string{"AA:BB"}, body[0].Target.Certs)
}

func TestStaticStylesheetIsSmallAndCacheable(t *testing.T) {
	_, mux := newTestHandler(t, Config{})
	rec := get(mux, "/static/site.css")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, strings.HasPrefix(rec.Header().Get("Content-Type"), "text/css"))
	assert.Equal(t, "public, max-age=86400", rec.Header().Get("Cache-Control"))
	assert.Less(t, rec.Body.Len(), 8<<10)
}

func TestRobotsAndFavicon(t *testing.T) {
	_, mux := newTestHandler(t, Config{})

	robots := get(mux, "/robots.txt")
	require.Equal(t, http.StatusOK, robots.Code)
	assert.True(t, strings.HasPrefix(robots.Header().Get("Content-Type"), "text/plain"))
	assert.Contains(t, robots.Body.String(), "Allow: /t/")
	assert.Contains(t, robots.Body.String(), "Allow: /c/")
	assert.Contains(t, robots.Body.String(), "Disallow: /api/")
	assert.NotContains(t, robots.Body.String(), "Sitemap:", "no absolute URL without a base URL")

	_, withBase := newTestHandler(t, Config{BaseURL: "https://adera.example/"})
	assert.Contains(t, get(withBase, "/robots.txt").Body.String(), "Sitemap: https://adera.example/sitemap.xml")

	ico := get(mux, "/favicon.ico")
	assert.Equal(t, http.StatusMovedPermanently, ico.Code)
	assert.Equal(t, "/static/favicon.svg", ico.Header().Get("Location"))
	svg := get(mux, "/static/favicon.svg")
	require.Equal(t, http.StatusOK, svg.Code)
	assert.Equal(t, "image/svg+xml", svg.Header().Get("Content-Type"))
}

func TestHeadingsDontSkipLevels(t *testing.T) {
	// The review partial sits directly under each page's <h1>, so its
	// headings must be <h2>: skipping to <h3> breaks heading navigation.
	raw, err := templateFS.ReadFile("templates/partials.html")
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "<h3")
}
