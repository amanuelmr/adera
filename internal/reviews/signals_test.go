package reviews

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNetworkPrefix(t *testing.T) {
	tests := []struct {
		ip     string
		want   string
		wantOK bool
	}{
		{"196.188.12.34", "196.188.12.0/24", true},
		{"196.188.12.255", "196.188.12.0/24", true},
		{"::ffff:196.188.12.34", "196.188.12.0/24", true}, // IPv4-mapped counts as IPv4
		{"2001:db8:abcd:12:1::5", "2001:db8:abcd::/48", true},
		{"fe80::1%eth0", "fe80::/48", true},
		{"", "", false},
		{"not-an-ip", "", false},
		{"196.188.12.34:443", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			got, ok := NetworkPrefix(tt.ip)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSignalHasherCapture(t *testing.T) {
	h := NewSignalHasher("unit-test-signal-key-at-least-32-bytes", false)
	request := func(remote, install string) Signals {
		r := httptest.NewRequest("POST", "/api/v1/reviews", nil)
		r.RemoteAddr = remote
		if install != "" {
			r.Header.Set(InstallIDHeader, install)
		}
		return h.Capture(r)
	}
	const phone = "6f1c2b9e-8d43-4a57-9b0e-2f6c1d3a4b5c"

	a := request("196.188.12.34:5000", phone)
	assert.Len(t, a.NetworkHash, 32)
	assert.Len(t, a.InstallHash, 32)

	sameNetwork := request("196.188.12.99:6000", "")
	assert.Equal(t, a.NetworkHash, sameNetwork.NetworkHash, "same /24")
	assert.Nil(t, sameNetwork.InstallHash)

	otherNetwork := request("196.188.13.34:5000", phone)
	assert.NotEqual(t, a.NetworkHash, otherNetwork.NetworkHash)
	assert.Equal(t, a.InstallHash, otherNetwork.InstallHash, "same install")

	assert.Equal(t, a.InstallHash, request("10.0.0.1:1", "6F1C2B9E-8D43-4A57-9B0E-2F6C1D3A4B5C").InstallHash,
		"install IDs compare case-insensitively")
	assert.Nil(t, request("10.0.0.1:1", "00000000-0000-0000-0000-000000000000").InstallHash)
	assert.Nil(t, request("10.0.0.1:1", "my-phone").InstallHash)

	other := NewSignalHasher("another-signal-key-at-least-32-bytes!", false)
	r := httptest.NewRequest("POST", "/", nil)
	r.RemoteAddr = "196.188.12.34:5000"
	assert.NotEqual(t, a.NetworkHash, other.Capture(r).NetworkHash, "keyed: a different key gives different hashes")

	var off *SignalHasher
	assert.Equal(t, Signals{}, off.Capture(r))
}
