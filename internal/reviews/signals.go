package reviews

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/adera-platform/backend/internal/platform/web"
)

// Fraud signals (docs/moderation-policy.md, "Fraud signals"): when a review is created we
// keep a keyed hash of the network it came from and of the app install that
// sent it, so moderators can see several "different" reviewers of one place
// sharing a phone or a connection. Signals only: nothing is blocked or
// hidden automatically, and raw IP addresses and install IDs are never stored.

// SignalRetention is how long signals are kept before PurgeExpiredSignals
// removes them.
const SignalRetention = 90 * 24 * time.Hour

// InstallIDHeader carries the app's random per-install identifier.
const InstallIDHeader = "X-Install-ID"

// Signals are the hashed request fingerprints stored with a new review. A nil
// field means the signal was unavailable (no install ID, unparsable address).
type Signals struct {
	NetworkHash []byte
	InstallHash []byte
}

func (s Signals) empty() bool { return s.NetworkHash == nil && s.InstallHash == nil }

// SignalHasher turns request details into Signals with HMAC-SHA256, so the
// stored values can be compared with each other but not reversed, or matched
// against a known address, without the key.
type SignalHasher struct {
	key        []byte
	trustProxy bool
}

func NewSignalHasher(key string, trustProxy bool) *SignalHasher {
	return &SignalHasher{key: []byte(key), trustProxy: trustProxy}
}

// Capture derives the signals for a request. A nil hasher captures nothing.
func (h *SignalHasher) Capture(r *http.Request) Signals {
	if h == nil {
		return Signals{}
	}
	var s Signals
	if prefix, ok := NetworkPrefix(web.ClientIP(r, h.trustProxy)); ok {
		s.NetworkHash = h.sum("network", prefix)
	}
	if id, err := uuid.Parse(r.Header.Get(InstallIDHeader)); err == nil && id != uuid.Nil {
		s.InstallHash = h.sum("install", id.String())
	}
	return s
}

// sum keeps the two kinds of signal in separate domains, so an install ID can
// never collide with a network prefix.
func (h *SignalHasher) sum(kind, value string) []byte {
	mac := hmac.New(sha256.New, h.key)
	mac.Write([]byte(kind + ":" + value))
	return mac.Sum(nil)
}

// NetworkPrefix reduces an address to the network it belongs to: /24 for IPv4
// and /48 for IPv6, the granularity at which one household, office or mobile
// carrier pool shows up. Coarser than the address, so it identifies a
// connection rather than a person.
func NetworkPrefix(ip string) (string, bool) {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return "", false
	}
	addr = addr.Unmap().WithZone("")
	bits := 48
	if addr.Is4() {
		bits = 24
	}
	prefix, err := addr.Prefix(bits)
	if err != nil {
		return "", false
	}
	return prefix.String(), true
}

func insertSignals(ctx context.Context, tx pgx.Tx, reviewID, targetID uuid.UUID, s Signals) error {
	if s.empty() {
		return nil
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO review_signals (review_id, target_id, network_hash, install_hash)
		VALUES ($1, $2, $3, $4)`, reviewID, targetID, s.NetworkHash, s.InstallHash); err != nil {
		return fmt.Errorf("storing review signals: %w", err)
	}
	return nil
}

// SignalGroup is a set of reviews of one place, by more than one account,
// that came from the same device or the same network.
type SignalGroup struct {
	// Kind is "device" (same app install) or "network" (same /24 or /48).
	Kind string `json:"kind"`
	// Key is an opaque label for the shared signal, stable across requests,
	// so the same group can be recognised on other places.
	Key     string            `json:"key"`
	Reviews []SignalGroupItem `json:"reviews"`
}

type SignalGroupItem struct {
	ReviewID         uuid.UUID `json:"review_id"`
	UserID           uuid.UUID `json:"user_id"`
	ReviewerName     string    `json:"reviewer_name"`
	AccountCreatedAt time.Time `json:"account_created_at"`
	OverallRating    int       `json:"overall_rating"`
	Status           string    `json:"moderation_status"`
	CreatedAt        time.Time `json:"created_at"`
}

// SignalGroupsForTarget lists the reviews of a place that share a device or a
// network with a review by a different account. Device groups come first:
// one phone behind several accounts is the stronger signal, since many
// honest people share a network (an office, a café's Wi-Fi, a carrier pool).
func (r *Repo) SignalGroupsForTarget(ctx context.Context, targetID uuid.UUID) ([]SignalGroup, error) {
	rows, err := r.pool.Query(ctx, `
		WITH signals AS (
			SELECT 'device' AS kind, s.install_hash AS hash, s.review_id
			FROM review_signals s WHERE s.target_id = $1 AND s.install_hash IS NOT NULL
			UNION ALL
			SELECT 'network', s.network_hash, s.review_id
			FROM review_signals s WHERE s.target_id = $1 AND s.network_hash IS NOT NULL
		),
		shared AS (
			SELECT sg.kind, sg.hash
			FROM signals sg JOIN reviews rv ON rv.id = sg.review_id
			GROUP BY sg.kind, sg.hash
			HAVING count(DISTINCT rv.user_id) > 1
		)
		SELECT sh.kind, sh.hash, rv.id, rv.user_id, u.display_name, u.created_at,
			rv.overall_rating, rv.moderation_status, rv.created_at
		FROM shared sh
		JOIN signals sg ON sg.kind = sh.kind AND sg.hash = sh.hash
		JOIN reviews rv ON rv.id = sg.review_id
		JOIN users u ON u.id = rv.user_id
		ORDER BY sh.kind = 'network', sh.hash, rv.created_at`, targetID)
	if err != nil {
		return nil, fmt.Errorf("listing review signal groups: %w", err)
	}
	defer rows.Close()

	groups := []SignalGroup{}
	for rows.Next() {
		var kind string
		var hash []byte
		var it SignalGroupItem
		if err := rows.Scan(&kind, &hash, &it.ReviewID, &it.UserID, &it.ReviewerName, &it.AccountCreatedAt,
			&it.OverallRating, &it.Status, &it.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning review signal group: %w", err)
		}
		key := hex.EncodeToString(hash[:6])
		if n := len(groups); n == 0 || groups[n-1].Kind != kind || groups[n-1].Key != key {
			groups = append(groups, SignalGroup{Kind: kind, Key: key})
		}
		g := &groups[len(groups)-1]
		g.Reviews = append(g.Reviews, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading review signal groups: %w", err)
	}
	return groups, nil
}

// PurgeExpiredSignals deletes signals older than SignalRetention and reports
// how many were removed.
func (r *Repo) PurgeExpiredSignals(ctx context.Context) (int64, error) {
	tag, err := r.pool.Exec(ctx, `DELETE FROM review_signals WHERE created_at < $1`, time.Now().Add(-SignalRetention))
	if err != nil {
		return 0, fmt.Errorf("purging review signals: %w", err)
	}
	return tag.RowsAffected(), nil
}
