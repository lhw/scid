package oidcclient

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/lhw/scid/companion/internal/pocketid"
)

// Client wraps a standard OIDC discovery provider and OAuth2 config.
type Client struct {
	issuerURL string
	clientID  string

	mu           sync.Mutex
	provider     *oidc.Provider
	oauth2Config *oauth2.Config
}

// New creates a lazily initialized OIDC client.
func New(issuerURL, clientID string) *Client {
	return &Client{issuerURL: issuerURL, clientID: clientID}
}

func (c *Client) ensureProvider(ctx context.Context) (*oidc.Provider, *oauth2.Config, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.provider != nil && c.oauth2Config != nil {
		return c.provider, c.oauth2Config, nil
	}

	provider, err := oidc.NewProvider(ctx, c.issuerURL)
	if err != nil {
		return nil, nil, fmt.Errorf("discover oidc provider: %w", err)
	}

	config := &oauth2.Config{
		ClientID: c.clientID,
		Endpoint: provider.Endpoint(),
	}

	c.provider = provider
	c.oauth2Config = config

	return provider, config, nil
}

// ExchangeAuthorizationCode exchanges an auth code plus PKCE verifier for an access token.
func (c *Client) ExchangeAuthorizationCode(ctx context.Context, redirectURI, code, codeVerifier string) (*oauth2.Token, error) {
	_, config, err := c.ensureProvider(ctx)
	if err != nil {
		return nil, err
	}

	localConfig := *config
	localConfig.RedirectURL = redirectURI

	token, err := localConfig.Exchange(ctx, code, oauth2.VerifierOption(codeVerifier))
	if err != nil {
		return nil, fmt.Errorf("exchange auth code: %w", err)
	}
	if token.Expiry.IsZero() && token.ExpiresIn > 0 {
		token.Expiry = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	}
	return token, nil
}

// GetCurrentUser resolves the current user from the OIDC userinfo endpoint.
func (c *Client) GetCurrentUser(ctx context.Context, accessToken string) (*pocketid.User, error) {
	provider, _, err := c.ensureProvider(ctx)
	if err != nil {
		return nil, err
	}

	info, err := provider.UserInfo(ctx, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: accessToken,
		TokenType:   "Bearer",
	}))
	if err != nil {
		return nil, fmt.Errorf("get userinfo: %w", err)
	}

	var claims struct {
		PreferredUsername string `json:"preferred_username"`
		Name              string `json:"name"`
	}
	if err := info.Claims(&claims); err != nil {
		return nil, fmt.Errorf("decode userinfo claims: %w", err)
	}

	username := claims.PreferredUsername
	if username == "" {
		username = claims.Name
	}

	return &pocketid.User{
		ID:       info.Subject,
		Username: username,
		Email:    info.Email,
	}, nil
}

// Ping verifies that the issuer discovery document is reachable.
func (c *Client) Ping(ctx context.Context) error {
	_, _, err := c.ensureProvider(ctx)
	return err
}
