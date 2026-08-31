package notifications

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRegistry stands in for the device-token store.
type fakeRegistry struct {
	tokens  map[uuid.UUID][]string
	deleted []string
	listErr error
}

func (f *fakeRegistry) DeviceTokensFor(_ context.Context, userID uuid.UUID) ([]string, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.tokens[userID], nil
}

func (f *fakeRegistry) DeleteDeviceTokens(_ context.Context, tokens []string) error {
	f.deleted = append(f.deleted, tokens...)
	return nil
}

// testCredentials builds a service-account key file with a fresh RSA key.
func testCredentials(t *testing.T) []byte {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})

	raw, err := json.Marshal(map[string]string{
		"type":         "service_account",
		"project_id":   "adera-test",
		"client_email": "pusher@adera-test.iam.gserviceaccount.com",
		"private_key":  string(keyPEM),
		"token_uri":    "https://oauth2.googleapis.com/token",
	})
	require.NoError(t, err)
	return raw
}

// fcmServer fakes the OAuth2 token endpoint and the FCM send endpoint.
type fcmServer struct {
	srv        *httptest.Server
	tokenCalls atomic.Int32
	sendCalls  atomic.Int32
	sent       chan map[string]any
	status     atomic.Int32
	body       atomic.Value // string
}

func newFCMServer(t *testing.T) *fcmServer {
	t.Helper()
	f := &fcmServer{sent: make(chan map[string]any, 16)}
	f.status.Store(http.StatusOK)
	f.body.Store(`{}`)

	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		f.tokenCalls.Add(1)
		require.NoError(t, r.ParseForm())
		assert.Equal(t, jwtBearerGrantType, r.PostForm.Get("grant_type"))
		assert.NotEmpty(t, r.PostForm.Get("assertion"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"minted-access-token","expires_in":3600}`))
	})
	mux.HandleFunc("/v1/projects/{project}/messages:send", func(w http.ResponseWriter, r *http.Request) {
		f.sendCalls.Add(1)
		assert.Equal(t, "Bearer minted-access-token", r.Header.Get("Authorization"))
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		f.sent <- payload
		w.WriteHeader(int(f.status.Load()))
		_, _ = w.Write([]byte(f.body.Load().(string)))
	})

	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

func newTestProvider(t *testing.T, reg deviceRegistry, srv *fcmServer) *FCMProvider {
	t.Helper()
	p, err := NewFCMProvider(testCredentials(t), reg, 5*time.Second)
	require.NoError(t, err)
	p.tokenURL = srv.srv.URL + "/token"
	p.sendURL = srv.srv.URL + "/v1/projects/%s/messages:send"
	return p
}

func testEvent(t *testing.T, userID uuid.UUID) OutboxEvent {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"notification_id": uuid.New(),
		"user_id":         userID,
		"event_type":      "owner_response_created",
		"subject_type":    "review",
		"subject_id":      uuid.New(),
		"data":            map[string]string{"target_name": "ካተኛ"},
	})
	require.NoError(t, err)
	return OutboxEvent{ID: uuid.New(), Topic: "notification.created", Payload: payload}
}

func TestNewFCMProviderValidation(t *testing.T) {
	tests := []struct {
		name        string
		credentials string
	}{
		{"not json", "{"},
		{"wrong credential type", `{"type":"authorized_user","project_id":"p","client_email":"e","private_key":"k"}`},
		{"missing project", `{"type":"service_account","client_email":"e","private_key":"k"}`},
		{"unparseable private key", `{"type":"service_account","project_id":"p","client_email":"e","private_key":"not-a-pem"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewFCMProvider([]byte(tt.credentials), &fakeRegistry{}, time.Second)
			assert.Error(t, err)
		})
	}

	t.Run("accepts a service account key", func(t *testing.T) {
		p, err := NewFCMProvider(testCredentials(t), &fakeRegistry{}, time.Second)
		require.NoError(t, err)
		assert.Equal(t, "adera-test", p.ProjectID())
	})
}

func TestFCMSendWithoutDevicesIsDelivered(t *testing.T) {
	srv := newFCMServer(t)
	reg := &fakeRegistry{tokens: map[uuid.UUID][]string{}}
	p := newTestProvider(t, reg, srv)
	user := uuid.New()

	// Most accounts never install the app; failing here would retry forever.
	err := p.Send(context.Background(), testEvent(t, user))

	require.NoError(t, err)
	assert.Zero(t, srv.sendCalls.Load(), "no push should be attempted")
}

func TestFCMSendDataOnlyMessage(t *testing.T) {
	srv := newFCMServer(t)
	user := uuid.New()
	reg := &fakeRegistry{tokens: map[uuid.UUID][]string{user: {"device-a"}}}
	p := newTestProvider(t, reg, srv)

	require.NoError(t, p.Send(context.Background(), testEvent(t, user)))

	payload := <-srv.sent
	message := payload["message"].(map[string]any)
	assert.Equal(t, "device-a", message["token"])
	assert.Equal(t, "high", message["android"].(map[string]any)["priority"])
	// No server-rendered text: the client localizes from event_type.
	assert.NotContains(t, message, "notification")

	data := message["data"].(map[string]any)
	assert.Equal(t, "owner_response_created", data["event_type"])
	assert.Equal(t, "review", data["subject_type"])
	assert.Equal(t, user.String(), data["user_id"])
	// The nested data map travels as a JSON string, since FCM data values
	// must all be strings.
	var nested map[string]string
	require.NoError(t, json.Unmarshal([]byte(data["data"].(string)), &nested))
	assert.Equal(t, "ካተኛ", nested["target_name"])
}

func TestFCMPrunesUnregisteredTokens(t *testing.T) {
	srv := newFCMServer(t)
	srv.status.Store(http.StatusNotFound)
	srv.body.Store(`{"error":{"status":"UNREGISTERED","message":"gone"}}`)
	user := uuid.New()
	reg := &fakeRegistry{tokens: map[uuid.UUID][]string{user: {"dead-token"}}}
	p := newTestProvider(t, reg, srv)

	err := p.Send(context.Background(), testEvent(t, user))

	// Retrying a dead token can never succeed, so the event is done.
	require.NoError(t, err)
	assert.Equal(t, []string{"dead-token"}, reg.deleted)
}

func TestFCMRetriesOnServerError(t *testing.T) {
	srv := newFCMServer(t)
	srv.status.Store(http.StatusInternalServerError)
	srv.body.Store(`{"error":{"status":"INTERNAL"}}`)
	user := uuid.New()
	reg := &fakeRegistry{tokens: map[uuid.UUID][]string{user: {"device-a"}}}
	p := newTestProvider(t, reg, srv)

	err := p.Send(context.Background(), testEvent(t, user))

	assert.Error(t, err, "a transient failure must leave the event in the outbox")
	assert.Empty(t, reg.deleted, "a transient failure must not prune the token")
}

func TestFCMSucceedsWhenAnyDeviceAccepts(t *testing.T) {
	srv := newFCMServer(t)
	user := uuid.New()
	reg := &fakeRegistry{tokens: map[uuid.UUID][]string{user: {"good", "dead"}}}
	p := newTestProvider(t, reg, srv)

	// Fail only the second device.
	var calls atomic.Int32
	srv.srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			_, _ = w.Write([]byte(`{"access_token":"minted-access-token","expires_in":3600}`))
			return
		}
		if calls.Add(1) == 1 {
			_, _ = w.Write([]byte(`{}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"status":"UNREGISTERED"}}`))
	})

	err := p.Send(context.Background(), testEvent(t, user))

	require.NoError(t, err)
	assert.Equal(t, []string{"dead"}, reg.deleted)
}

func TestFCMCachesAccessTokenAcrossSends(t *testing.T) {
	srv := newFCMServer(t)
	user := uuid.New()
	reg := &fakeRegistry{tokens: map[uuid.UUID][]string{user: {"device-a"}}}
	p := newTestProvider(t, reg, srv)

	for i := 0; i < 3; i++ {
		require.NoError(t, p.Send(context.Background(), testEvent(t, user)))
		<-srv.sent
	}

	assert.Equal(t, int32(1), srv.tokenCalls.Load(), "the access token should be minted once")
	assert.Equal(t, int32(3), srv.sendCalls.Load())
}

func TestFCMDiscardsUndecodablePayload(t *testing.T) {
	srv := newFCMServer(t)
	p := newTestProvider(t, &fakeRegistry{}, srv)

	err := p.Send(context.Background(), OutboxEvent{ID: uuid.New(), Payload: []byte(`not json`)})

	// A payload that cannot be decoded will never decode, so do not retry.
	require.NoError(t, err)
	assert.Zero(t, srv.sendCalls.Load())
}

func TestFCMPropagatesRegistryFailure(t *testing.T) {
	srv := newFCMServer(t)
	reg := &fakeRegistry{listErr: assert.AnError}
	p := newTestProvider(t, reg, srv)

	err := p.Send(context.Background(), testEvent(t, uuid.New()))

	assert.Error(t, err, "a database failure must leave the event for retry")
}
