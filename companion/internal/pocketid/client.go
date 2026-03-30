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
	"strings"
	"time"

	"github.com/lhw/scid/companion/internal/rsi"
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

// TokenResponse is returned by the OIDC token endpoint.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// ExchangeAuthorizationCode exchanges an authorization code plus PKCE verifier
// for an access token.
func (c *Client) ExchangeAuthorizationCode(ctx context.Context, clientID, redirectURI, code, codeVerifier string) (*TokenResponse, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {clientID},
		"redirect_uri":  {redirectURI},
		"code":          {code},
		"code_verifier": {codeVerifier},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/oidc/token", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var token TokenResponse
	if err := c.do(req, &token); err != nil {
		return nil, fmt.Errorf("exchange auth code: %w", err)
	}
	return &token, nil
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
	if !rsi.IsAllowedImageURL(imageURL) {
		return fmt.Errorf("refusing to fetch image from untrusted URL: %s", imageURL)
	}

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

// OIDCClientParams holds the fields for creating or updating an OIDC client.
type OIDCClientParams struct {
	Name               string   `json:"name"`
	LaunchURL          *string  `json:"launchURL"`
	IsPublic           bool     `json:"isPublic"`
	PkceEnabled        bool     `json:"pkceEnabled"`
	CallbackURLs       []string `json:"callbackURLs"`
	LogoutCallbackURLs []string `json:"logoutCallbackURLs"`
}

// OIDCClient is returned by the Pocket ID admin API for OIDC client operations.
// Note: the ID field is the OIDC client_id (the UUID clients use to authenticate).
type OIDCClient struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	LaunchURL          *string  `json:"launchURL"`
	HasLogo            bool     `json:"hasLogo"`
	IsPublic           bool     `json:"isPublic"`
	PkceEnabled        bool     `json:"pkceEnabled"`
	CallbackURLs       []string `json:"callbackURLs"`
	LogoutCallbackURLs []string `json:"logoutCallbackURLs"`
	IsGroupRestricted  bool     `json:"isGroupRestricted"`
	AllowedUserGroups  []Group  `json:"allowedUserGroups"`
}

// oidcClientPage is the paginated OIDC clients list response.
type oidcClientPage struct {
	Data []OIDCClient `json:"data"`
}

// CreateOIDCClient creates a new OIDC client.
// The returned client's ID is also the OIDC client_id.
func (c *Client) CreateOIDCClient(ctx context.Context, params OIDCClientParams) (*OIDCClient, error) {
	data, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("marshal params: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/oidc/clients", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	c.setAdminAuth(req)
	req.Header.Set("Content-Type", "application/json")

	var client OIDCClient
	if err := c.do(req, &client); err != nil {
		return nil, fmt.Errorf("create oidc client: %w", err)
	}
	return &client, nil
}

// GetOIDCClient retrieves an OIDC client by its internal ID.
func (c *Client) GetOIDCClient(ctx context.Context, id string) (*OIDCClient, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.baseURL+"/api/oidc/clients/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	c.setAdminAuth(req)

	var client OIDCClient
	if err := c.do(req, &client); err != nil {
		return nil, fmt.Errorf("get oidc client: %w", err)
	}
	return &client, nil
}

// UpdateOIDCClient updates an OIDC client by its internal ID.
func (c *Client) UpdateOIDCClient(ctx context.Context, id string, params OIDCClientParams) (*OIDCClient, error) {
	data, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("marshal params: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut,
		c.baseURL+"/api/oidc/clients/"+url.PathEscape(id), bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	c.setAdminAuth(req)
	req.Header.Set("Content-Type", "application/json")

	var client OIDCClient
	if err := c.do(req, &client); err != nil {
		return nil, fmt.Errorf("update oidc client: %w", err)
	}
	return &client, nil
}

// DeleteOIDCClient permanently deletes an OIDC client.
func (c *Client) DeleteOIDCClient(ctx context.Context, id string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		c.baseURL+"/api/oidc/clients/"+url.PathEscape(id), nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	c.setAdminAuth(req)

	if err := c.do(req, nil); err != nil {
		return fmt.Errorf("delete oidc client: %w", err)
	}
	return nil
}

// RotateOIDCClientSecret generates a new client secret and returns it.
// This is the only way to retrieve the secret — it is never stored by Pocket ID.
func (c *Client) RotateOIDCClientSecret(ctx context.Context, id string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/oidc/clients/"+url.PathEscape(id)+"/secret", http.NoBody)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	c.setAdminAuth(req)

	var result struct {
		Secret string `json:"secret"`
	}
	if err := c.do(req, &result); err != nil {
		return "", fmt.Errorf("rotate oidc client secret: %w", err)
	}
	return result.Secret, nil
}

// SetOIDCClientAllowedGroups replaces the user groups allowed to use an OIDC
// client. Pass an empty slice to remove all group restrictions.
func (c *Client) SetOIDCClientAllowedGroups(ctx context.Context, id string, groupIDs []string) (*OIDCClient, error) {
	if groupIDs == nil {
		groupIDs = []string{}
	}
	body := struct {
		UserGroupIDs []string `json:"userGroupIds"`
	}{UserGroupIDs: groupIDs}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut,
		c.baseURL+"/api/oidc/clients/"+url.PathEscape(id)+"/allowed-user-groups",
		bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	c.setAdminAuth(req)
	req.Header.Set("Content-Type", "application/json")

	var client OIDCClient
	if err := c.do(req, &client); err != nil {
		return nil, fmt.Errorf("set oidc client allowed groups: %w", err)
	}
	return &client, nil
}

// contentTypeExt maps MIME types to file extensions for logo uploads.
var contentTypeExt = map[string]string{
	"image/png":  "logo.png",
	"image/jpeg": "logo.jpg",
	"image/webp": "logo.webp",
}

// SetOIDCClientLogo uploads a logo image for an OIDC client.
// imageData must be PNG, JPEG, or WebP; maxiumum 2 MB as enforced by Pocket ID.
func (c *Client) SetOIDCClientLogo(ctx context.Context, id string, imageData []byte, contentType string) error {
	filename := contentTypeExt[contentType]
	if filename == "" {
		filename = "logo.png"
	}
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("file", filename)
	if err != nil {
		return fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(imageData)); err != nil {
		return fmt.Errorf("write image data: %w", err)
	}
	mw.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/oidc/clients/"+url.PathEscape(id)+"/logo", &buf)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	c.setAdminAuth(req)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	if err := c.do(req, nil); err != nil {
		return fmt.Errorf("upload logo: %w", err)
	}
	return nil
}

// GetOIDCClientLogo fetches the logo image for an OIDC client from the internal
// Pocket ID service and streams it back to the caller via w. Uses the internal
// Docker network URL (not the public OIDC issuer URL) so it never goes through
// Caddy. Returns false (no logo) when Pocket ID returns 404.
func (c *Client) GetOIDCClientLogo(ctx context.Context, id string, w http.ResponseWriter) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.baseURL+"/api/oidc/clients/"+url.PathEscape(id)+"/logo", nil)
	if err != nil {
		return false, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return false, fmt.Errorf("pocket id responded %d: %s", resp.StatusCode, sanitizeBody(body))
	}

	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusOK)
	_, err = io.Copy(w, io.LimitReader(resp.Body, 2<<20)) // 2 MB cap
	return true, err
}

// DeleteUser permanently deletes a user from Pocket ID.
func (c *Client) DeleteUser(ctx context.Context, userID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		c.baseURL+"/api/users/"+url.PathEscape(userID), nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	c.setAdminAuth(req)

	if err := c.do(req, nil); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

// groupMembersPage is the paginated user list returned by the group members API.
type groupMembersPage struct {
	Data []struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
	} `json:"data"`
	Pagination struct {
		TotalItems   int `json:"totalItems"`
		TotalPages   int `json:"totalPages"`
		CurrentPage  int `json:"currentPage"`
		ItemsPerPage int `json:"itemsPerPage"`
	} `json:"pagination"`
}

// ListGroupMembers returns all users who belong to the named group.
// It pages through the full result set automatically.
func (c *Client) ListGroupMembers(ctx context.Context, groupName string) ([]User, error) {
	group, err := c.FindGroupByName(ctx, groupName)
	if err != nil {
		return nil, fmt.Errorf("find group %q: %w", groupName, err)
	}
	if group == nil {
		return nil, nil // group doesn't exist yet — no members
	}

	var users []User
	page := 1
	for {
		u := fmt.Sprintf("%s/api/users?groupId=%s&page=%d&limit=100",
			c.baseURL, url.PathEscape(group.ID), page)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, fmt.Errorf("build request: %w", err)
		}
		c.setAdminAuth(req)

		var result groupMembersPage
		if err := c.do(req, &result); err != nil {
			return nil, fmt.Errorf("list group members (page %d): %w", page, err)
		}

		for _, m := range result.Data {
			users = append(users, User{ID: m.ID, Username: m.Username, Email: m.Email})
		}

		if page >= result.Pagination.TotalPages || result.Pagination.TotalPages == 0 {
			break
		}
		page++
	}
	return users, nil
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

// Ping verifies that Pocket ID is reachable by fetching the OIDC discovery
// document. Any non-5xx response (including 404) is treated as "reachable".
func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.baseURL+"/.well-known/openid-configuration", nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) //nolint:errcheck — draining for connection reuse
	if resp.StatusCode >= 500 {
		return fmt.Errorf("pocket-id returned status %d", resp.StatusCode)
	}
	return nil
}
