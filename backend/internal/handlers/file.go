package handlers

import (
    "fmt"
    "io"
    "log"
    "net/http"
    "net/url"
    "os"
    "path/filepath"
    "strconv"
    "strings"
)

// 文件信息结构
type FileInfo struct {
    ID        int64  `json:"id"`
    Name      string `json:"name"`
    Size      int64  `json:"size"`
    CreatedAt string `json:"created_at"`
    OwnerID   int64  `json:"owner_id"`
    IsPrivate bool   `json:"isPrivate"`
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

    // 获取私密文件选项
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

    // 🆕 如果是私密文件，创建密码标记文件
    if isPrivate && sharePassword != "" {
        privateFilePath := filepath.Join(uploadDir, "."+header.Filename+".private")
        err = os.WriteFile(privateFilePath, []byte(sharePassword), 0644)
        if err != nil {
            log.Printf("创建私密标记文件失败: %v", err)
        } else {
            log.Printf("✅ 私密文件标记创建成功: %s", header.Filename)
        }
    }

    writeJSON(w, http.StatusOK, map[string]interface{}{
        "message": "文件上传成功",
        "file":    header.Filename,
        "size":    strconv.FormatInt(header.Size, 10),
        "owner_id": strconv.FormatInt(uid, 10),
        "isPrivate": isPrivate,
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
    _ = uid // 使用变量避免编译警告

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
            // 🆕 跳过私密标记文件
            if strings.HasPrefix(file.Name(), ".") && strings.HasSuffix(file.Name(), ".private") {
                continue
            }
            
            info, err := file.Info()
            if err == nil {
                // 🆕 检查是否为私密文件
                privateFilePath := filepath.Join(uploadDir, "."+file.Name()+".private")
                isPrivate := false
                
                if _, err := os.Stat(privateFilePath); err == nil {
                    isPrivate = true
                }

                fileList = append(fileList, FileInfo{
                    Name:      file.Name(),
                    Size:      info.Size(),
                    IsPrivate: isPrivate,
                })
            }
        }
    }

    writeJSON(w, http.StatusOK, fileList)
}

// 下载文件
func HandleFileDownload(w http.ResponseWriter, r *http.Request) {
    // 验证用户认证
    _, err := getAuthUserID(r)
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

    // 🆕 检查是否为私密文件
    privateFilePath := filepath.Join("./uploads", "."+fileName+".private")
    if _, err := os.Stat(privateFilePath); err == nil {
        // 是私密文件，需要密码验证
        providedPassword := r.URL.Query().Get("password")
        
        // 读取存储的密码
        storedPassword, err := os.ReadFile(privateFilePath)
        if err != nil {
            log.Printf("读取私密文件密码失败: %v", err)
            writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "服务器错误"})
            return
        }

        // 如果没提供密码，返回密码输入页面
        if providedPassword == "" {
            http.Redirect(w, r, 
                fmt.Sprintf("/static/file_password.html?filename=%s", 
                    url.QueryEscape(fileName)), 
                http.StatusFound)
            return
        }

        // 验证密码
        if providedPassword != string(storedPassword) {
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
    _, err := getAuthUserID(r)
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

    // 🆕 删除私密标记文件（如果存在）
    privateFilePath := filepath.Join("./uploads", "."+fileName+".private")
    if _, err := os.Stat(privateFilePath); err == nil {
        os.Remove(privateFilePath)
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

// 辅助函数 - 获取认证用户ID
// func getAuthUserID(r *http.Request) (int64, error) {
    // 简化：暂时返回固定用户ID
    // return 1, nil
// }