package notifications

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adera-platform/backend/internal/platform/testdb"
)

// deviceToken builds a token of the minimum accepted length.
func deviceToken(suffix string) string {
	return strings.Repeat("t", 32) + suffix
}

func seedUser(t *testing.T, pool *pgxpool.Pool, label string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO users (id, display_name, email) VALUES ($1, $2, $3)`,
		id, label, id.String()+"@example.com")
	require.NoError(t, err)
	return id
}

func TestRegisterDeviceValidation(t *testing.T) {
	svc := NewService(nil) // validation runs before any query
	ctx := context.Background()
	user := uuid.New()

	tests := []struct {
		name       string
		token      string
		platform   string
		appVersion string
	}{
		{"unknown platform", deviceToken("a"), "symbian", ""},
		{"empty platform", deviceToken("a"), "", ""},
		{"token too short", "abc", PlatformAndroid, ""},
		{"token too long", strings.Repeat("t", 5000), PlatformAndroid, ""},
		{"app version too long", deviceToken("a"), PlatformAndroid, strings.Repeat("v", 41)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.RegisterDevice(ctx, user, tt.token, tt.platform, tt.appVersion)
			assert.Error(t, err)
		})
	}
}

func TestRegisterDevice(t *testing.T) {
	pool := testdb.New(t)
	svc := NewService(pool)
	ctx := context.Background()
	user := seedUser(t, pool, "Device Owner")

	t.Run("registers an install without echoing the token", func(t *testing.T) {
		d, err := svc.RegisterDevice(ctx, user, deviceToken("one"), PlatformAndroid, "1.0.0")

		require.NoError(t, err)
		assert.Equal(t, PlatformAndroid, d.Platform)
		assert.Equal(t, "1.0.0", d.AppVersion)
		assert.NotEqual(t, uuid.Nil, d.ID)
		// DeviceToken deliberately has no token field to serialize.
		assert.NotContains(t, fmt.Sprintf("%+v", d), "tttt")
	})

	t.Run("re-registering the same token is idempotent", func(t *testing.T) {
		first, err := svc.RegisterDevice(ctx, user, deviceToken("two"), PlatformAndroid, "1.0.0")
		require.NoError(t, err)

		again, err := svc.RegisterDevice(ctx, user, deviceToken("two"), PlatformAndroid, "1.1.0")

		require.NoError(t, err)
		assert.Equal(t, first.ID, again.ID, "the same device must not create a second row")
		assert.Equal(t, "1.1.0", again.AppVersion, "app version should refresh")
	})

	t.Run("a token moves to whoever signed in last", func(t *testing.T) {
		shared := deviceToken("shared")
		other := seedUser(t, pool, "Second Owner")
		_, err := svc.RegisterDevice(ctx, user, shared, PlatformAndroid, "")
		require.NoError(t, err)

		_, err = svc.RegisterDevice(ctx, other, shared, PlatformAndroid, "")
		require.NoError(t, err)

		// The first account must stop receiving that handset's pushes.
		firstTokens, err := svc.DeviceTokensFor(ctx, user)
		require.NoError(t, err)
		assert.NotContains(t, firstTokens, shared)

		secondTokens, err := svc.DeviceTokensFor(ctx, other)
		require.NoError(t, err)
		assert.Contains(t, secondTokens, shared)
	})
}

func TestDeviceLifecycle(t *testing.T) {
	pool := testdb.New(t)
	svc := NewService(pool)
	ctx := context.Background()
	user := seedUser(t, pool, "Lifecycle User")

	_, err := svc.RegisterDevice(ctx, user, deviceToken("phone"), PlatformAndroid, "1.0.0")
	require.NoError(t, err)
	_, err = svc.RegisterDevice(ctx, user, deviceToken("tablet"), PlatformIOS, "1.0.0")
	require.NoError(t, err)

	t.Run("lists both installs", func(t *testing.T) {
		items, err := svc.ListDevices(ctx, user)

		require.NoError(t, err)
		assert.Len(t, items, 2)
	})

	t.Run("unregistering removes one", func(t *testing.T) {
		require.NoError(t, svc.UnregisterDevice(ctx, user, deviceToken("phone")))

		items, err := svc.ListDevices(ctx, user)
		require.NoError(t, err)
		assert.Len(t, items, 1)
		assert.Equal(t, PlatformIOS, items[0].Platform)
	})

	t.Run("cannot unregister a token owned by someone else", func(t *testing.T) {
		other := seedUser(t, pool, "Bystander")
		_, err := svc.RegisterDevice(ctx, other, deviceToken("theirs"), PlatformAndroid, "")
		require.NoError(t, err)

		err = svc.UnregisterDevice(ctx, user, deviceToken("theirs"))

		assert.Error(t, err)
		theirs, listErr := svc.DeviceTokensFor(ctx, other)
		require.NoError(t, listErr)
		assert.Contains(t, theirs, deviceToken("theirs"), "the other user's token must survive")
	})

	t.Run("deleting stale tokens is a no-op on an empty list", func(t *testing.T) {
		assert.NoError(t, svc.DeleteDeviceTokens(ctx, nil))
	})

	t.Run("stale tokens are removed by the delivery provider", func(t *testing.T) {
		require.NoError(t, svc.DeleteDeviceTokens(ctx, []string{deviceToken("tablet")}))

		items, err := svc.ListDevices(ctx, user)
		require.NoError(t, err)
		assert.Empty(t, items)
	})
}

func TestRegisterDeviceCapsInstallsPerUser(t *testing.T) {
	pool := testdb.New(t)
	svc := NewService(pool)
	ctx := context.Background()
	user := seedUser(t, pool, "Prolific User")

	for i := 0; i < maxDevicesPerUser+5; i++ {
		_, err := svc.RegisterDevice(ctx, user, deviceToken(fmt.Sprintf("dev%02d", i)), PlatformAndroid, "")
		require.NoError(t, err)
	}

	items, err := svc.ListDevices(ctx, user)

	require.NoError(t, err)
	assert.Len(t, items, maxDevicesPerUser)
}
