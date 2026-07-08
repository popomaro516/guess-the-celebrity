package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const principalKey = "auth.principal"

func Require(verifier TokenVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		scheme, token, ok := strings.Cut(c.GetHeader("Authorization"), " ")
		if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
			unauthorized(c)
			return
		}

		principal, err := verifier.Verify(c.Request.Context(), strings.TrimSpace(token))
		if err != nil {
			unauthorized(c)
			return
		}
		c.Set(principalKey, principal)
		c.Next()
	}
}

func Disabled() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

func PrincipalFromContext(c *gin.Context) (Principal, bool) {
	value, exists := c.Get(principalKey)
	if !exists {
		return Principal{}, false
	}
	principal, ok := value.(Principal)
	return principal, ok
}

func unauthorized(c *gin.Context) {
	c.Header("WWW-Authenticate", "Bearer")
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
}
