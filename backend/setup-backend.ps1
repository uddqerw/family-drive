# setup-backend.ps1
# 在 backend 目录生成后端源码文件，设置 GOPROXY，尝试 go mod tidy 与构建可执行文件
# 使用方法（在 PowerShell 中）:
# cd C:\dev\family-drive\backend
# Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass -Force
# .\setup-backend.ps1

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$root = (Get-Location).Path
Write-Host "当前路径: $root"

if ((Split-Path $root -Leaf) -ne "backend") {
    Write-Host "警告：建议在 backend 目录中运行此脚本（例如 C:\dev\family-drive\backend）" -ForegroundColor Yellow
}

function Write-File($path, $content) {
    $dir = Split-Path $path -Parent
    if (!(Test-Path $dir)) {
        New-Item -ItemType Directory -Path $dir -Force | Out-Null
    }
    $content | Out-File -FilePath $path -Encoding utf8 -Force
    Write-Host "写入 $path"
}

# go.mod
Write-File "$root\go.mod" @'
module familydrive

go 1.20

require (
    github.com/golang-jwt/jwt/v5 v5.0.0
    modernc.org/sqlite v1.19.0
)
'@

# cmd/server/main.go
Write-File "$root\cmd\server\main.go" @'
package main

import (
	"log"
	"net/http"
	"os"

	"familydrive/internal/db"
	"familydrive/internal/handlers"
)

func main() {
	// DB file path (current dir)
	dbPath := "./family.db"
	if e := os.Getenv("FAMILYDRIVE_DB"); e != "" {
		dbPath = e
	}

	if err := db.Init(dbPath); err != nil {
		log.Fatalf("db.Init: %v", err)
	}
	defer db.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/register", handlers.HandleRegister)
	mux.HandleFunc("/api/auth/login", handlers.HandleLogin)
	mux.HandleFunc("/api/auth/refresh", handlers.HandleRefresh)
	mux.HandleFunc("/api/auth/me", handlers.HandleMe)
	mux.HandleFunc("/api/auth/logout", handlers.HandleLogout)

	addr := ":8000"
	if v := os.Getenv("FAMILYDRIVE_ADDR"); v != "" {
		addr = v
	}
	log.Printf("server starting at %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("ListenAndServe: %v", err)
	}
}
'@

# internal/db/db.go
Write-File "$root\internal\db\db.go" @'
package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

var conn *sql.DB

func Init(path string) error {
	// sqlite DSN: file:path?_foreign_keys=1
	dsn := fmt.Sprintf("file:%s?_foreign_keys=1", path)
	var err error
	conn, err = sql.Open("sqlite", dsn)
	if err != nil {
		return err
	}
	// set sensible limits
	conn.SetMaxOpenConns(1)
	// migrate: create tables
	return migrate()
}

func Close() {
	if conn != nil {
		_ = conn.Close()
	}
}

func DB() *sql.DB {
	return conn
}

func migrate() error {
	s := `
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    token TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL,
    expires_at DATETIME NOT NULL,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);`
	_, err := conn.Exec(s)
	return err
}
'@

# internal/models/user.go
Write-File "$root\internal\models\user.go" @'
package models

import (
	"database/sql"
	"time"

	"familydrive/internal/db"
)

type User struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

func CreateUser(name, email, passwordHash string) (*User, error) {
	res, err := db.DB().Exec("INSERT INTO users(name,email,password_hash) VALUES(?,?,?)", name, email, passwordHash)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return GetUserByID(id)
}

func GetUserByEmail(email string) (*User, error) {
	row := db.DB().QueryRow("SELECT id,name,email,password_hash,created_at FROM users WHERE email = ?", email)
	var u User
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func GetUserByID(id int64) (*User, error) {
	row := db.DB().QueryRow("SELECT id,name,email,password_hash,created_at FROM users WHERE id = ?", id)
	var u User
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func StoreRefreshToken(token string, userID int64, expiresAt time.Time) error {
	_, err := db.DB().Exec("INSERT INTO refresh_tokens(token,user_id,expires_at) VALUES(?,?,?)", token, userID, expiresAt)
	return err
}

func GetRefreshTokenOwner(token string) (int64, time.Time, error) {
	row := db.DB().QueryRow("SELECT user_id,expires_at FROM refresh_tokens WHERE token = ?", token)
	var uid int64
	var expStr string
	err := row.Scan(&uid, &expStr)
	if err == sql.ErrNoRows {
		return 0, time.Time{}, sql.ErrNoRows
	}
	if err != nil {
		return 0, time.Time{}, err
	}
	exp, err := time.Parse(time.RFC3339, expStr)
	if err != nil {
		return 0, time.Time{}, err
	}
	return uid, exp, nil
}

func RevokeRefreshToken(token string) error {
	_, err := db.DB().Exec("DELETE FROM refresh_tokens WHERE token = ?", token)
	return err
}
'@

# internal/auth/jwt.go
Write-File "$root\internal\auth\jwt.go" @'
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
'@

# internal/handlers/auth.go
Write-File "$root\internal\handlers\auth.go" @'
package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"familydrive/internal/auth"
	"familydrive/internal/models"

	"golang.org/x/crypto/bcrypt"

	"github.com/google/uuid"
)

type registerReq struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResp struct {
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var rq registerReq
	if err := json.NewDecoder(r.Body).Decode(&rq); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	rq.Email = strings.TrimSpace(strings.ToLower(rq.Email))
	if rq.Email == "" || rq.Password == "" || rq.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name/email/password required"})
		return
	}
	// check exists
	if u, _ := models.GetUserByEmail(rq.Email); u != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "email already registered"})
		return
	}
	// hash
	hash, err := bcrypt.GenerateFromPassword([]byte(rq.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "hash failed"})
		return
	}
	u, err := models.CreateUser(rq.Name, rq.Email, string(hash))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "create user failed"})
		return
	}
	// auto login: generate tokens
	access, _ := auth.GenerateAccessToken(u.ID, 15*time.Minute)
	refresh := uuid.NewString()
	exp := time.Now().Add(7 * 24 * time.Hour)
	if err := models.StoreRefreshToken(refresh, u.ID, exp); err != nil {
		log.Println("store refresh:", err)
	}
	resp := authResp{AccessToken: access, RefreshToken: refresh, TokenType: "bearer"}
	writeJSON(w, http.StatusCreated, resp)
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var rq loginReq
	if err := json.NewDecoder(r.Body).Decode(&rq); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	rq.Email = strings.TrimSpace(strings.ToLower(rq.Email))
	u, err := models.GetUserByEmail(rq.Email)
	if err != nil || u == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(rq.Password)); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}
	access, _ := auth.GenerateAccessToken(u.ID, 15*time.Minute)
	refresh := uuid.NewString()
	exp := time.Now().Add(7 * 24 * time.Hour)
	if err := models.StoreRefreshToken(refresh, u.ID, exp); err != nil {
		log.Println("store refresh:", err)
	}
	resp := authResp{AccessToken: access, RefreshToken: refresh, TokenType: "bearer"}
	writeJSON(w, http.StatusOK, resp)
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

func HandleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var rq refreshReq
	if err := json.NewDecoder(r.Body).Decode(&rq); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if rq.RefreshToken == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "refresh_token required"})
		return
	}
	uid, exp, err := models.GetRefreshTokenOwner(rq.RefreshToken)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid refresh token"})
		return
	}
	if time.Now().After(exp) {
		_ = models.RevokeRefreshToken(rq.RefreshToken)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "refresh token expired"})
		return
	}
	access, err := auth.GenerateAccessToken(uid, 15*time.Minute)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "generate token failed"})
		return
	}
	writeJSON(w, http.StatusOK, authResp{AccessToken: access, TokenType: "bearer"})
}

func getAuthUserID(r *http.Request) (int64, error) {
	h := r.Header.Get("Authorization")
	if h == "" {
		return 0, errors.New("no auth")
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 {
		return 0, errors.New("invalid auth header")
	}
	token := parts[1]
	uid, err := auth.ParseAccessToken(token)
	if err != nil || uid == 0 {
		return 0, errors.New("invalid token")
	}
	return uid, nil
}

func HandleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	uid, err := getAuthUserID(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	u, err := models.GetUserByID(uid)
	if err != nil || u == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "user not found"})
		return
	}
	// return basic user info
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":    u.ID,
		"name":  u.Name,
		"email": u.Email,
	})
}

type logoutReq struct {
	RefreshToken string `json:"refresh_token"`
}

func HandleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var rq logoutReq
	if err := json.NewDecoder(r.Body).Decode(&rq); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if rq.RefreshToken == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "refresh_token required"})
		return
	}
	if err := models.RevokeRefreshToken(rq.RefreshToken); err != nil {
		// ignore error but log
		log.Println("revoke refresh token:", err)
	}
	writeJSON(w, http.StatusNoContent, nil)
}
'@

# config.example.env
Write-File "$root\config.example.env" @'
# 示例环境变量
FAMILYDRIVE_JWT_SECRET=replace-with-strong-secret
FAMILYDRIVE_DB=./family.db
FAMILYDRIVE_ADDR=:8000
'@

# 设置 GOPROXY（仅当前会话）
$env:GOPROXY = "https://goproxy.cn,direct"
Write-Host "已为当前 PowerShell 会话设置 GOPROXY=$env:GOPROXY"

# 运行 go mod tidy & go build（捕获输出）
Write-Host "正在执行: go mod tidy"
try {
    & go mod tidy 2>&1 | Write-Host
} catch {
    Write-Host "go mod tidy 报错（网络或其他问题）。错误信息如下：" -ForegroundColor Red
    Write-Host $_
}

Write-Host "正在尝试构建可执行文件: go build -o family-drive-server.exe ./cmd/server"
try {
    & go build -o family-drive-server.exe ./cmd/server 2>&1 | Write-Host
    if (Test-Path "$root\family-drive-server.exe") {
        Write-Host "构建成功：$root\family-drive-server.exe" -ForegroundColor Green
    } else {
        Write-Host "构建失败，请检查上面的错误信息。" -ForegroundColor Red
    }
} catch {
    Write-Host "go build 报错：" -ForegroundColor Red
    Write-Host $_
}

Write-Host ""
Write-Host "完成。若出现依赖下载失败，请把错误信息复制发给我，我来帮你继续排查。"
Write-Host "要运行服务器（开发模式），可以执行： go run ./cmd/server"
Write-Host "或运行可执行： .\\family-drive-server.exe"