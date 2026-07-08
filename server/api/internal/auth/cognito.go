package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	defaultJWKSCacheTTL = time.Hour
	maxJWKSResponseSize = 1 << 20
)

var ErrInvalidToken = errors.New("invalid token")

type Principal struct {
	Subject  string
	Username string
}

type TokenVerifier interface {
	Verify(ctx context.Context, rawToken string) (Principal, error)
}

type CognitoVerifier struct {
	issuer     string
	clientID   string
	jwksURL    string
	httpClient *http.Client

	mu        sync.Mutex
	keys      map[string]*rsa.PublicKey
	expiresAt time.Time
}

func NewCognitoVerifier(issuer, clientID string, httpClient *http.Client) (*CognitoVerifier, error) {
	issuer = strings.TrimRight(issuer, "/")
	if issuer == "" || clientID == "" {
		return nil, errors.New("cognito issuer and app client ID are required")
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &CognitoVerifier{
		issuer:     issuer,
		clientID:   clientID,
		jwksURL:    issuer + "/.well-known/jwks.json",
		httpClient: httpClient,
	}, nil
}

func (v *CognitoVerifier) Verify(ctx context.Context, rawToken string) (Principal, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(
		rawToken,
		claims,
		func(token *jwt.Token) (any, error) {
			kid, ok := token.Header["kid"].(string)
			if !ok || kid == "" {
				return nil, ErrInvalidToken
			}
			return v.key(ctx, kid)
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithIssuer(v.issuer),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(30*time.Second),
	)
	if err != nil || !token.Valid {
		return Principal{}, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if stringClaim(claims, "token_use") != "access" ||
		stringClaim(claims, "client_id") != v.clientID {
		return Principal{}, ErrInvalidToken
	}

	subject := stringClaim(claims, "sub")
	if subject == "" {
		return Principal{}, ErrInvalidToken
	}
	return Principal{
		Subject:  subject,
		Username: stringClaim(claims, "username"),
	}, nil
}

func (v *CognitoVerifier) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if time.Now().Before(v.expiresAt) {
		if key := v.keys[kid]; key != nil {
			return key, nil
		}
	}
	if err := v.refreshKeys(ctx); err != nil {
		return nil, err
	}
	if key := v.keys[kid]; key != nil {
		return key, nil
	}
	return nil, ErrInvalidToken
}

func (v *CognitoVerifier) refreshKeys(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return fmt.Errorf("create JWKS request: %w", err)
	}
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch JWKS: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch JWKS: unexpected status %d", resp.StatusCode)
	}

	var document struct {
		Keys []struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
			Alg string `json:"alg"`
			Use string `json:"use"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	body := io.LimitReader(resp.Body, maxJWKSResponseSize)
	if err := json.NewDecoder(body).Decode(&document); err != nil {
		return fmt.Errorf("decode JWKS: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(document.Keys))
	for _, jwk := range document.Keys {
		if jwk.Kid == "" || jwk.Kty != "RSA" || jwk.Alg != "RS256" || jwk.Use != "sig" {
			continue
		}
		key, err := rsaPublicKey(jwk.N, jwk.E)
		if err != nil {
			continue
		}
		keys[jwk.Kid] = key
	}
	if len(keys) == 0 {
		return errors.New("JWKS contains no supported signing keys")
	}

	v.keys = keys
	v.expiresAt = time.Now().Add(cacheTTL(resp.Header.Get("Cache-Control")))
	return nil
}

func rsaPublicKey(modulus, exponent string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(modulus)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(exponent)
	if err != nil || len(eBytes) == 0 || len(eBytes) > 4 {
		return nil, errors.New("invalid RSA exponent")
	}

	var padded [4]byte
	copy(padded[4-len(eBytes):], eBytes)
	e := binary.BigEndian.Uint32(padded[:])
	if e < 2 {
		return nil, errors.New("invalid RSA exponent")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: int(e)}, nil
}

func cacheTTL(cacheControl string) time.Duration {
	for _, directive := range strings.Split(cacheControl, ",") {
		name, value, ok := strings.Cut(strings.TrimSpace(directive), "=")
		if !ok || strings.ToLower(name) != "max-age" {
			continue
		}
		seconds, err := strconv.Atoi(strings.Trim(value, `"`))
		if err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	return defaultJWKSCacheTTL
}

func stringClaim(claims jwt.MapClaims, name string) string {
	value, _ := claims[name].(string)
	return value
}
