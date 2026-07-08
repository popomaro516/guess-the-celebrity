package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type verifierFunc func(context.Context, string) (Principal, error)

func (f verifierFunc) Verify(ctx context.Context, token string) (Principal, error) {
	return f(ctx, token)
}

func TestRequire(t *testing.T) {
	gin.SetMode(gin.TestMode)
	verifier := verifierFunc(func(_ context.Context, token string) (Principal, error) {
		if token != "valid-token" {
			return Principal{}, errors.New("invalid")
		}
		return Principal{Subject: "user-1"}, nil
	})

	tests := []struct {
		name       string
		header     string
		wantStatus int
	}{
		{name: "valid", header: "Bearer valid-token", wantStatus: http.StatusNoContent},
		{name: "missing", wantStatus: http.StatusUnauthorized},
		{name: "wrong scheme", header: "Basic valid-token", wantStatus: http.StatusUnauthorized},
		{name: "invalid token", header: "Bearer invalid-token", wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/", Require(verifier), func(c *gin.Context) {
				if _, ok := PrincipalFromContext(c); !ok {
					t.Error("PrincipalFromContext() did not return a principal")
				}
				c.Status(http.StatusNoContent)
			})
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", tt.header)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusUnauthorized &&
				rec.Header().Get("WWW-Authenticate") != "Bearer" {
				t.Fatal("unauthorized response is missing WWW-Authenticate")
			}
		})
	}
}
