package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
        "log"

	"familydrive/internal/auth"
	"golang.org/x/crypto/bcrypt"
)

// 登录请求结构
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// 注册请求结构
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// 登录响应结构
type LoginResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Token string `json:"access_token"`
		User  struct {
			ID       int    `json:"id"`
			Username string `json:"username"`
			Email    string `json:"email"`
		} `json:"user"`
	} `json:"data"`
}

// 注册响应结构
type RegisterResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
	} `json:"data"`
}

// 密码加密
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// 密码验证
func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// 登录处理
func HandleLogin(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔐 处理登录请求")
	
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "POST" {
		http.Error(w, "只支持POST请求", http.StatusMethodNotAllowed)
		return
	}
	
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "无效的请求数据", http.StatusBadRequest)
		return
	}

	// 从数据库查询用户
	var userID int
	var username, email, passwordHash string
	err := db.QueryRow(
		"SELECT id, username, email, password_hash FROM users WHERE email = ?", 
		req.Email,
	).Scan(&userID, &username, &email, &passwordHash)
	
	if err == sql.ErrNoRows {
		log.Printf("❌ 登录失败: 邮箱未注册 - %s", req.Email)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "邮箱未注册",
		})
		return
	} else if err != nil {
		log.Printf("❌ 数据库查询错误: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "服务器内部错误",
		})
		return
	}
	
	// 验证密码
	if !checkPasswordHash(req.Password, passwordHash) {
		log.Printf("❌ 登录失败: 密码错误 - %s", req.Email)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "密码错误",
		})
		return
	}
	
	// 生成包含用户信息的JWT Token (24小时有效期)
	token, err := auth.GenerateUserToken(userID, username, email, 24*time.Hour)
	if err != nil {
		log.Printf("❌ 生成Token失败: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "生成Token失败",
		})
		return
	}
	
	// 登录成功
	log.Printf("✅ 用户登录成功: %s (%s)", username, email)
	
	// 构建响应
	response := LoginResponse{
		Success: true,
		Message: "登录成功",
	}
	response.Data.Token = token
	response.Data.User.ID = userID
	response.Data.User.Username = username
	response.Data.User.Email = email
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// 注册处理
func HandleRegister(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔐 处理注册请求")
	
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "POST" {
		http.Error(w, "只支持POST请求", http.StatusMethodNotAllowed)
		return
	}
	
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "无效的请求数据", http.StatusBadRequest)
		return
	}

	// 验证输入
	if req.Username == "" || req.Email == "" || req.Password == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "用户名、邮箱和密码不能为空",
		})
		return
	}

	if len(req.Password) < 6 {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "密码至少需要6位",
		})
		return
	}

	// 检查邮箱是否已存在
	var existingEmail string
	err := db.QueryRow("SELECT email FROM users WHERE email = ?", req.Email).Scan(&existingEmail)
	if err != sql.ErrNoRows {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "邮箱已被注册",
		})
		return
	}

	// 检查用户名是否已存在
	var existingUsername string
	err = db.QueryRow("SELECT username FROM users WHERE username = ?", req.Username).Scan(&existingUsername)
	if err != sql.ErrNoRows {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "用户名已被使用",
		})
		return
	}
	
	// 密码加密
	passwordHash, err := hashPassword(req.Password)
	if err != nil {
		log.Printf("❌ 密码加密失败: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "服务器内部错误",
		})
		return
	}
	
	// 插入新用户
	result, err := db.Exec(
		"INSERT INTO users (username, email, password_hash) VALUES (?, ?, ?)",
		req.Username, req.Email, passwordHash,
	)
	if err != nil {
		log.Printf("❌ 用户注册失败: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "注册失败，请重试",
		})
		return
	}
	
	userID, _ := result.LastInsertId()
	
	log.Printf("✅ 新用户注册: %s (%s)", req.Username, req.Email)
	
	// 构建响应
	response := RegisterResponse{
		Success: true,
		Message: "注册成功",
	}
	response.Data.ID = userID
	response.Data.Username = req.Username
	response.Data.Email = req.Email
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// 获取当前用户信息
func HandleGetCurrentUser(w http.ResponseWriter, r *http.Request) {
	log.Printf("👤 获取当前用户信息")
	
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	// 从请求头获取用户信息（由中间件设置）
	userID := r.Header.Get("X-User-ID")
	username := r.Header.Get("X-Username")
	email := r.Header.Get("X-User-Email")
	
	if userID == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "用户未登录",
		})
		return
	}
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"id":       userID,
			"username": username,
			"email":    email,
		},
	})
}

// 全局数据库连接变量
var db *sql.DB

// 设置数据库连接
func SetDB(database *sql.DB) {
	db = database
}