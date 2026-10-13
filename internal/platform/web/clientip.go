package web

import (
	"fmt"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
)

// IPSource says where the caller's address comes from. Behind a proxy the
// TCP peer is the proxy, so every caller would share one rate-limit bucket;
// in front of none, a forwarded header is whatever the caller wrote. The
// right choice depends on the hosting, so it is configured (CLIENT_IP_SOURCE)
// and checked once against GET /api/v1/admin/request-info.
//
// The zero value uses the TCP peer address.
type IPSource struct {
	// Header, when set, is a single header written by a proxy that
	// overwrites any value the caller sent (e.g. CF-Connecting-IP).
	Header string
	// Hops, when positive, takes the Hops-th X-Forwarded-For entry from the
	// right: 1 is the address the nearest proxy saw. Entries further left
	// were written by the caller and aren't trusted.
	Hops int
}

// ParseIPSource reads "remote", "xff:N" (N ≥ 1) or "header:Name".
func ParseIPSource(s string) (IPSource, error) {
	switch kind, arg, _ := strings.Cut(strings.TrimSpace(s), ":"); strings.ToLower(kind) {
	case "", "remote":
		if arg != "" {
			return IPSource{}, fmt.Errorf("client IP source %q: remote takes no argument", s)
		}
		return IPSource{}, nil
	case "xff":
		n, err := strconv.Atoi(arg)
		if err != nil || n < 1 || n > 10 {
			return IPSource{}, fmt.Errorf("client IP source %q: xff needs a hop count from 1 to 10", s)
		}
		return IPSource{Hops: n}, nil
	case "header":
		name := http.CanonicalHeaderKey(strings.TrimSpace(arg))
		if name == "" || strings.ContainsAny(name, " \t,:") {
			return IPSource{}, fmt.Errorf("client IP source %q: header needs a header name", s)
		}
		if name == "X-Forwarded-For" {
			return IPSource{}, fmt.Errorf("client IP source %q: use xff:N for X-Forwarded-For", s)
		}
		return IPSource{Header: name}, nil
	default:
		return IPSource{}, fmt.Errorf("client IP source %q: want remote, xff:N or header:Name", s)
	}
}

func (s IPSource) String() string {
	switch {
	case s.Header != "":
		return "header:" + s.Header
	case s.Hops > 0:
		return "xff:" + strconv.Itoa(s.Hops)
	default:
		return "remote"
	}
}

// ClientIP returns the caller's address from the configured source. A
// missing or malformed forwarded value falls back to the TCP peer, so a
// misconfiguration degrades to shared buckets rather than to a value the
// caller chose.
func (s IPSource) ClientIP(r *http.Request) string {
	switch {
	case s.Header != "":
		if ip, ok := parseIP(r.Header.Get(s.Header)); ok {
			return ip
		}
	case s.Hops > 0:
		// Several X-Forwarded-For headers are one list, in order.
		var hops []string
		for _, v := range r.Header.Values("X-Forwarded-For") {
			hops = append(hops, strings.Split(v, ",")...)
		}
		if len(hops) >= s.Hops {
			if ip, ok := parseIP(hops[len(hops)-s.Hops]); ok {
				return ip
			}
		}
	}
	return remoteIP(r.RemoteAddr)
}

func parseIP(v string) (string, bool) {
	addr, err := netip.ParseAddr(strings.TrimSpace(v))
	if err != nil {
		return "", false
	}
	return addr.Unmap().WithZone("").String(), true
}

func remoteIP(remoteAddr string) string {
	if ap, err := netip.ParseAddrPort(remoteAddr); err == nil {
		return ap.Addr().Unmap().WithZone("").String()
	}
	if ip, ok := parseIP(remoteAddr); ok {
		return ip
	}
	return remoteAddr
}
