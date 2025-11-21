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
		secret = "family-drive-super-secret-key-change-in-production" // 开发时使用，生产请设 env
	}
	jwtSecret = []byte(secret)
}

// UserClaims 包含完整的用户信息
type UserClaims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	jwt.RegisteredClaims
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

// GenerateUserToken 生成包含完整用户信息的 JWT Token
func GenerateUserToken(userID int, username, email string, d time.Duration) (string, error) {
	claims := &UserClaims{
		UserID:   userID,
		Username: username,
		Email:    email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(d)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "family-drive",
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ParseUserToken 解析Token并返回完整的用户信息
func ParseUserToken(tok string) (*UserClaims, error) {
	claims := &UserClaims{}
	
	token, err := jwt.ParseWithClaims(tok, claims, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	
	if err != nil || !token.Valid {
		return nil, err
	}
	
	return claims, nil
}

// ValidateToken 验证Token有效性
func ValidateToken(tok string) bool {
	_, err := ParseUserToken(tok)
	return err == nil
}