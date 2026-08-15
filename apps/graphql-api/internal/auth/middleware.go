package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type contextKey string

const (
	contextKeyUserID   contextKey = "user_id"
	contextKeyUserRole contextKey = "user_role"
)

// Config controls how Supabase JWTs are verified.
type Config struct {
	// Secret is the shared secret for legacy HS256 tokens. Used only when the
	// HS256 path is active (AllowHS256, or no JWKSURL configured).
	Secret string
	// JWKSURL is the Supabase endpoint publishing asymmetric signing keys
	// (<supabase>/auth/v1/.well-known/jwks.json). When set, ES256/RS256
	// tokens are accepted and HS256 is rejected unless AllowHS256 is true.
	JWKSURL string
	// Audience, when non-empty, must match the token's aud claim. Supabase
	// user access tokens carry aud "authenticated"; legacy anon/service-role
	// keys carry no aud, so disable this check if those must be accepted.
	Audience string
	// Issuer, when non-empty, must match the token's iss claim.
	Issuer string
	// AllowHS256 keeps the legacy shared-secret path open alongside JWKS.
	AllowHS256 bool
}

// Verifier validates Supabase JWTs. Symmetric (HS256) tokens are verified with
// the shared secret; asymmetric (ES*/RS*) tokens are verified against the
// project's JWKS endpoint — newer Supabase versions sign user access tokens
// with a rotating asymmetric key while the legacy anon/service keys stay HS256.
type Verifier struct {
	secret     []byte
	jwks       *jwksCache
	allowHS256 bool
	parseOpts  []jwt.ParserOption
	logger     *slog.Logger
}

// NewVerifier creates a Verifier from cfg. When cfg.JWKSURL is empty the
// shared secret is the only verification path and HS256 stays enabled
// regardless of cfg.AllowHS256; an empty secret disables HS256 outright.
func NewVerifier(cfg Config, logger *slog.Logger) *Verifier {
	if logger == nil {
		logger = slog.Default()
	}
	v := &Verifier{secret: []byte(cfg.Secret), logger: logger}
	if cfg.JWKSURL != "" {
		v.jwks = newJWKSCache(cfg.JWKSURL)
	}
	v.allowHS256 = (cfg.AllowHS256 || v.jwks == nil) && cfg.Secret != ""

	var methods []string
	if v.jwks != nil {
		methods = append(methods, "ES256", "RS256")
	}
	if v.allowHS256 {
		methods = append(methods, "HS256")
	}
	v.parseOpts = []jwt.ParserOption{
		jwt.WithValidMethods(methods),
		jwt.WithExpirationRequired(),
	}
	if cfg.Audience != "" {
		v.parseOpts = append(v.parseOpts, jwt.WithAudience(cfg.Audience))
	}
	if cfg.Issuer != "" {
		v.parseOpts = append(v.parseOpts, jwt.WithIssuer(cfg.Issuer))
	}
	return v
}

// Middleware validates Supabase JWTs and injects user_id + role into context.
// Unauthenticated requests pass through — resolvers enforce auth per operation.
func (v *Verifier) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			authHeader := r.Header.Get("Authorization")
			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
				ctx = v.contextWithClaims(ctx, tokenStr)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ContextWithToken parses a bearer token and injects auth claims into context.
// Invalid tokens are ignored and the original context is returned unchanged.
func (v *Verifier) ContextWithToken(ctx context.Context, tokenStr string) context.Context {
	return v.contextWithClaims(ctx, tokenStr)
}

// contextWithClaims returns ctx unchanged when the token does not verify; the
// request then proceeds unauthenticated and resolvers reject it per operation.
// Failures are logged (reason only, never the token) so misconfiguration —
// e.g. an unreachable JWKS endpoint — is visible server-side.
func (v *Verifier) contextWithClaims(ctx context.Context, tokenStr string) context.Context {
	token, err := jwt.Parse(tokenStr, v.keyFor, v.parseOpts...)
	if err != nil || !token.Valid {
		reason := "token not valid"
		if err != nil {
			reason = err.Error()
		}
		v.logger.Warn("jwt verification failed", "reason", reason)
		return ctx
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return ctx
	}

	if sub, ok := claims["sub"].(string); ok {
		if id, err := uuid.Parse(sub); err == nil {
			ctx = context.WithValue(ctx, contextKeyUserID, id)
		}
	}

	if role, ok := claims["role"].(string); ok {
		ctx = context.WithValue(ctx, contextKeyUserRole, role)
	}

	return ctx
}

func (v *Verifier) keyFor(t *jwt.Token) (interface{}, error) {
	switch t.Method.(type) {
	case *jwt.SigningMethodHMAC:
		if !v.allowHS256 {
			return nil, errors.New("HS256 tokens are disabled while JWKS is configured (set AUTH_ALLOW_HS256=true to allow)")
		}
		return v.secret, nil
	case *jwt.SigningMethodECDSA, *jwt.SigningMethodRSA, *jwt.SigningMethodRSAPSS:
		if v.jwks == nil {
			return nil, errors.New("asymmetric token but SUPABASE_JWKS_URL is not configured")
		}
		kid, _ := t.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("token missing kid header")
		}
		return v.jwks.key(kid)
	default:
		return nil, errors.New("unexpected signing method")
	}
}

// UserIDFromCtx extracts the authenticated user's UUID from context.
// Returns an error if the user is not authenticated.
func UserIDFromCtx(ctx context.Context) (uuid.UUID, error) {
	id, ok := ctx.Value(contextKeyUserID).(uuid.UUID)
	if !ok {
		return uuid.UUID{}, errors.New("unauthenticated")
	}
	return id, nil
}

// RoleFromCtx returns the user's role from context ("authenticated" | "anon").
func RoleFromCtx(ctx context.Context) string {
	role, _ := ctx.Value(contextKeyUserRole).(string)
	if role == "" {
		return "anon"
	}
	return role
}
