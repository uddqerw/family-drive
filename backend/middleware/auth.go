package middleware

import (
	"net/http"
	"strings"
	"familydrive/internal/auth"
)

// CORS 中间件处理跨域请求（用于不需要认证的路由）
func CORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 设置 CORS 头
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "86400") // 24小时

		// 处理预检请求
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 调用下一个处理器
		next.ServeHTTP(w, r)
	}
}

// AuthMiddleware 验证JWT Token的中间件（已经包含CORS）
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 设置CORS头
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// 处理预检请求
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 从请求头获取Token
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"success": false, "message": "缺少认证Token"}`, http.StatusUnauthorized)
			return
		}

		// 检查Token格式：Bearer <token>
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, `{"success": false, "message": "Token格式错误"}`, http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]

		// 验证Token
		claims, err := auth.ParseUserToken(tokenString)
		if err != nil {
			http.Error(w, `{"success": false, "message": "无效的Token: `+err.Error()+`"}`, http.StatusUnauthorized)
			return
		}

		// 将用户信息添加到请求头，供后续处理器使用
		r.Header.Set("X-User-ID", string(claims.UserID))
		r.Header.Set("X-Username", claims.Username)
		r.Header.Set("X-User-Email", claims.Email)

		// 调用下一个处理器
		next.ServeHTTP(w, r)
	}
}