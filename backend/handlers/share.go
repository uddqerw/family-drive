package handlers

import (
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"familydrive/models"
)

// var db *sql.DB

// SetShareDB 设置数据库连接
// func SetShareDB(database *sql.DB) {
// 	db = database
// }

// GenerateShareLink 生成文件分享链接
func GenerateShareLink(w http.ResponseWriter, r *http.Request) {
	// 从URL路径获取文件名
	filename := strings.TrimPrefix(r.URL.Path, "/api/files/share/")
	if filename == "" {
		http.Error(w, `{"success": false, "message": "文件名不能为空"}`, http.StatusBadRequest)
		return
	}

	// 解析请求参数
	var req struct {
		ExpireHours int    `json:"expire_hours"` // 过期时间（小时）
		MaxAccess   int    `json:"max_access"`   // 最大访问次数
		Password    string `json:"password"`     // 访问密码
		UserID      int    `json:"user_id"`      // 从前端传递或从token获取
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"success": false, "message": "无效的请求数据"}`, http.StatusBadRequest)
		return
	}

	// 设置默认值
	if req.ExpireHours == 0 {
		req.ExpireHours = 24 * 7 // 默认7天
	}
	if req.UserID == 0 {
		req.UserID = 1 // 默认用户，实际应从JWT token获取
	}

	// 检查文件是否存在
	filepath := filepath.Join("./uploads", filename)
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		http.Error(w, `{"success": false, "message": "文件不存在"}`, http.StatusNotFound)
		return
	}

	// 生成唯一分享ID
	shareID := generateShareID()
	expiresAt := time.Now().Add(time.Duration(req.ExpireHours) * time.Hour)

	// 插入数据库
	_, err := db.Exec(`
		INSERT INTO share_links 
		(id, filename, filepath, share_url, created_by, expires_at, password, max_access, is_active) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		shareID, filename, filepath, fmt.Sprintf("/api/files/shared/%s", shareID), 
		req.UserID, expiresAt, req.Password, req.MaxAccess, true,
	)

	if err != nil {
		log.Printf("❌ 插入分享链接失败: %v", err)
		http.Error(w, `{"success": false, "message": "创建分享链接失败"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("📤 生成分享链接: %s -> %s (有效期: %d小时)", filename, shareID, req.ExpireHours)

	// 返回响应
	response := models.ShareLinkResponse{
		ID:          shareID,
		Filename:    filename,
		ShareURL:    fmt.Sprintf("https://localhost:8000/api/files/shared/%s", shareID),
		CreatedAt:   time.Now(),
		ExpiresAt:   expiresAt,
		AccessCount: 0,
		MaxAccess:   req.MaxAccess,
		IsProtected: req.Password != "",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "分享链接生成成功",
		"data":    response,
	})
}

// AccessSharedFile 通过分享链接访问文件
func AccessSharedFile(w http.ResponseWriter, r *http.Request) {
	shareID := strings.TrimPrefix(r.URL.Path, "/api/files/shared/")
	if shareID == "" {
		http.Error(w, `{"success": false, "message": "分享链接无效"}`, http.StatusBadRequest)
		return
	}

	// 从数据库查询分享链接
	var shareLink models.ShareLink
	err := db.QueryRow(`
		SELECT id, filename, filepath, created_by, created_at, expires_at, 
		       password, access_count, max_access, is_active 
		FROM share_links WHERE id = ?`,
		shareID,
	).Scan(&shareLink.ID, &shareLink.Filename, &shareLink.Filepath, &shareLink.CreatedBy,
		&shareLink.CreatedAt, &shareLink.ExpiresAt, &shareLink.Password,
		&shareLink.AccessCount, &shareLink.MaxAccess, &shareLink.IsActive)

	if err == sql.ErrNoRows {
		http.Error(w, `{"success": false, "message": "分享链接不存在或已过期"}`, http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("❌ 查询分享链接失败: %v", err)
		http.Error(w, `{"success": false, "message": "服务器内部错误"}`, http.StatusInternalServerError)
		return
	}

	// 检查分享链接是否有效
	if !shareLink.IsActive {
		http.Error(w, `{"success": false, "message": "分享链接已被禁用"}`, http.StatusGone)
		return
	}

	// 检查是否过期
	if time.Now().After(shareLink.ExpiresAt) {
		// 标记为过期
		db.Exec("UPDATE share_links SET is_active = FALSE WHERE id = ?", shareID)
		http.Error(w, `{"success": false, "message": "分享链接已过期"}`, http.StatusGone)
		return
	}

	// 检查访问次数限制
	if shareLink.MaxAccess > 0 && shareLink.AccessCount >= shareLink.MaxAccess {
		http.Error(w, `{"success": false, "message": "分享链接访问次数已达上限"}`, http.StatusForbidden)
		return
	}

	// 检查密码保护
	if shareLink.Password != "" {
		password := r.URL.Query().Get("password")
		if password != shareLink.Password {
			http.Error(w, `{"success": false, "message": "访问密码错误"}`, http.StatusUnauthorized)
			return
		}
	}

	// 更新访问计数
	_, err = db.Exec("UPDATE share_links SET access_count = access_count + 1 WHERE id = ?", shareID)
	if err != nil {
		log.Printf("❌ 更新访问计数失败: %v", err)
	}

	// 提供文件下载
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", shareLink.Filename))
	w.Header().Set("Content-Type", "application/octet-stream")
	
	http.ServeFile(w, r, shareLink.Filepath)

	log.Printf("📥 通过分享链接下载: %s (链接: %s, 访问次数: %d)", shareLink.Filename, shareID, shareLink.AccessCount+1)
}

// GetShareLinks 获取用户的分享链接列表
func GetShareLinks(w http.ResponseWriter, r *http.Request) {
	userID := 1 // 实际应从JWT token获取
	
	rows, err := db.Query(`
		SELECT id, filename, share_url, created_at, expires_at, 
		       access_count, max_access, password 
		FROM share_links 
		WHERE created_by = ? AND is_active = TRUE AND expires_at > NOW() 
		ORDER BY created_at DESC`,
		userID,
	)

	if err != nil {
		log.Printf("❌ 查询分享链接列表失败: %v", err)
		http.Error(w, `{"success": false, "message": "获取分享链接失败"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var activeLinks []models.ShareLinkResponse
	for rows.Next() {
		var link models.ShareLink
		err := rows.Scan(&link.ID, &link.Filename, &link.ShareURL, &link.CreatedAt,
			&link.ExpiresAt, &link.AccessCount, &link.MaxAccess, &link.Password)
		if err != nil {
			continue
		}

		activeLinks = append(activeLinks, models.ShareLinkResponse{
			ID:          link.ID,
			Filename:    link.Filename,
			ShareURL:    fmt.Sprintf("https://localhost:8000/api/files/shared/%s", link.ID),
			CreatedAt:   link.CreatedAt,
			ExpiresAt:   link.ExpiresAt,
			AccessCount: link.AccessCount,
			MaxAccess:   link.MaxAccess,
			IsProtected: link.Password != "",
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    activeLinks,
	})
}

// DeleteShareLink 删除分享链接
func DeleteShareLink(w http.ResponseWriter, r *http.Request) {
	shareID := strings.TrimPrefix(r.URL.Path, "/api/files/share/delete/")
	if shareID == "" {
		http.Error(w, `{"success": false, "message": "分享ID不能为空"}`, http.StatusBadRequest)
		return
	}

	_, err := db.Exec("UPDATE share_links SET is_active = FALSE WHERE id = ?", shareID)
	if err != nil {
		log.Printf("❌ 删除分享链接失败: %v", err)
		http.Error(w, `{"success": false, "message": "删除分享链接失败"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("🗑️ 删除分享链接: %s", shareID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "分享链接已删除",
	})
}

// 生成分享ID
func generateShareID() string {
	timestamp := time.Now().UnixNano()
	randomData := fmt.Sprintf("%d%s%d", timestamp, "family-drive-share", timestamp)
	return fmt.Sprintf("%x", md5.Sum([]byte(randomData)))[:12] // 取前12位
}