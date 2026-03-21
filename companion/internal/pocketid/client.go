package pocketid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"
)

const defaultTimeout = 5 * time.Second

// User represents a Pocket ID user.
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// userinfoResponse maps OIDC standard userinfo claims.
type userinfoResponse struct {
	Sub               string `json:"sub"`
	PreferredUsername string `json:"preferred_username"`
	Name              string `json:"name"`
	Email             string `json:"email"`
}

// Group represents a Pocket ID user group.
type Group struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	FriendlyName string `json:"friendlyName"`
}

// groupsPage is the paginated response from /api/user-groups.
type groupsPage struct {
	Data []Group `json:"data"`
}

// CustomClaim is a single key/value custom claim for a user.
type CustomClaim struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Client is a Pocket ID API client.
type Client struct {
	baseURL    string
	adminKey   string
	httpClient *http.Client
}

// New creates a new Client.
func New(baseURL, adminKey string) *Client {
	return &Client{
		baseURL:  baseURL,
		adminKey: adminKey,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// GetCurrentUser validates an OIDC Bearer token via the userinfo endpoint
// and returns the associated user.
func (c *Client) GetCurrentUser(ctx context.Context, bearerToken string) (*User, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/oidc/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+bearerToken)

	var info userinfoResponse
	if err := c.do(req, &info); err != nil {
		return nil, fmt.Errorf("get userinfo: %w", err)
	}

	username := info.PreferredUsername
	if username == "" {
		username = info.Name
	}

	return &User{
		ID:       info.Sub,
		Username: username,
		Email:    info.Email,
	}, nil
}

// GetUserGroups returns all groups the given user belongs to.
func (c *Client) GetUserGroups(ctx context.Context, userID string) ([]Group, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.baseURL+"/api/users/"+url.PathEscape(userID)+"/groups", nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	c.setAdminAuth(req)

	var groups []Group
	if err := c.do(req, &groups); err != nil {
		return nil, fmt.Errorf("get user groups: %w", err)
	}
	return groups, nil
}

// SetUserGroups replaces all groups for a user.
func (c *Client) SetUserGroups(ctx context.Context, userID string, groupIDs []string) error {
	body := struct {
		UserGroupIDs []string `json:"userGroupIds"`
	}{UserGroupIDs: groupIDs}

	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut,
		c.baseURL+"/api/users/"+url.PathEscape(userID)+"/user-groups",
		bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	c.setAdminAuth(req)
	req.Header.Set("Content-Type", "application/json")

	if err := c.do(req, nil); err != nil {
		return fmt.Errorf("set user groups: %w", err)
	}
	return nil
}

// FindGroupByName searches for a group with the given exact name.
// Returns nil if not found.
func (c *Client) FindGroupByName(ctx context.Context, name string) (*Group, error) {
	u := c.baseURL + "/api/user-groups?search=" + url.QueryEscape(name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	c.setAdminAuth(req)

	var page groupsPage
	if err := c.do(req, &page); err != nil {
		return nil, fmt.Errorf("search groups: %w", err)
	}

	for i := range page.Data {
		if page.Data[i].Name == name {
			return &page.Data[i], nil
		}
	}
	return nil, nil
}

// CreateGroup creates a new user group and returns it.
func (c *Client) CreateGroup(ctx context.Context, name, friendlyName string) (*Group, error) {
	body := struct {
		Name         string `json:"name"`
		FriendlyName string `json:"friendlyName"`
	}{Name: name, FriendlyName: friendlyName}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/user-groups", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	c.setAdminAuth(req)
	req.Header.Set("Content-Type", "application/json")

	var group Group
	if err := c.do(req, &group); err != nil {
		return nil, fmt.Errorf("create group: %w", err)
	}
	return &group, nil
}

// EnsureGroupExists finds or creates the named group, returning it either way.
func (c *Client) EnsureGroupExists(ctx context.Context, name, friendlyName string) (*Group, error) {
	existing, err := c.FindGroupByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	return c.CreateGroup(ctx, name, friendlyName)
}

// SetCustomClaims replaces all custom claims for a user.
func (c *Client) SetCustomClaims(ctx context.Context, userID string, claims []CustomClaim) error {
	data, err := json.Marshal(claims)
	if err != nil {
		return fmt.Errorf("marshal claims: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut,
		c.baseURL+"/api/custom-claims/user/"+url.PathEscape(userID),
		bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	c.setAdminAuth(req)
	req.Header.Set("Content-Type", "application/json")

	if err := c.do(req, nil); err != nil {
		return fmt.Errorf("set custom claims: %w", err)
	}
	return nil
}

// SignupToken is a single-use Pocket ID registration token.
type SignupToken struct {
	ID         string    `json:"id"`
	Token      string    `json:"token"`
	ExpiresAt  time.Time `json:"expiresAt"`
	UsageLimit int       `json:"usageLimit"`
	UsageCount int       `json:"usageCount"`
}

// CreateSignupToken creates a 1-use signup token that expires in 1 hour.
// The token can be used at <pocket_id_public_url>/signup?token=<token>.
func (c *Client) CreateSignupToken(ctx context.Context) (*SignupToken, error) {
	body := struct {
		ExpiresAt  string `json:"expiresAt"`
		UsageLimit int    `json:"usageLimit"`
	}{
		ExpiresAt:  time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
		UsageLimit: 1,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/signup-tokens", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	c.setAdminAuth(req)
	req.Header.Set("Content-Type", "application/json")

	var st SignupToken
	if err := c.do(req, &st); err != nil {
		return nil, fmt.Errorf("create signup token: %w", err)
	}
	return &st, nil
}

// GetUser returns the full Pocket ID user record for the given user ID.
// The returned UserDetail includes custom claims and group memberships.
func (c *Client) GetUser(ctx context.Context, userID string) (*UserDetail, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.baseURL+"/api/users/"+url.PathEscape(userID), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	c.setAdminAuth(req)

	var u UserDetail
	if err := c.do(req, &u); err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &u, nil
}

// UserDetail is the full Pocket ID user object returned by the admin API.
type UserDetail struct {
	ID           string        `json:"id"`
	Username     string        `json:"username"`
	Email        string        `json:"email"`
	CustomClaims []CustomClaim `json:"customClaims"`
}

// SetProfilePicture downloads imageURL and uploads it as the Pocket ID profile
// picture for the given user. The upload uses PUT /api/users/{id}/profile-picture
// with a multipart/form-data body containing a field named "file".
// A timeout longer than the default is used because the RSI CDN can be slow.
func (c *Client) SetProfilePicture(ctx context.Context, userID, imageURL string) error {
	// Fetch the image with a generous timeout.
	fetchClient := &http.Client{Timeout: 15 * time.Second}
	imgReq, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return fmt.Errorf("build image fetch request: %w", err)
	}
	imgReq.Header.Set("User-Agent", "SCID/1.0 (+https://scid.my)")
	imgResp, err := fetchClient.Do(imgReq)
	if err != nil {
		return fmt.Errorf("fetch image: %w", err)
	}
	defer imgResp.Body.Close()
	if imgResp.StatusCode >= 400 {
		return fmt.Errorf("image fetch returned %d", imgResp.StatusCode)
	}

	// Build the multipart body in memory.
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("file", "avatar.jpg")
	if err != nil {
		return fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, imgResp.Body); err != nil {
		return fmt.Errorf("copy image data: %w", err)
	}
	mw.Close()

	uploadURL := c.baseURL + "/api/users/" + url.PathEscape(userID) + "/profile-picture"
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, &buf)
	if err != nil {
		return fmt.Errorf("build upload request: %w", err)
	}
	c.setAdminAuth(req)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	if err := c.do(req, nil); err != nil {
		return fmt.Errorf("upload profile picture: %w", err)
	}
	return nil
}

func (c *Client) setAdminAuth(req *http.Request) {
	req.Header.Set("X-API-Key", c.adminKey)
}

// do executes req, checks the status, and JSON-decodes the response body into
// dest (if dest is non-nil).
func (c *Client) do(req *http.Request, dest any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("pocket id responded %d: %s", resp.StatusCode, sanitizeBody(body))
	}

	if dest != nil {
		if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

// sanitizeBody truncates a response body for safe inclusion in error messages.
func sanitizeBody(b []byte) string {
	if len(b) > 200 {
		return string(b[:200]) + "\u2026"
	}
	return string(b)
}
