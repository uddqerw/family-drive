package handlers

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "net/url"  // 🆕 添加这个导入
    "os"
    "path/filepath"
    "strings"
    "time"
)

// 🆕 在文件顶部定义 Share 结构体
type Share struct {
    ID          string    `db:"id" json:"id"`
    Filename    string    `db:"filename" json:"filename"`
    Password    string    `db:"password" json:"password"`
    ExpiresAt   time.Time `db:"expires_at" json:"expires_at"`
    MaxAccess   int       `db:"max_access" json:"max_access"`
    AccessCount int       `db:"access_count" json:"access_count"`
    UserID      int       `db:"user_id" json:"user_id"`
    CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

// 创建分享 - 标准库版本
func CreateShare(w http.ResponseWriter, r *http.Request) {
    // 设置 CORS 头
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
    w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

    // 处理预检请求
    if r.Method == "OPTIONS" {
        w.WriteHeader(http.StatusOK)
        return
    }

    if r.Method != "POST" {
        http.Error(w, `{"success":false,"message":"方法不允许"}`, http.StatusMethodNotAllowed)
        return
    }

    // 从 URL 中获取 filename
    filename := strings.TrimPrefix(r.URL.Path, "/api/files/share/")
    if filename == "" {
        http.Error(w, `{"success":false,"message":"文件名不能为空"}`, http.StatusBadRequest)
        return
    }

    // 解析 JSON 请求体
    var req struct {
        ExpireHours int    `json:"expire_hours"`
        MaxAccess   int    `json:"max_access"`
        Password    string `json:"password"`
        UserID      int    `json:"user_id"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, `{"success":false,"message":"无效的请求数据"}`, http.StatusBadRequest)
        return
    }

    // 检查文件是否存在
    filePath := filepath.Join("uploads", filename)
    if !fileExists(filePath) {
        http.Error(w, `{"success":false,"message":"文件不存在"}`, http.StatusNotFound)
        return
    }

    // 生成分享ID
    shareID := generateShareID()
    expiresAt := time.Now().Add(time.Duration(req.ExpireHours) * time.Hour)

    // 保存到数据库
    _, err := db.Exec(`
        INSERT INTO shares (id, filename, password, expires_at, max_access, access_count, user_id, created_at)
        VALUES (?, ?, ?, ?, ?, 0, ?, NOW())
    `, shareID, filename, req.Password, expiresAt, req.MaxAccess, req.UserID)

    if err != nil {
        log.Printf("创建分享失败: %v", err)
        http.Error(w, `{"success":false,"message":"创建分享失败"}`, http.StatusInternalServerError)
        return
    }

    // 返回成功响应
    response := map[string]interface{}{
        "success": true,
        "message": "分享链接创建成功",
        "data": map[string]interface{}{
            "id":         shareID,
            "filename":   filename,
            "share_url":  fmt.Sprintf("https://localhost:8000/api/files/shared/%s", shareID),
            "created_at": time.Now().Format("2006/01/02 15:04:05"),
            "expires_at": expiresAt.Format("2006/01/02 15:04:05"),
        },
    }

    json.NewEncoder(w).Encode(response)
}

// 获取分享文件 - 优化版本
func GetSharedFile(w http.ResponseWriter, r *http.Request) {
    // 设置 CORS 头
    w.Header().Set("Access-Control-Allow-Origin", "*")

    // 从 URL 中获取分享ID
    shareID := strings.TrimPrefix(r.URL.Path, "/api/files/shared/")
    if shareID == "" {
        http.Error(w, `{"success":false,"message":"分享ID不能为空"}`, http.StatusBadRequest)
        return
    }

    // 从数据库获取分享信息
    var share Share
    err := db.QueryRow(`
        SELECT id, filename, password, expires_at, max_access, access_count, user_id, created_at 
        FROM shares WHERE id = ?
    `, shareID).Scan(
        &share.ID, &share.Filename, &share.Password, &share.ExpiresAt,
        &share.MaxAccess, &share.AccessCount, &share.UserID, &share.CreatedAt,
    )

    if err == sql.ErrNoRows {
        http.Error(w, `{"success":false,"message":"分享链接不存在或已过期"}`, http.StatusNotFound)
        return
    } else if err != nil {
        log.Printf("查询分享失败: %v", err)
        http.Error(w, `{"success":false,"message":"服务器错误"}`, http.StatusInternalServerError)
        return
    }

    // 检查是否过期
    if time.Now().After(share.ExpiresAt) {
        http.Error(w, `{"success":false,"message":"分享链接已过期"}`, http.StatusBadRequest)
        return
    }

    // 检查访问次数限制
    if share.MaxAccess > 0 && share.AccessCount >= share.MaxAccess {
        http.Error(w, `{"success":false,"message":"分享链接访问次数已达上限"}`, http.StatusBadRequest)
        return
    }

    // 如果有密码，验证密码
    if share.Password != "" {
        providedPassword := r.URL.Query().Get("password")
        
        // 如果没提供密码，返回密码输入页面
        if providedPassword == "" {
            // 重定向到密码输入页面
            http.Redirect(w, r, 
                fmt.Sprintf("/static/file_password.html?id=%s&filename=%s", 
                    shareID, 
                    url.QueryEscape(share.Filename)), 
                http.StatusFound)
            return
        }

        // 验证密码
        if providedPassword != share.Password {
            // 重定向回密码页面并显示错误
            http.Redirect(w, r, 
                fmt.Sprintf("/static/file_password.html?id=%s&filename=%s&error=%s", 
                    shareID, 
                    url.QueryEscape(share.Filename),
                    url.QueryEscape("密码错误，请重新输入")), 
                http.StatusFound)
            return
        }
    }

    // 更新访问次数
    _, err = db.Exec("UPDATE shares SET access_count = access_count + 1 WHERE id = ?", shareID)
    if err != nil {
        log.Printf("更新访问次数失败: %v", err)
    }

    // 提供文件下载
    filePath := filepath.Join("uploads", share.Filename)
    
    // 检查文件是否存在
    if !fileExists(filePath) {
        http.Error(w, `{"success":false,"message":"文件不存在"}`, http.StatusNotFound)
        return
    }

    // 设置下载头
    w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", share.Filename))
    w.Header().Set("Content-Type", "application/octet-stream")
    
    // 提供文件下载
    http.ServeFile(w, r, filePath)
}

// 辅助函数
func generateShareID() string {
    return fmt.Sprintf("%x", time.Now().UnixNano())[:12]
}

func fileExists(path string) bool {
    _, err := os.Stat(path)
    return !os.IsNotExist(err)
}