package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseIPSource(t *testing.T) {
	good := map[string]IPSource{
		"":                        {},
		"remote":                  {},
		"xff:1":                   {Hops: 1},
		"XFF:2":                   {Hops: 2},
		"header:cf-connecting-ip": {Header: "Cf-Connecting-Ip"},
		" header:True-Client-IP ": {Header: "True-Client-Ip"},
	}
	for in, want := range good {
		got, err := ParseIPSource(in)
		require.NoError(t, err, in)
		assert.Equal(t, want, got, in)
	}
	for _, in := range []string{"xff", "xff:0", "xff:-1", "xff:11", "xff:a", "header:", "header:X-Forwarded-For", "header:a b", "remote:1", "proxy"} {
		_, err := ParseIPSource(in)
		assert.Error(t, err, in)
	}
}

func TestIPSourceString(t *testing.T) {
	for _, s := range []string{"remote", "xff:2", "header:Cf-Connecting-Ip"} {
		src, err := ParseIPSource(s)
		require.NoError(t, err)
		assert.Equal(t, s, src.String())
	}
}

func TestClientIP(t *testing.T) {
	req := func(remote string, headers ...string) *http.Request {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = remote
		for i := 0; i+1 < len(headers); i += 2 {
			r.Header.Add(headers[i], headers[i+1])
		}
		return r
	}
	tests := []struct {
		name string
		src  IPSource
		r    *http.Request
		want string
	}{
		{"remote ignores forwarded headers", IPSource{},
			req("10.0.0.1:5000", "X-Forwarded-For", "1.2.3.4", "CF-Connecting-IP", "1.2.3.4"), "10.0.0.1"},
		{"remote IPv6 peer", IPSource{}, req("[2001:db8::1]:443"), "2001:db8::1"},
		{"remote IPv4-mapped peer", IPSource{}, req("[::ffff:10.0.0.1]:443"), "10.0.0.1"},
		{"xff:1 takes the rightmost hop, not the caller's claim", IPSource{Hops: 1},
			req("10.0.0.1:5000", "X-Forwarded-For", "6.6.6.6, 196.188.1.2"), "196.188.1.2"},
		{"xff:2 skips one trusted proxy", IPSource{Hops: 2},
			req("10.0.0.1:5000", "X-Forwarded-For", "6.6.6.6, 196.188.1.2, 172.70.1.1"), "196.188.1.2"},
		{"xff spans repeated headers", IPSource{Hops: 2},
			req("10.0.0.1:5000", "X-Forwarded-For", "196.188.1.2", "X-Forwarded-For", "172.70.1.1"), "196.188.1.2"},
		{"xff with too few hops falls back to the peer", IPSource{Hops: 2},
			req("10.0.0.1:5000", "X-Forwarded-For", "196.188.1.2"), "10.0.0.1"},
		{"xff garbage falls back to the peer", IPSource{Hops: 1},
			req("10.0.0.1:5000", "X-Forwarded-For", "not-an-ip"), "10.0.0.1"},
		{"xff missing falls back to the peer", IPSource{Hops: 1}, req("10.0.0.1:5000"), "10.0.0.1"},
		{"header is used when valid", IPSource{Header: "Cf-Connecting-Ip"},
			req("10.0.0.1:5000", "CF-Connecting-IP", " 196.188.1.2 ", "X-Forwarded-For", "6.6.6.6"), "196.188.1.2"},
		{"header IPv6", IPSource{Header: "Cf-Connecting-Ip"},
			req("10.0.0.1:5000", "CF-Connecting-IP", "2001:db8::7"), "2001:db8::7"},
		{"header garbage falls back to the peer", IPSource{Header: "Cf-Connecting-Ip"},
			req("10.0.0.1:5000", "CF-Connecting-IP", "1.2.3.4, 5.6.7.8"), "10.0.0.1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.src.ClientIP(tt.r))
		})
	}
}
