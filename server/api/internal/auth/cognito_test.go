package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestCognitoVerifierAcceptsAccessTokenAndCachesJWKS(t *testing.T) {
	privateKey := testRSAKey(t)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Cache-Control", "public, max-age=3600")
		writeJWKS(t, w, "key-1", &privateKey.PublicKey)
	}))
	defer server.Close()

	verifier, err := NewCognitoVerifier(server.URL, "client-1", server.Client())
	if err != nil {
		t.Fatalf("NewCognitoVerifier: %v", err)
	}
	token := signedToken(t, privateKey, "key-1", jwt.MapClaims{
		"iss":       server.URL,
		"sub":       "user-1",
		"username":  "alice",
		"client_id": "client-1",
		"token_use": "access",
		"iat":       time.Now().Add(-time.Minute).Unix(),
		"exp":       time.Now().Add(time.Hour).Unix(),
	})

	for range 2 {
		principal, err := verifier.Verify(context.Background(), token)
		if err != nil {
			t.Fatalf("Verify: %v", err)
		}
		if principal.Subject != "user-1" || principal.Username != "alice" {
			t.Fatalf("principal = %#v", principal)
		}
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("JWKS requests = %d, want 1", got)
	}
}

func TestCognitoVerifierRejectsInvalidClaims(t *testing.T) {
	privateKey := testRSAKey(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJWKS(t, w, "key-1", &privateKey.PublicKey)
	}))
	defer server.Close()

	verifier, err := NewCognitoVerifier(server.URL, "client-1", server.Client())
	if err != nil {
		t.Fatalf("NewCognitoVerifier: %v", err)
	}
	tests := []struct {
		name   string
		claims jwt.MapClaims
	}{
		{
			name: "ID token",
			claims: jwt.MapClaims{
				"iss": server.URL, "sub": "user-1", "client_id": "client-1",
				"token_use": "id", "iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
			},
		},
		{
			name: "wrong client",
			claims: jwt.MapClaims{
				"iss": server.URL, "sub": "user-1", "client_id": "client-2",
				"token_use": "access", "iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
			},
		},
		{
			name: "expired",
			claims: jwt.MapClaims{
				"iss": server.URL, "sub": "user-1", "client_id": "client-1",
				"token_use": "access", "iat": time.Now().Add(-time.Hour).Unix(), "exp": time.Now().Add(-time.Minute).Unix(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := signedToken(t, privateKey, "key-1", tt.claims)
			if _, err := verifier.Verify(context.Background(), token); err == nil {
				t.Fatal("Verify() error = nil, want invalid token")
			}
		})
	}
}

func testRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	return key
}

func signedToken(t *testing.T, key *rsa.PrivateKey, kid string, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	raw, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return raw
}

func writeJWKS(t *testing.T, w http.ResponseWriter, kid string, key *rsa.PublicKey) {
	t.Helper()
	exponent := big.NewInt(int64(key.E)).Bytes()
	document := map[string]any{
		"keys": []map[string]string{{
			"kid": kid,
			"kty": "RSA",
			"alg": "RS256",
			"use": "sig",
			"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(exponent),
		}},
	}
	if err := json.NewEncoder(w).Encode(document); err != nil {
		t.Fatalf("encode JWKS: %v", err)
	}
}
