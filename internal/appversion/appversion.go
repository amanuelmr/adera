// Package appversion serves the mobile client version gate: the minimum
// build the API still supports, and the newest build available. Without it a
// broken client release can never be retired, so it has to exist before the
// first store submission rather than after.
package appversion

import (
	"fmt"
	"strconv"
	"strings"
)

// Supported client platforms.
const (
	PlatformAndroid = "android"
	PlatformIOS     = "ios"
)

// Gate is the version policy for one platform. An empty MinimumSupported
// disables the check, which is the default.
type Gate struct {
	MinimumSupported string
	Latest           string
	StoreURL         string
}

// Policy holds the per-platform gates.
type Policy struct {
	Android Gate
	IOS     Gate
}

// Gate returns the policy for a platform, and whether it is a known one.
func (p Policy) Gate(platform string) (Gate, bool) {
	switch platform {
	case PlatformAndroid:
		return p.Android, true
	case PlatformIOS:
		return p.IOS, true
	default:
		return Gate{}, false
	}
}

// Parse reads a dotted numeric version such as "1.4.2". Missing components
// count as zero, so "1.4" equals "1.4.0". Pre-release and build metadata
// after a '-' or '+' are ignored: they do not order reliably, and the gate
// only needs to compare releases.
func Parse(v string) ([3]int, error) {
	var out [3]int
	trimmed := strings.TrimSpace(v)
	if i := strings.IndexAny(trimmed, "-+"); i >= 0 {
		trimmed = trimmed[:i]
	}
	if trimmed == "" {
		return out, fmt.Errorf("empty version")
	}
	parts := strings.Split(trimmed, ".")
	if len(parts) > 3 {
		return out, fmt.Errorf("version %q has too many components", v)
	}
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return out, fmt.Errorf("version %q is not numeric", v)
		}
		out[i] = n
	}
	return out, nil
}

// Compare orders two versions, returning -1, 0, or 1. Both must parse.
func Compare(a, b string) (int, error) {
	av, err := Parse(a)
	if err != nil {
		return 0, err
	}
	bv, err := Parse(b)
	if err != nil {
		return 0, err
	}
	for i := range av {
		switch {
		case av[i] < bv[i]:
			return -1, nil
		case av[i] > bv[i]:
			return 1, nil
		}
	}
	return 0, nil
}
