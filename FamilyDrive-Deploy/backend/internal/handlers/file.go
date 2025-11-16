package handlers

import (
    "net/http"
    "os"
    "path/filepath"
    "strconv"
    "io"
)

// 文件信息结构
type FileInfo struct {
    ID        int64  `json:"id"`
    Name      string `json:"name"`
    Size      int64  `json:"size"`
    CreatedAt string `json:"created_at"`
    OwnerID   int64  `json:"owner_id"`
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
    
    writeJSON(w, http.StatusOK, map[string]string{
        "message": "文件上传成功",
        "file": header.Filename,
        "size": strconv.FormatInt(header.Size, 10),
        "owner_id": strconv.FormatInt(uid, 10), // 使用uid变量
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
            info, err := file.Info()
            if err == nil {
                fileList = append(fileList, FileInfo{
                    Name: file.Name(),
                    Size: info.Size(),
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