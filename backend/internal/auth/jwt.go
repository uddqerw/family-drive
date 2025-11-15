package auth

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret []byte

func init() {
	secret := os.Getenv("FAMILYDRIVE_JWT_SECRET")
	if secret == "" {
		secret = "change-me-to-strong-secret" // 开发时使用，生产请设 env
	}
	jwtSecret = []byte(secret)
}

// GenerateAccessToken generates JWT access token valid for duration d.
func GenerateAccessToken(userID int64, d time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(d).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ParseAccessToken validates token and returns subject (user id).
func ParseAccessToken(tok string) (int64, error) {
	token, err := jwt.Parse(tok, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return 0, err
	}
	if m, ok := token.Claims.(jwt.MapClaims); ok {
		if subf, ok := m["sub"].(float64); ok {
			return int64(subf), nil
		}
	}
	return 0, nil
}
