package workosauth

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultAPIBase = "https://api.workos.com"

// Client talks to WorkOS User Management for AuthKit.
type Client struct {
	APIKey      string
	ClientID    string
	RedirectURI string
	HTTPClient  *http.Client
	APIBase     string
}

func (c *Client) Enabled() bool {
	return strings.TrimSpace(c.APIKey) != "" &&
		strings.TrimSpace(c.ClientID) != "" &&
		strings.TrimSpace(c.RedirectURI) != ""
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 15 * time.Second}
}

func (c *Client) apiBase() string {
	if strings.TrimSpace(c.APIBase) != "" {
		return strings.TrimRight(c.APIBase, "/")
	}
	return defaultAPIBase
}

// AuthorizationURL builds the AuthKit hosted sign-in/up URL.
// codeChallenge is the S256 PKCE challenge; state is the unguessable CSRF token.
func (c *Client) AuthorizationURL(screenHint, state, codeChallenge string) (string, error) {
	if !c.Enabled() {
		return "", fmt.Errorf("workos authkit is not configured")
	}
	values := url.Values{}
	values.Set("response_type", "code")
	values.Set("client_id", c.ClientID)
	values.Set("redirect_uri", c.RedirectURI)
	values.Set("provider", "authkit")
	if screenHint != "" {
		values.Set("screen_hint", screenHint)
	}
	if state != "" {
		values.Set("state", state)
	}
	if codeChallenge != "" {
		values.Set("code_challenge", codeChallenge)
		values.Set("code_challenge_method", "S256")
	}
	return c.apiBase() + "/user_management/authorize?" + values.Encode(), nil
}

type AuthenticateResult struct {
	User         User   `json:"user"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type User struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
}

type apiError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	Message          string `json:"message"`
}

func (c *Client) AuthenticateWithCode(ctx context.Context, code, codeVerifier, ipAddress, userAgent string) (*AuthenticateResult, error) {
	if !c.Enabled() {
		return nil, fmt.Errorf("workos authkit is not configured")
	}
	body := map[string]string{
		"client_id":     c.ClientID,
		"client_secret": c.APIKey,
		"grant_type":    "authorization_code",
		"code":          code,
	}
	if codeVerifier != "" {
		body["code_verifier"] = codeVerifier
	}
	if ipAddress != "" {
		body["ip_address"] = ipAddress
	}
	if userAgent != "" {
		body["user_agent"] = userAgent
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiBase()+"/user_management/authenticate", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr apiError
		_ = json.Unmarshal(raw, &apiErr)
		msg := apiErr.ErrorDescription
		if msg == "" {
			msg = apiErr.Message
		}
		if msg == "" {
			msg = apiErr.Error
		}
		if msg == "" {
			msg = strings.TrimSpace(string(raw))
		}
		if msg == "" {
			msg = "workos authentication failed"
		}
		return nil, fmt.Errorf("%s", msg)
	}
	var result AuthenticateResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	if result.User.ID == "" || result.User.Email == "" {
		return nil, fmt.Errorf("workos authentication returned an incomplete user")
	}
	if !result.User.EmailVerified {
		return nil, fmt.Errorf("workos email is not verified")
	}
	return &result, nil
}

func DisplayName(user User) string {
	name := strings.TrimSpace(strings.TrimSpace(user.FirstName) + " " + strings.TrimSpace(user.LastName))
	if name != "" {
		return name
	}
	return user.Email
}

// SessionID returns the AuthKit session id (`sid` claim) carried by a WorkOS
// access token, or "" when absent. The token comes straight from the
// authenticate response over TLS and is only used to end that session later,
// so its signature is not checked here.
func SessionID(accessToken string) string {
	parts := strings.Split(accessToken, ".")
	if len(parts) != 3 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims struct {
		SID string `json:"sid"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	return strings.TrimSpace(claims.SID)
}

// LogoutURL builds the hosted AuthKit logout URL for sessionID. WorkOS ends
// the session and redirects to the logout redirect configured in the
// dashboard. Returns "" when AuthKit is off or there is no session.
func (c *Client) LogoutURL(sessionID string) string {
	sessionID = strings.TrimSpace(sessionID)
	if !c.Enabled() || sessionID == "" {
		return ""
	}
	values := url.Values{}
	values.Set("session_id", sessionID)
	return c.apiBase() + "/user_management/sessions/logout?" + values.Encode()
}
