package handlers

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "net/url"
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "time"
)

// 文件信息结构
type FileInfo struct {
    ID        int64  `json:"id"`
    Name      string `json:"name"`
    Size      int64  `json:"size"`
    CreatedAt string `json:"created_at"`
    OwnerID   int64  `json:"owner_id"`
    IsPrivate bool   `json:"isPrivate"` // 🆕 添加私密文件标识
}

// 上传文件
func HandleFileUpload(w http.ResponseWriter, r *http.Request) {
    // 验证用户认证
    uid, err := getAuthUserID(r)
    if err != nil {
        writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
        return
    }

    // 解析 multipart 表单
    err = r.ParseMultipartForm(32 << 20) // 32MB
    if err != nil {
        writeJSON(w, http.StatusBadRequest, map[string]string{"error": "parse form failed"})
        return
    }

    file, header, err := r.FormFile("file")
    if err != nil {
        writeJSON(w, http.StatusBadRequest, map[string]string{"error": "get file failed"})
        return
    }
    defer file.Close()

    // 🆕 获取私密文件选项
    isPrivate := r.FormValue("is_private") == "true"
    sharePassword := r.FormValue("share_password")

    // 创建上传目录
    uploadDir := "./uploads"
    os.MkdirAll(uploadDir, 0755)

    // 保存文件
    filePath := filepath.Join(uploadDir, header.Filename)
    dst, err := os.Create(filePath)
    if err != nil {
        writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "create file failed"})
        return
    }
    defer dst.Close()

    // 复制文件内容
    _, err = io.Copy(dst, file)
    if err != nil {
        writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "save file failed"})
        return
    }

    // 🆕 保存到 share_links 表（创建分享链接）
    shareID := generateShareID()
    expiresAt := time.Now().Add(24 * 365 * time.Hour) // 1年有效期

    _, err = db.Exec(`
        INSERT INTO share_links (id, filename, password, expires_at, max_access, access_count, user_id, created_at, is_private, share_password) 
        VALUES (?, ?, ?, ?, 0, 0, ?, NOW(), ?, ?)
    `, shareID, header.Filename, sharePassword, expiresAt, uid, isPrivate, sharePassword)

    if err != nil {
        log.Printf("创建分享链接失败: %v", err)
        // 不返回错误，因为文件已经上传成功
    }

    writeJSON(w, http.StatusOK, map[string]interface{}{
        "message": "文件上传成功",
        "file":    header.Filename,
        "size":    strconv.FormatInt(header.Size, 10),
        "owner_id": strconv.FormatInt(uid, 10),
        "isPrivate": isPrivate, // 🆕 返回私密状态
        "share_url": fmt.Sprintf("https://localhost:8000/api/files/shared/%s", shareID), // 🆕 返回分享链接
    })
}

// 获取文件列表
func HandleFileList(w http.ResponseWriter, r *http.Request) {
    // 验证用户
    uid, err := getAuthUserID(r)
    if err != nil {
        writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
        return
    }

    uploadDir := "./uploads"

    // 确保上传目录存在
    if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
        // 目录不存在，返回空数组
        writeJSON(w, http.StatusOK, []FileInfo{})
        return
    }

    // 读取目录
    files, err := os.ReadDir(uploadDir)
    if err != nil {
        writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "read dir failed"})
        return
    }

    var fileList []FileInfo
    for _, file := range files {
        if !file.IsDir() {
            info, err := file.Info()
            if err == nil {
                // 🆕 查询文件的私密状态
                var isPrivate bool
                err := db.QueryRow(`
                    SELECT is_private FROM share_links 
                    WHERE filename = ? AND user_id = ? 
                    ORDER BY created_at DESC LIMIT 1
                `, file.Name(), uid).Scan(&isPrivate)
                
                // 如果查询失败，默认为非私密
                if err != nil {
                    isPrivate = false
                }

                fileList = append(fileList, FileInfo{
                    Name:      file.Name(),
                    Size:      info.Size(),
                    IsPrivate: isPrivate, // 🆕 添加私密标识
                })
            }
        }
    }

    writeJSON(w, http.StatusOK, fileList)
}

// 下载文件
func HandleFileDownload(w http.ResponseWriter, r *http.Request) {
    // 验证用户认证
    uid, err := getAuthUserID(r)
    if err != nil {
        writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
        return
    }

    // 从URL路径获取文件名
    fileName := r.URL.Path[len("/api/files/download/"):]
    if fileName == "" {
        writeJSON(w, http.StatusBadRequest, map[string]string{"error": "filename required"})
        return
    }

    filePath := filepath.Join("./uploads", fileName)

    // 检查文件是否存在
    if _, err := os.Stat(filePath); os.IsNotExist(err) {
        writeJSON(w, http.StatusNotFound, map[string]string{"error": "file not found"})
        return
    }

    // 🆕 查询文件的私密状态和密码
    var isPrivate bool
    var sharePassword string
    err = db.QueryRow(`
        SELECT is_private, share_password FROM share_links 
        WHERE filename = ? AND user_id = ? 
        ORDER BY created_at DESC LIMIT 1
    `, fileName, uid).Scan(&isPrivate, &sharePassword)

    // 🆕 如果是私密文件，验证密码
    if err == nil && isPrivate && sharePassword != "" {
        providedPassword := r.URL.Query().Get("password")
        
        // 如果没提供密码，返回密码输入页面
        if providedPassword == "" {
            http.Redirect(w, r, 
                fmt.Sprintf("/static/file_password.html?filename=%s", 
                    url.QueryEscape(fileName)), 
                http.StatusFound)
            return
        }

        // 验证密码
        if providedPassword != sharePassword {
            http.Redirect(w, r, 
                fmt.Sprintf("/static/file_password.html?filename=%s&error=%s", 
                    url.QueryEscape(fileName),
                    url.QueryEscape("密码错误，请重新输入")), 
                http.StatusFound)
            return
        }
    }

    // 设置下载头信息
    w.Header().Set("Content-Disposition", "attachment; filename="+fileName)
    w.Header().Set("Content-Type", "application/octet-stream")

    // 提供文件下载
    http.ServeFile(w, r, filePath)
}

// 删除文件
func HandleFileDelete(w http.ResponseWriter, r *http.Request) {
    // 验证用户认证
    uid, err := getAuthUserID(r)
    if err != nil {
        writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
        return
    }

    // 从URL路径获取文件名
    fileName := r.URL.Path[len("/api/files/delete/"):]
    if fileName == "" {
        writeJSON(w, http.StatusBadRequest, map[string]string{"error": "filename required"})
        return
    }

    filePath := filepath.Join("./uploads", fileName)

    // 检查文件是否存在
    if _, err := os.Stat(filePath); os.IsNotExist(err) {
        writeJSON(w, http.StatusNotFound, map[string]string{"error": "file not found"})
        return
    }

    // 🆕 删除分享链接记录
    _, err = db.Exec("DELETE FROM share_links WHERE filename = ? AND user_id = ?", fileName, uid)
    if err != nil {
        log.Printf("删除分享链接失败: %v", err)
    }

    // 删除文件
    err = os.Remove(filePath)
    if err != nil {
        writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "delete file failed"})
        return
    }

    writeJSON(w, http.StatusOK, map[string]string{
        "message": "文件删除成功",
        "file": fileName,
    })
}

// 🆕 辅助函数 - 生成分享ID
func generateShareID() string {
    return fmt.Sprintf("%x", time.Now().UnixNano())[:12]
}

// 🆕 辅助函数 - 写入JSON响应
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(data)
}