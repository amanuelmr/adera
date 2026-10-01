package pages

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

//go:embed i18n/*.json
var localeFS embed.FS

// Languages the web pages are translated into; the first is the fallback.
var languages = []string{"en", "am"}

type dictionary map[string]any

func loadDictionaries() (map[string]dictionary, error) {
	out := make(map[string]dictionary, len(languages))
	for _, lang := range languages {
		raw, err := localeFS.ReadFile("i18n/" + lang + ".json")
		if err != nil {
			return nil, fmt.Errorf("reading %s strings: %w", lang, err)
		}
		var d dictionary
		if err := json.Unmarshal(raw, &d); err != nil {
			return nil, fmt.Errorf("parsing %s strings: %w", lang, err)
		}
		out[lang] = d
	}
	return out, nil
}

// lookup walks a dotted key ("target.noReviews") through nested objects.
func (d dictionary) lookup(key string) any {
	var node any = map[string]any(d)
	for _, part := range strings.Split(key, ".") {
		m, ok := node.(map[string]any)
		if !ok {
			return nil
		}
		node = m[part]
	}
	return node
}

// translate returns the string for key with {{name}} placeholders filled
// from args (name, value pairs). A "count" arg selects key_one/key_other.
// Both languages here pluralize the same way (one vs. other), so the
// rule is shared.
func translate(dicts map[string]dictionary, lang, key string, args ...any) string {
	params := map[string]string{}
	for i := 0; i+1 < len(args); i += 2 {
		params[fmt.Sprint(args[i])] = fmt.Sprint(args[i+1])
	}
	resolved := key
	if count, ok := params["count"]; ok {
		if count == "1" {
			resolved = key + "_one"
		} else {
			resolved = key + "_other"
		}
	}
	for _, l := range []string{lang, languages[0]} {
		for _, k := range []string{resolved, key} {
			if s, ok := dicts[l].lookup(k).(string); ok {
				for name, value := range params {
					s = strings.ReplaceAll(s, "{{"+name+"}}", value)
				}
				return s
			}
		}
	}
	return key
}

// pickLanguage: an explicit ?lang= wins (and is what language-switch links
// set), then the browser's Accept-Language, then English.
func pickLanguage(r *http.Request) (lang string, explicit bool) {
	if q := r.URL.Query().Get("lang"); isLanguage(q) {
		return q, true
	}
	for _, part := range strings.Split(r.Header.Get("Accept-Language"), ",") {
		tag := strings.ToLower(strings.TrimSpace(strings.SplitN(part, ";", 2)[0]))
		base := strings.SplitN(tag, "-", 2)[0]
		if isLanguage(base) {
			return base, false
		}
	}
	return languages[0], false
}

func isLanguage(s string) bool {
	for _, l := range languages {
		if s == l {
			return true
		}
	}
	return false
}
