package middleware

import (
	"errors"
	coreenum "logisfy/core/enum"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/golang-jwt/jwt/v5"
)

func AuthAdmin(secretKey string) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		// get and validate token
		token, _, err := extractAndValidateToken(ctx, secretKey)
		if err != nil {
			return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": err.Error(),
			})
		}

		// validate payload token
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			if claims["role"].(string) == string(coreenum.CTXEnumRoleAdmin) {
				ctx.Locals(string(coreenum.CTXEnumIDUserID), claims["user_id"].(string))
				ctx.Locals(string(coreenum.CTXEnumIDUserEmail), claims["email"].(string))
				ctx.Locals(string(coreenum.CTXEnumIDUserName), claims["name"].(string))
				ctx.Locals(string(coreenum.CTXEnumRoleUser), claims["role"].(string))
				return ctx.Next()
			}
			return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "Unauthorized: not admin",
			})
		}

		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Invalid token",
		})
	}
}

func AuthUser(secretKey string) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		// get and validate token
		token, _, err := extractAndValidateToken(ctx, secretKey)
		if err != nil {
			return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": err.Error(),
			})
		}

		// validate payload token
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			if claims["role"].(string) == string(coreenum.CTXEnumRoleUser) || claims["role"].(string) == string(coreenum.CTXEnumRoleAdmin) {
				ctx.Locals(string(coreenum.CTXEnumIDUserID), claims["user_id"].(string))
				ctx.Locals(string(coreenum.CTXEnumIDUserEmail), claims["email"].(string))
				ctx.Locals(string(coreenum.CTXEnumIDUserName), claims["name"].(string))
				ctx.Locals(string(coreenum.CTXEnumRoleUser), claims["role"].(string))
				return ctx.Next()
			}
			return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "Unauthorized: not user or admin",
			})
		}

		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Invalid token",
		})
	}
}

func extractAndValidateToken(ctx *fiber.Ctx, secretKey string) (*jwt.Token, *string, error) {

	// get header
	authHeader := ctx.Get("Authorization")

	// check header
	if authHeader == "" {
		return nil, nil, errors.New("unauthorized: authorization header is missing")
	}

	// check contains bearer
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, nil, errors.New("unauthorized: invalid token format")
	}

	// get token without prefix "Bearer "
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	// validate token
	token, err := validateToken(tokenString, secretKey)
	if err != nil {
		return nil, nil, err
	}

	return token, &tokenString, nil
}

func validateToken(authHeader, secretKey string) (*jwt.Token, error) {
	return jwt.Parse(authHeader, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}

		return []byte(secretKey), nil
	})
}

func RateLimiterResetPassword() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        3,
		Expiration: 15 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"message": "Terlalu banyak permintaan reset password. Silakan coba lagi setelah 15 menit.",
			})
		},
	})
}
