package notifications

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// FCM endpoints. The send URL takes the project ID.
const (
	defaultFCMSendURL  = "https://fcm.googleapis.com/v1/projects/%s/messages:send"
	fcmScope           = "https://www.googleapis.com/auth/firebase.messaging"
	jwtBearerGrantType = "urn:ietf:params:oauth:grant-type:jwt-bearer"
)

// errTokenGone marks a device token the push service has permanently
// rejected, so it can be deleted instead of retried.
var errTokenGone = errors.New("device token no longer registered")

// deviceRegistry is the slice of Service the push provider needs, kept narrow
// so the provider can be tested without a database.
type deviceRegistry interface {
	DeviceTokensFor(ctx context.Context, userID uuid.UUID) ([]string, error)
	DeleteDeviceTokens(ctx context.Context, tokens []string) error
}

// serviceAccount is the subset of a Google service-account key file needed to
// mint access tokens.
type serviceAccount struct {
	Type        string `json:"type"`
	ProjectID   string `json:"project_id"`
	PrivateKey  string `json:"private_key"`
	ClientEmail string `json:"client_email"`
	TokenURI    string `json:"token_uri"`
}

// FCMProvider delivers outbox events to Firebase Cloud Messaging.
//
// Messages are **data-only**: no notification title or body is sent. The
// inbox is designed so clients translate event_type and interpolate data
// themselves, and rendering text here would move localization onto the server
// for three languages. The cost is that Android will not display these
// automatically — the app has to build the notification — and data-only
// messages are treated as lower priority when the app is not running, which
// is why android.priority is set to high.
type FCMProvider struct {
	projectID   string
	clientEmail string
	privateKey  *rsa.PrivateKey
	tokenURL    string
	sendURL     string

	registry deviceRegistry
	client   *http.Client

	mu          sync.Mutex
	accessToken string
	expiresAt   time.Time
}

// NewFCMProvider builds a provider from the contents of a service-account key
// file.
func NewFCMProvider(credentials []byte, registry deviceRegistry, timeout time.Duration) (*FCMProvider, error) {
	var sa serviceAccount
	if err := json.Unmarshal(credentials, &sa); err != nil {
		return nil, fmt.Errorf("parsing fcm credentials: %w", err)
	}
	if sa.Type != "service_account" {
		return nil, fmt.Errorf("fcm credentials must be a service_account key, got %q", sa.Type)
	}
	if sa.ProjectID == "" || sa.ClientEmail == "" || sa.PrivateKey == "" {
		return nil, errors.New("fcm credentials missing project_id, client_email, or private_key")
	}
	key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(sa.PrivateKey))
	if err != nil {
		return nil, fmt.Errorf("parsing fcm private key: %w", err)
	}
	if sa.TokenURI == "" {
		sa.TokenURI = "https://oauth2.googleapis.com/token"
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &FCMProvider{
		projectID:   sa.ProjectID,
		clientEmail: sa.ClientEmail,
		privateKey:  key,
		tokenURL:    sa.TokenURI,
		sendURL:     defaultFCMSendURL,
		registry:    registry,
		client:      &http.Client{Timeout: timeout},
	}, nil
}

// ProjectID reports the Firebase project the provider sends to.
func (p *FCMProvider) ProjectID() string { return p.projectID }

// outboxPayload mirrors the JSON written by EnqueueTx.
type outboxPayload struct {
	NotificationID uuid.UUID         `json:"notification_id"`
	UserID         uuid.UUID         `json:"user_id"`
	EventType      string            `json:"event_type"`
	SubjectType    string            `json:"subject_type"`
	SubjectID      uuid.UUID         `json:"subject_id"`
	Data           map[string]string `json:"data"`
}

// Send pushes one outbox event to every device the recipient has registered.
//
// An event for a user with no registered device is reported delivered: most
// accounts never install the app, and failing would make the outbox retry
// them forever.
func (p *FCMProvider) Send(ctx context.Context, event OutboxEvent) error {
	var payload outboxPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		// Undecodable payloads will never succeed, so do not ask for a retry.
		slog.ErrorContext(ctx, "discarding push for undecodable outbox payload",
			"event_id", event.ID, "error", err)
		return nil
	}

	tokens, err := p.registry.DeviceTokensFor(ctx, payload.UserID)
	if err != nil {
		return fmt.Errorf("loading device tokens: %w", err)
	}
	if len(tokens) == 0 {
		return nil
	}

	body, err := p.messageData(payload)
	if err != nil {
		return err
	}

	var (
		stale     []string
		delivered int
		failures  []error
	)
	for _, token := range tokens {
		switch err := p.sendToToken(ctx, token, body); {
		case err == nil:
			delivered++
		case errors.Is(err, errTokenGone):
			stale = append(stale, token)
		default:
			failures = append(failures, err)
		}
	}

	if len(stale) > 0 {
		// Pruning is best-effort: failing here would retry a push that
		// already succeeded for the other devices.
		if err := p.registry.DeleteDeviceTokens(ctx, stale); err != nil {
			slog.ErrorContext(ctx, "pruning stale device tokens", "count", len(stale), "error", err)
		}
	}
	if delivered > 0 {
		return nil
	}
	if len(failures) > 0 {
		return fmt.Errorf("pushing to %d device(s): %w", len(failures), errors.Join(failures...))
	}
	// Every token was permanently gone; a retry cannot change that.
	return nil
}

// messageData renders the FCM data payload. Every value must be a string, so
// the nested data map travels as JSON for the client to decode.
func (p *FCMProvider) messageData(payload outboxPayload) (map[string]string, error) {
	nested, err := json.Marshal(payload.Data)
	if err != nil {
		return nil, fmt.Errorf("encoding notification data: %w", err)
	}
	return map[string]string{
		"notification_id": payload.NotificationID.String(),
		// The recipient is included so a client can drop a push that arrives
		// for an account other than the one currently signed in.
		"user_id":      payload.UserID.String(),
		"event_type":   payload.EventType,
		"subject_type": payload.SubjectType,
		"subject_id":   payload.SubjectID.String(),
		"data":         string(nested),
	}, nil
}

// sendToToken delivers one message, classifying the failure so the caller can
// tell "prune this token" from "try again later".
func (p *FCMProvider) sendToToken(ctx context.Context, token string, data map[string]string) error {
	accessToken, err := p.ensureAccessToken(ctx)
	if err != nil {
		return err
	}
	body, err := json.Marshal(map[string]any{
		"message": map[string]any{
			"token":   token,
			"data":    data,
			"android": map[string]any{"priority": "high"},
		},
	})
	if err != nil {
		return fmt.Errorf("encoding fcm message: %w", err)
	}

	endpoint := fmt.Sprintf(p.sendURL, p.projectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("building fcm request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("calling fcm: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))

	switch {
	case resp.StatusCode < 300:
		return nil
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		// The cached token may have been revoked; drop it so the next
		// attempt mints a fresh one.
		p.invalidateAccessToken()
		return fmt.Errorf("fcm rejected credentials: %s", resp.Status)
	case isTokenGone(resp.StatusCode, respBody):
		return errTokenGone
	default:
		return fmt.Errorf("fcm returned %s: %s", resp.Status, truncateError(errors.New(string(respBody))))
	}
}

// isTokenGone reports whether FCM permanently rejected the device token.
func isTokenGone(status int, body []byte) bool {
	if status != http.StatusNotFound && status != http.StatusBadRequest {
		return false
	}
	text := string(body)
	return strings.Contains(text, "UNREGISTERED") ||
		strings.Contains(text, "NOT_FOUND") ||
		strings.Contains(text, "INVALID_ARGUMENT")
}

// ensureAccessToken returns a cached OAuth2 access token, minting a new one
// when it is missing or close to expiry.
func (p *FCMProvider) ensureAccessToken(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// Refresh a minute early so a token cannot expire mid-flight.
	if p.accessToken != "" && time.Now().Add(time.Minute).Before(p.expiresAt) {
		return p.accessToken, nil
	}

	assertion, err := p.signAssertion()
	if err != nil {
		return "", err
	}
	form := url.Values{"grant_type": {jwtBearerGrantType}, "assertion": {assertion}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("building token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("requesting fcm access token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("fcm token endpoint returned %s", resp.Status)
	}
	var minted struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &minted); err != nil {
		return "", fmt.Errorf("decoding fcm access token: %w", err)
	}
	if minted.AccessToken == "" {
		return "", errors.New("fcm token endpoint returned no access token")
	}
	if minted.ExpiresIn <= 0 {
		minted.ExpiresIn = 3600
	}
	p.accessToken = minted.AccessToken
	p.expiresAt = time.Now().Add(time.Duration(minted.ExpiresIn) * time.Second)
	return p.accessToken, nil
}

func (p *FCMProvider) invalidateAccessToken() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.accessToken = ""
	p.expiresAt = time.Time{}
}

// signAssertion builds the RS256 JWT that is exchanged for an access token.
func (p *FCMProvider) signAssertion() (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   p.clientEmail,
		"scope": fcmScope,
		"aud":   p.tokenURL,
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(p.privateKey)
	if err != nil {
		return "", fmt.Errorf("signing fcm assertion: %w", err)
	}
	return signed, nil
}
