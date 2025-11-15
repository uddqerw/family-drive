package handlers

import (
	"encoding/json"
	"errors"
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