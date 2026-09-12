package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"

	"github.com/list-pandora/isi-stasiun-backend/internal/response"
)

const (
	CtxUserIDKey = "user_id"
	CtxRoleKey   = "user_role"
)

// RequireAuth validates a Bearer JWT access token and injects claims into ctx.Locals.
func RequireAuth(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			return response.Unauthorized(c, "missing bearer token")
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, fiber.ErrUnauthorized
			}
			return []byte(secret), nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
		if err != nil || !token.Valid {
			return response.Unauthorized(c, "invalid or expired token")
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return response.Unauthorized(c, "invalid token claims")
		}
		if claims["typ"] != "access" {
			return response.Unauthorized(c, "access token required")
		}
		userID, userOK := claims["sub"].(string)
		role, roleOK := claims["role"].(string)
		if !userOK || userID == "" || !roleOK || role == "" {
			return response.Unauthorized(c, "invalid token claims")
		}

		c.Locals(CtxUserIDKey, userID)
		c.Locals(CtxRoleKey, role)
		return c.Next()
	}
}

// RequireRole gates a route to specific roles (e.g. "operator", "admin").
// Must run after RequireAuth.
func RequireRole(roles ...string) fiber.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(c *fiber.Ctx) error {
		role, _ := c.Locals(CtxRoleKey).(string)
		if !allowed[role] {
			return response.Forbidden(c, "insufficient role")
		}
		return c.Next()
	}
}
