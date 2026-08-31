package appversion

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	valid := map[string][3]int{
		"1.4.2":         {1, 4, 2},
		"1.4":           {1, 4, 0},
		"2":             {2, 0, 0},
		"1.4.2-beta.1":  {1, 4, 2},
		"1.4.2+build99": {1, 4, 2},
		" 1.4.2 ":       {1, 4, 2},
	}
	for in, want := range valid {
		got, err := Parse(in)
		require.NoError(t, err, in)
		assert.Equal(t, want, got, in)
	}

	for _, in := range []string{"", "abc", "1.x.3", "1.2.3.4", "-1.0.0", "1..3"} {
		_, err := Parse(in)
		assert.Error(t, err, in)
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.0.1", "1.0.0", 1},
		// Numeric, not lexicographic: the classic version-sorting bug.
		{"1.9.0", "1.10.0", -1},
		{"1.10.0", "1.9.0", 1},
		{"2.0.0", "1.99.99", 1},
	}
	for _, tt := range tests {
		got, err := Compare(tt.a, tt.b)
		require.NoError(t, err)
		assert.Equal(t, tt.want, got, "%s vs %s", tt.a, tt.b)
	}

	_, err := Compare("nonsense", "1.0.0")
	assert.Error(t, err)
}

func testPolicy() Policy {
	return Policy{
		Android: Gate{
			MinimumSupported: "1.2.0",
			Latest:           "1.10.0",
			StoreURL:         "https://play.google.com/store/apps/details?id=et.adera",
		},
	}
}

func serve(t *testing.T, policy Policy, query string) (int, Response) {
	t.Helper()
	mux := http.NewServeMux()
	NewHandler(policy).Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/app/version"+query, nil))

	var body struct {
		Data Response `json:"data"`
	}
	if rec.Code == http.StatusOK {
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	}
	return rec.Code, body.Data
}

func TestVersionGate(t *testing.T) {
	t.Run("defaults to android", func(t *testing.T) {
		code, body := serve(t, testPolicy(), "")

		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, PlatformAndroid, body.Platform)
		assert.Equal(t, "1.2.0", body.MinimumSupported)
		// Without a client version nothing is evaluated, so absence can
		// never be misread as "you are up to date".
		assert.Nil(t, body.UpdateRequired)
		assert.Nil(t, body.UpdateAvailable)
	})

	t.Run("an outdated client must update", func(t *testing.T) {
		code, body := serve(t, testPolicy(), "?platform=android&version=1.1.9")

		assert.Equal(t, http.StatusOK, code)
		require.NotNil(t, body.UpdateRequired)
		assert.True(t, *body.UpdateRequired)
		assert.NotEmpty(t, body.StoreURL)
	})

	t.Run("a supported but older client is only offered an update", func(t *testing.T) {
		code, body := serve(t, testPolicy(), "?version=1.9.0")

		assert.Equal(t, http.StatusOK, code)
		require.NotNil(t, body.UpdateRequired)
		assert.False(t, *body.UpdateRequired)
		require.NotNil(t, body.UpdateAvailable)
		assert.True(t, *body.UpdateAvailable, "1.9.0 is older than 1.10.0")
	})

	t.Run("the newest client needs nothing", func(t *testing.T) {
		_, body := serve(t, testPolicy(), "?version=1.10.0")

		require.NotNil(t, body.UpdateAvailable)
		assert.False(t, *body.UpdateAvailable)
		assert.False(t, *body.UpdateRequired)
	})

	t.Run("an unconfigured platform evaluates nothing", func(t *testing.T) {
		code, body := serve(t, testPolicy(), "?platform=ios&version=1.0.0")

		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, PlatformIOS, body.Platform)
		assert.Nil(t, body.UpdateRequired)
	})

	t.Run("rejects an unknown platform", func(t *testing.T) {
		code, _ := serve(t, testPolicy(), "?platform=symbian")

		assert.Equal(t, http.StatusUnprocessableEntity, code)
	})

	t.Run("rejects an unparseable client version", func(t *testing.T) {
		code, _ := serve(t, testPolicy(), "?version=banana")

		assert.Equal(t, http.StatusUnprocessableEntity, code)
	})
}
