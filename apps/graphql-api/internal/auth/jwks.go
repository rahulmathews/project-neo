package auth

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"
)

// jwks mirrors the RFC 7517 key set served by Supabase Auth at
// /auth/v1/.well-known/jwks.json.
type jwks struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// jwksCache fetches and caches asymmetric public keys by kid. An unknown kid
// triggers a refetch (key rotation), rate-limited per outcome: a successful
// fetch opens a long window, a failed one a short retry window, so a single
// bad fetch cannot blackhole verification for the full interval. Fetches run
// outside the lock and are deduplicated across goroutines; a bad or empty
// response never evicts previously cached keys.
type jwksCache struct {
	url    string
	client *http.Client

	mu          sync.Mutex // guards the fields below; never held across I/O
	keys        map[string]crypto.PublicKey
	fetching    bool
	fetchDone   chan struct{}
	lastAttempt time.Time
	lastFailed  bool
}

const (
	jwksRefreshInterval = 30 * time.Second // unknown-kid refetch limit after a successful fetch
	jwksFailureRetry    = 5 * time.Second  // retry window after a failed fetch
	jwksFetchTimeout    = 5 * time.Second
	jwksMaxBody         = 1 << 20
)

func newJWKSCache(url string) *jwksCache {
	return &jwksCache{
		url:    url,
		client: &http.Client{Timeout: jwksFetchTimeout},
		keys:   map[string]crypto.PublicKey{},
	}
}

// key returns the public key for kid, refetching the JWKS at most once per
// call when the kid is unknown and the rate limit allows.
func (c *jwksCache) key(kid string) (crypto.PublicKey, error) {
	for attempt := 0; ; attempt++ {
		c.mu.Lock()
		if k, ok := c.keys[kid]; ok {
			c.mu.Unlock()
			return k, nil
		}
		if attempt > 0 {
			c.mu.Unlock()
			return nil, fmt.Errorf("unknown signing key %q", kid)
		}
		if c.fetching {
			// Another goroutine is already fetching — wait for it, then
			// re-check the cache exactly once.
			done := c.fetchDone
			c.mu.Unlock()
			<-done
			continue
		}
		window := jwksRefreshInterval
		if c.lastFailed {
			window = jwksFailureRetry
		}
		if time.Since(c.lastAttempt) < window {
			c.mu.Unlock()
			return nil, fmt.Errorf("unknown signing key %q", kid)
		}
		c.fetching = true
		c.fetchDone = make(chan struct{})
		c.lastAttempt = time.Now()
		done := c.fetchDone
		c.mu.Unlock()

		keys, err := c.fetch()

		c.mu.Lock()
		if err == nil {
			c.keys = keys
			c.lastFailed = false
		} else {
			c.lastFailed = true
		}
		c.fetching = false
		close(done)
		c.mu.Unlock()

		if err != nil {
			return nil, err
		}
	}
}

// fetch retrieves and parses the JWKS. It returns an error when the response
// is invalid or contains no usable signature keys, so callers never replace a
// good cache with a bad response.
func (c *jwksCache) fetch() (map[string]crypto.PublicKey, error) {
	ctx, cancel := context.WithTimeout(context.Background(), jwksFetchTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return nil, fmt.Errorf("build jwks request: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch jwks: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch jwks: unexpected status %d", resp.StatusCode)
	}

	var set jwks
	if err := json.NewDecoder(io.LimitReader(resp.Body, jwksMaxBody)).Decode(&set); err != nil {
		return nil, fmt.Errorf("decode jwks: %w", err)
	}

	keys := make(map[string]crypto.PublicKey, len(set.Keys))
	for _, k := range set.Keys {
		if k.Kid == "" || (k.Use != "" && k.Use != "sig") {
			continue
		}
		pub, err := k.publicKey()
		if err != nil || pub == nil {
			continue // skip symmetric and unsupported entries
		}
		keys[k.Kid] = pub
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("jwks: no usable signature keys in response")
	}
	return keys, nil
}

// publicKey converts a JWK into a crypto.PublicKey. Key types that are not
// asymmetric verification keys (e.g. "oct") return (nil, nil).
func (k jwk) publicKey() (crypto.PublicKey, error) {
	switch k.Kty {
	case "EC":
		return k.ecPublicKey()
	case "RSA":
		return k.rsaPublicKey()
	default:
		return nil, nil
	}
}

func (k jwk) ecPublicKey() (crypto.PublicKey, error) {
	var curve elliptic.Curve
	switch k.Crv {
	case "P-256":
		curve = elliptic.P256()
	case "P-384":
		curve = elliptic.P384()
	case "P-521":
		curve = elliptic.P521()
	default:
		return nil, fmt.Errorf("unsupported curve %q", k.Crv)
	}
	x, err := base64.RawURLEncoding.DecodeString(k.X)
	if err != nil {
		return nil, fmt.Errorf("decode x: %w", err)
	}
	y, err := base64.RawURLEncoding.DecodeString(k.Y)
	if err != nil {
		return nil, fmt.Errorf("decode y: %w", err)
	}
	return &ecdsa.PublicKey{Curve: curve, X: new(big.Int).SetBytes(x), Y: new(big.Int).SetBytes(y)}, nil
}

func (k jwk) rsaPublicKey() (crypto.PublicKey, error) {
	n, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("decode n: %w", err)
	}
	e, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("decode e: %w", err)
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}, nil
}
