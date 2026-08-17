package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Token types. An access token and a refresh token are otherwise structurally
// identical, so without an explicit type claim a refresh token is accepted as a
// bearer credential on every protected route — which would make the short access
// expiry meaningless — and an access token can be replayed at /auth/refresh to mint
// fresh long-lived credentials. Every token carries exactly one of these values and
// each consumer accepts exactly one.
const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// Claims represents the JWT token claims.
type Claims struct {
	EmployeeID   string  `json:"employee_id"`
	Email        string  `json:"email"`
	Role         string  `json:"role"`
	DepartmentID *string `json:"department_id,omitempty"`
	// TokenType is required. Tokens issued before this claim existed have no value
	// here and are rejected, which forces a one-time re-login for every session.
	// There is deliberately no compatibility fallback: accepting an untyped token
	// would keep the original vulnerability open for the length of the grace period.
	TokenType string `json:"typ"`
	jwt.RegisteredClaims
}

// ParseToken validates a signed token and asserts that it is of the expected type.
// The signing method is pinned to HMAC to prevent algorithm-confusion attacks, and
// the issuer is verified so tokens minted by another system sharing the secret are
// not accepted.
func ParseToken(tokenStr, secret, issuer, expectedType string) (*Claims, error) {
	claims := &Claims{}

	parserOpts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
	}
	if issuer != "" {
		parserOpts = append(parserOpts, jwt.WithIssuer(issuer))
	}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	}, parserOpts...)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("token is not valid")
	}

	if claims.EmployeeID == "" {
		return nil, fmt.Errorf("token missing employee_id claim")
	}
	if claims.Role == "" {
		return nil, fmt.Errorf("token missing role claim")
	}
	if claims.TokenType == "" {
		return nil, fmt.Errorf("token missing type claim; please sign in again")
	}
	if claims.TokenType != expectedType {
		return nil, fmt.Errorf("expected a %s token", expectedType)
	}

	return claims, nil
}

// bearerToken extracts a token from the Authorization header. It returns an empty
// string when the header is absent or not a well-formed Bearer credential.
func bearerToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// SetAuthContext stores validated claims for downstream handlers.
func SetAuthContext(c *gin.Context, claims *Claims) {
	c.Set("employee_id", claims.EmployeeID)
	c.Set("email", claims.Email)
	c.Set("role", claims.Role)
	if claims.DepartmentID != nil {
		c.Set("department_id", *claims.DepartmentID)
	}
	c.Set("claims", claims)
}

// JWTAuth returns middleware that validates access tokens. Only tokens of type
// "access" are accepted; a refresh token presented here is rejected.
//
// Credentials are read exclusively from the Authorization header. A ?token=
// query fallback used to apply to every protected route, which meant any request
// could put a bearer token somewhere that reverse proxies log, browsers keep in
// history, and pages leak through Referer. The WebSocket, the one endpoint that
// genuinely cannot set a header, now uses a separate short-lived single-use
// ticket instead (see WSTicketAuth).
func JWTAuth(secret, issuer string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := bearerToken(c)

		if tokenStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "authorization header is required",
			})
			return
		}

		claims, err := ParseToken(tokenStr, secret, issuer, TokenTypeAccess)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "invalid or expired token: " + err.Error(),
			})
			return
		}

		SetAuthContext(c, claims)
		c.Next()
	}
}

// RequireRole returns middleware that checks if the authenticated user has one of the allowed roles.
func RequireRole(roles ...string) gin.HandlerFunc {
	if len(roles) == 0 {
		panic("middleware: RequireRole called with no roles")
	}

	roleSet := make(map[string]bool, len(roles))
	for _, r := range roles {
		roleSet[r] = true
	}

	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "authentication required",
			})
			return
		}

		roleStr, ok := role.(string)
		if !ok || !roleSet[roleStr] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   fmt.Sprintf("insufficient permissions: role '%s' is not authorized", roleStr),
			})
			return
		}

		c.Next()
	}
}
