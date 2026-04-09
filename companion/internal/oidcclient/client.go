package oidcclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/lhw/scid/companion/internal/pocketid"
)

// Client wraps a standard OIDC discovery provider and OAuth2 config.
type Client struct {
	issuerURL       string // URL used for OIDC discovery (may be an internal URL)
	publicIssuerURL string // browser-facing issuer URL used to validate the `iss` JWT claim
	clientID        string
	clientSecret    string       // empty for public clients
	httpClient      *http.Client // nil means use the default; set when insecureTLS=true

	mu           sync.Mutex
	provider     *oidc.Provider
	oauth2Config *oauth2.Config
}

// New creates a lazily initialized OIDC client.
// issuerURL is used for OIDC discovery and token exchange; it may be an
// internal URL (e.g. http://pocket-id:1411) to avoid TLS overhead.
// publicIssuerURL is the browser-facing issuer URL that appears in JWT `iss`
// claims; when it differs from issuerURL, InsecureIssuerURLContext is used so
// go-oidc accepts the mismatch. Pass "" to use issuerURL for both.
// insecureTLS disables TLS certificate verification — dev only (e.g. mkcert).
// Pass a non-empty clientSecret to use a confidential client.
func New(issuerURL, publicIssuerURL, clientID, clientSecret string, insecureTLS bool) *Client {
	if publicIssuerURL == "" {
		publicIssuerURL = issuerURL
	}
	var hc *http.Client
	if insecureTLS {
		hc = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // dev-only, gated by explicit env var
			},
		}
	}
	return &Client{
		issuerURL:       issuerURL,
		publicIssuerURL: publicIssuerURL,
		clientID:        clientID,
		clientSecret:    clientSecret,
		httpClient:      hc,
	}
}

func (c *Client) ensureProvider(ctx context.Context) (*oidc.Provider, *oauth2.Config, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.provider != nil && c.oauth2Config != nil {
		return c.provider, c.oauth2Config, nil
	}

	// When the internal discovery URL differs from the public issuer URL (e.g.
	// in local dev where the companion connects via http://pocket-id:1411 but
	// JWTs carry iss=https://auth-dev.scid.my), tell go-oidc which issuer to
	// expect so it doesn't reject the mismatch.
	discoveryCtx := ctx
	if c.httpClient != nil {
		// Inject the custom HTTP client so both go-oidc and oauth2 use it for
		// ALL requests (discovery, token exchange, userinfo).
		discoveryCtx = oidc.ClientContext(discoveryCtx, c.httpClient)
	}
	if c.publicIssuerURL != c.issuerURL {
		discoveryCtx = oidc.InsecureIssuerURLContext(discoveryCtx, c.publicIssuerURL)
	}

	provider, err := oidc.NewProvider(discoveryCtx, c.issuerURL)
	if err != nil {
		return nil, nil, fmt.Errorf("discover oidc provider: %w", err)
	}

	config := &oauth2.Config{
		ClientID:     c.clientID,
		ClientSecret: c.clientSecret,
		Endpoint:     provider.Endpoint(),
	}

	c.provider = provider
	c.oauth2Config = config

	return provider, config, nil
}

// clientCtx returns ctx enriched with the custom HTTP client when one is
// configured (e.g. insecureTLS mode). oauth2 and go-oidc both read the client
// from the context via the oauth2.HTTPClient key.
func (c *Client) clientCtx(ctx context.Context) context.Context {
	if c.httpClient != nil {
		return oidc.ClientContext(ctx, c.httpClient)
	}
	return ctx
}

// ExchangeAuthorizationCode exchanges an auth code plus PKCE verifier for an access token.
func (c *Client) ExchangeAuthorizationCode(ctx context.Context, redirectURI, code, codeVerifier string) (*oauth2.Token, error) {
	_, config, err := c.ensureProvider(ctx)
	if err != nil {
		return nil, err
	}

	localConfig := *config
	localConfig.RedirectURL = redirectURI

	token, err := localConfig.Exchange(c.clientCtx(ctx), code, oauth2.VerifierOption(codeVerifier))
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

	info, err := provider.UserInfo(c.clientCtx(ctx), oauth2.StaticTokenSource(&oauth2.Token{
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
