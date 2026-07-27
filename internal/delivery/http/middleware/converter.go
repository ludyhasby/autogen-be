package middleware

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateJWT(authJWT AuthJWT, secretKeyStr string, rememberMe *bool) (tokenString string, err error) {
	secretKey := []byte(secretKeyStr)
	exp := time.Now().Add(24 * time.Hour).Unix()
	if rememberMe != nil && *rememberMe {
		exp = time.Now().Add(72 * time.Hour).Unix()
	}

	// token claims
	claims := jwt.MapClaims{
		"user_id": authJWT.UserID,
		"email":   authJWT.Email,
		"name":    authJWT.Name,
		"role":    authJWT.Role,
		"iat":     time.Now().Unix(),
		"exp":     exp,
	}
	// create a new JWT token
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// sign the token with secret key
	tokenString, err = jwtToken.SignedString(secretKey)
	if err != nil {
		return "", err
	}
	return
}
