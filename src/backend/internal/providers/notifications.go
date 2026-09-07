package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	webpush "github.com/marknefedov/go-webpush/v2"
	"github.com/roomies/backend/internal/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type PushTarget struct{ Platform, Endpoint, P256DH, Auth, Token string }
type PushResult struct {
	Revoke     bool
	RetryAfter time.Time
	Fake       bool
}

type TransientDeliveryError struct {
	Cause   error
	RetryAt time.Time
}

func (e *TransientDeliveryError) Error() string { return e.Cause.Error() }
func (e *TransientDeliveryError) Unwrap() error { return e.Cause }

type NotificationProvider interface {
	Dispatch(context.Context, PushTarget, []byte) (PushResult, error)
	Fake() bool
}
type FakeNotificationProvider struct{}

func (*FakeNotificationProvider) Dispatch(context.Context, PushTarget, []byte) (PushResult, error) {
	return PushResult{Fake: true}, nil
}
func (*FakeNotificationProvider) Fake() bool { return true }

type HTTPNotificationProvider struct {
	web       *webpush.Client
	vapid     *webpush.VAPIDKeys
	subject   string
	fcmClient *http.Client
	fcmURL    string
}

const notificationHTTPTimeout = 10 * time.Second

func NewNotificationProvider(ctx context.Context, cfg *config.Config) (NotificationProvider, error) {
	if cfg.WebPushPrivateKey == "" && cfg.FCMProjectID == "" {
		return &FakeNotificationProvider{}, nil
	}
	p := &HTTPNotificationProvider{web: webpush.NewClient(webpush.Config{HTTPClient: &http.Client{Timeout: notificationHTTPTimeout}}), subject: cfg.WebPushSubject}
	if cfg.WebPushPrivateKey != "" {
		raw, _ := json.Marshal(map[string]string{"privateKey": cfg.WebPushPrivateKey, "publicKey": cfg.WebPushPublicKey})
		var keys webpush.VAPIDKeys
		if err := json.Unmarshal(raw, &keys); err != nil {
			return nil, fmt.Errorf("parse VAPID key: %w", err)
		}
		p.vapid = &keys
	}
	if cfg.FCMProjectID != "" {
		credentials, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/firebase.messaging")
		if err != nil {
			return nil, fmt.Errorf("load FCM application default credentials: %w", err)
		}
		p.fcmClient = oauth2.NewClient(ctx, credentials.TokenSource)
		p.fcmClient.Timeout = notificationHTTPTimeout
		p.fcmURL = "https://fcm.googleapis.com/v1/projects/" + url.PathEscape(cfg.FCMProjectID) + "/messages:send"
	}
	return p, nil
}
func (p *HTTPNotificationProvider) Fake() bool { return false }
func (p *HTTPNotificationProvider) Dispatch(ctx context.Context, target PushTarget, payload []byte) (PushResult, error) {
	if len(payload) > 2048 {
		return PushResult{}, errors.New("notification payload exceeds 2 KiB")
	}
	switch target.Platform {
	case "web_push":
		return p.webPush(ctx, target, payload)
	case "android_fcm":
		return p.fcm(ctx, target, payload)
	default:
		return PushResult{}, errors.New("unknown notification platform")
	}
}
func (p *HTTPNotificationProvider) webPush(ctx context.Context, target PushTarget, payload []byte) (PushResult, error) {
	if p.vapid == nil {
		return PushResult{Fake: true}, nil
	}
	keys, err := webpush.DecodeSubscriptionKeys(target.Auth, target.P256DH)
	if err != nil {
		return PushResult{Revoke: true}, nil
	}
	_, err = p.web.Send(ctx, payload, &webpush.Subscription{Endpoint: target.Endpoint, Keys: keys}, webpush.SendOptions{VAPIDKeys: p.vapid, Subject: p.subject, TTL: 300})
	if err == nil {
		return PushResult{}, nil
	}
	var serviceErr *webpush.PushServiceError
	if errors.As(err, &serviceErr) {
		if serviceErr.StatusCode == 404 || serviceErr.StatusCode == 410 {
			return PushResult{Revoke: true}, nil
		}
		if serviceErr.StatusCode == 429 || serviceErr.StatusCode >= 500 {
			return PushResult{RetryAfter: serviceErr.RetryAfter}, &TransientDeliveryError{Cause: err, RetryAt: serviceErr.RetryAfter}
		}
		return PushResult{}, fmt.Errorf("web push HTTP %d: %w", serviceErr.StatusCode, err)
	}
	return PushResult{}, err
}

type fcmErrorResponse struct {
	Error struct {
		Status  string `json:"status"`
		Details []struct {
			Type      string `json:"@type"`
			ErrorCode string `json:"errorCode"`
		} `json:"details"`
	} `json:"error"`
}

func (p *HTTPNotificationProvider) fcm(ctx context.Context, target PushTarget, payload []byte) (PushResult, error) {
	if p.fcmClient == nil {
		return PushResult{Fake: true}, nil
	}
	if strings.TrimSpace(target.Token) == "" {
		return PushResult{}, errors.New("FCM token is empty")
	}
	var data map[string]any
	if err := json.Unmarshal(payload, &data); err != nil {
		return PushResult{}, err
	}
	payloadValid := validFCMDataPayload(data)
	if !payloadValid {
		return PushResult{}, errors.New("FCM data payload is invalid")
	}
	stringData := map[string]string{}
	for k, v := range data {
		switch value := v.(type) {
		case string:
			stringData[k] = value
		default:
			encoded, err := json.Marshal(value)
			if err != nil {
				return PushResult{}, errors.New("FCM data payload contains an unsupported value")
			}
			stringData[k] = string(encoded)
		}
	}
	body, _ := json.Marshal(map[string]any{"message": map[string]any{"token": target.Token, "data": stringData}})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.fcmURL, bytes.NewReader(body))
	if err != nil {
		return PushResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.fcmClient.Do(req)
	if err != nil {
		return PushResult{}, err
	}
	defer resp.Body.Close()
	responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return PushResult{}, nil
	}
	var providerError fcmErrorResponse
	_ = json.Unmarshal(responseBody, &providerError)
	errorCode := ""
	for _, detail := range providerError.Error.Details {
		if detail.Type == "type.googleapis.com/google.firebase.fcm.v1.FcmError" {
			errorCode = detail.ErrorCode
			break
		}
	}
	if errorCode == "UNREGISTERED" {
		return PushResult{Revoke: true}, nil
	}
	if resp.StatusCode == http.StatusBadRequest && errorCode == "INVALID_ARGUMENT" && payloadValid {
		return PushResult{Revoke: true}, nil
	}
	if resp.StatusCode == 429 || resp.StatusCode >= 500 {
		retryAt := parseRetryAfter(resp.Header.Get("Retry-After"))
		return PushResult{RetryAfter: retryAt}, &TransientDeliveryError{Cause: fmt.Errorf("FCM transient HTTP %d", resp.StatusCode), RetryAt: retryAt}
	}
	return PushResult{}, fmt.Errorf("FCM HTTP %d (%s)", resp.StatusCode, providerError.Error.Status)
}

func validFCMDataPayload(data map[string]any) bool {
	if len(data) == 0 {
		return false
	}
	for key := range data {
		lower := strings.ToLower(key)
		if key == "" || lower == "from" || strings.HasPrefix(lower, "google.") || strings.HasPrefix(lower, "gcm.") {
			return false
		}
	}
	return true
}

func parseRetryAfter(value string) time.Time {
	if seconds, err := time.ParseDuration(strings.TrimSpace(value) + "s"); err == nil {
		return time.Now().Add(seconds)
	}
	when, _ := http.ParseTime(value)
	return when
}
