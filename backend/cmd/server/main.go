package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"net/url"

        "familydrive/handlers"
        "familydrive/middleware"
	
	"golang.org/x/crypto/bcrypt"
	"github.com/gorilla/websocket"
	_ "github.com/go-sql-driver/mysql"
        // "github.com/gin-gonic/gin"
)

// 聊天消息结构
type ChatMessage struct {
	ID        int    `json:"id"`
	UserID    int    `json:"user_id"`
	Username  string `json:"username"`
	Content   string `json:"content"`
	Type      string `json:"type"`
	VoiceURL  string `json:"voice_url,omitempty"`
	Duration  int    `json:"duration,omitempty"`
	Timestamp string `json:"timestamp"`
}

// 用户结构
type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

// 全局变量
var (
	chatMessages []ChatMessage
	mutex        sync.Mutex
	clients      = make(map[*websocket.Conn]bool)
	broadcast    = make(chan ChatMessage)
	port         = "8000"
	db           *sql.DB // MySQL 数据库连接
	upgrader     = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // 允许所有来源
		},
	}
)

// CORS 中间件
func enableCORS(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 设置 CORS 头
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
        
        // 处理预检请求
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }
        
        // 调用下一个处理器
        next(w, r)
    }
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

// 初始化数据库
func initDB() {
	var err error
	// MySQL 配置
	dsn := "root:Root@20160212@tcp(localhost:3306)/family_drive?charset=utf8mb4&parseTime=True"
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("❌ 数据库连接失败:", err)
	}
	
	// 测试连接
	err = db.Ping()
	if err != nil {
		log.Fatal("❌ 数据库连接测试失败:", err)
	}
	
	log.Println("✅ MySQL 数据库连接成功")
	
	// 设置连接池参数
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)
}

// WebSocket 处理
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ WebSocket 升级失败: %v", err)
		return
	}
	defer conn.Close()

	// 注册客户端
	clients[conn] = true
	log.Printf("🔗 WebSocket 客户端连接: %s", r.RemoteAddr)

	// 发送历史消息给新客户端
	mutex.Lock()
	for _, msg := range chatMessages {
		conn.WriteJSON(msg)
	}
	mutex.Unlock()

	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("❌ WebSocket 读取错误: %v", err)
			delete(clients, conn)
			break
		}
		log.Printf("📨 收到 WebSocket 消息: %v", msg)
	}
}

// 广播消息给所有客户端
func broadcastMessage(message ChatMessage) {
	log.Printf("📢 广播消息给 %d 个客户端", len(clients))
	
	for client := range clients {
		err := client.WriteJSON(message)
		if err != nil {
			log.Printf("❌ WebSocket 发送错误: %v", err)
			client.Close()
			delete(clients, client)
		}
	}
}

// 根路径处理
func handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service":   "家庭网盘",
		"status":    "running", 
		"version":   "1.0.0",
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
		"messages":  len(chatMessages),
		"clients":   len(clients),
		"endpoints": map[string]string{
			"websocket":     "/ws",
			"chat_messages": "/api/chat/messages",
			"file_list":     "/api/files/list", 
			"file_upload":   "/api/files/upload",
			"auth_login":    "/api/auth/login",
			"auth_register": "/api/auth/register",
		},
	})
}

// 登录处理 - MySQL 版本
func handleLogin(w http.ResponseWriter, r *http.Request) {
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
	
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

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
	
	// 登录成功
	log.Printf("✅ 用户登录成功: %s (%s)", username, email)
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "登录成功",
		"data": map[string]interface{}{
			"access_token": "family_drive_token_" + fmt.Sprintf("%d", time.Now().Unix()),
			"user": map[string]interface{}{
				"id":       userID,
				"username": username,
				"email":    email,
			},
		},
	})
}

// 注册处理 - MySQL 版本
func handleRegister(w http.ResponseWriter, r *http.Request) {
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
	
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

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
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "注册成功",
		"data": map[string]interface{}{
			"id":       userID,
			"username": req.Username,
			"email":    req.Email,
		},
	})
}

// 发送消息处理
func handleSendMessage(w http.ResponseWriter, r *http.Request) {
	log.Printf("💬 处理发送消息请求")
	
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
	
	var req struct {
		Username string `json:"username"`
		Content  string `json:"content"`
		UserID   int    `json:"user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "无效的请求数据", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Content == "" {
		http.Error(w, "用户名和消息内容不能为空", http.StatusBadRequest)
		return
	}

	message := ChatMessage{
		ID:        len(chatMessages) + 1,
		UserID:    req.UserID,
		Username:  req.Username,
		Content:   req.Content,
		Type:      "user",
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	}

	mutex.Lock()
	chatMessages = append(chatMessages, message)
	mutex.Unlock()

	// 通过 WebSocket 广播消息
	broadcastMessage(message)

	log.Printf("📢 新消息: %s: %s", req.Username, req.Content)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "消息发送成功",
		"data":    message,
	})
}

// 获取消息列表
func handleGetMessages(w http.ResponseWriter, r *http.Request) {
	log.Printf("📨 返回消息列表: %d 条消息", len(chatMessages))
	
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	
	mutex.Lock()
	defer mutex.Unlock()
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    chatMessages,
	})
}

// 清空消息
func handleClearMessages(w http.ResponseWriter, r *http.Request) {
	log.Printf("🗑️ 清空聊天记录")
	
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
	
	mutex.Lock()
	systemMessage := ChatMessage{
		ID:        1,
		UserID:    1,
		Username:  "🏠 家庭网盘",
		Content:   "💬 聊天记录已清空，开始新的对话吧！",
		Type:      "system",
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	}
	chatMessages = []ChatMessage{systemMessage}
	mutex.Unlock()

	// 广播清空消息
	broadcastMessage(systemMessage)

	log.Printf("✅ 聊天记录已清空")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "聊天记录已清空",
	})
}

// 语音消息上传
func handleVoiceUpload(w http.ResponseWriter, r *http.Request) {
	log.Printf("🎤 开始处理语音上传请求")
	
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
	
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, "无法解析表单数据", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("audio")
	if err != nil {
		http.Error(w, "无法获取音频文件", http.StatusBadRequest)
		return
	}
	defer file.Close()

	username := r.FormValue("username")
	userID := r.FormValue("user_id")
	duration := r.FormValue("duration")

	voiceDir := "./uploads/voices"
	if err := os.MkdirAll(voiceDir, 0755); err != nil {
		log.Printf("❌ 创建语音目录失败: %v", err)
		http.Error(w, "服务器内部错误", http.StatusInternalServerError)
		return
	}

	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("voice_%s_%d.webm", username, timestamp)
	filepath := filepath.Join(voiceDir, filename)

	out, err := os.Create(filepath)
	if err != nil {
		log.Printf("❌ 创建文件失败: %v", err)
		http.Error(w, "无法保存文件", http.StatusInternalServerError)
		return
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		log.Printf("❌ 保存文件失败: %v", err)
		http.Error(w, "无法保存文件", http.StatusInternalServerError)
		return
	}

	durationInt, _ := strconv.Atoi(duration)
	userIDInt, _ := strconv.Atoi(userID)
	
	voiceMsg := ChatMessage{
		ID:        int(timestamp),
		UserID:    userIDInt,
		Username:  username,
		Type:      "voice",
		VoiceURL:  "/api/chat/voice/" + filename,
		Duration:  durationInt,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Content:   fmt.Sprintf("[语音消息 %d秒]", durationInt),
	}

	mutex.Lock()
	chatMessages = append(chatMessages, voiceMsg)
	mutex.Unlock()

	// 通过 WebSocket 广播语音消息
	broadcastMessage(voiceMsg)

	log.Printf("✅ 语音上传成功: %s (时长: %s秒)", filename, duration)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "语音上传成功",
		"data":    voiceMsg,
	})
}

// 语音文件下载
func handleVoiceDownload(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimPrefix(r.URL.Path, "/api/chat/voice/")
	if filename == "" {
		http.Error(w, "文件名不能为空", http.StatusBadRequest)
		return
	}

	filepath := filepath.Join("./uploads/voices", filename)
	
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		http.Error(w, "语音文件不存在", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "audio/webm")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filename))
	
	http.ServeFile(w, r, filepath)
}

// 文件上传处理
func handleFileUpload(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔍 开始处理文件上传请求")
	
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
	
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, "无法解析表单数据", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "无法获取文件", http.StatusBadRequest)
		return
	}
	defer file.Close()

	log.Printf("📤 开始上传文件: %s", handler.Filename)

	uploadDir := "./uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Printf("❌ 创建上传目录失败: %v", err)
		http.Error(w, "服务器内部错误", http.StatusInternalServerError)
		return
	}

	dst, err := os.Create(filepath.Join(uploadDir, handler.Filename))
	if err != nil {
		log.Printf("❌ 创建文件失败: %v", err)
		http.Error(w, "无法创建文件", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	bytesWritten, err := io.Copy(dst, file)
	if err != nil {
		log.Printf("❌ 保存文件失败: %v", err)
		http.Error(w, "无法保存文件", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ 文件上传完成: %s (大小: %d 字节)", handler.Filename, bytesWritten)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "文件上传成功",
		"data": map[string]interface{}{
			"filename": handler.Filename,
			"size":     bytesWritten,
		},
	})
}

// 文件下载处理函数
func handleFileDownload(w http.ResponseWriter, r *http.Request) {
    // 设置 CORS 头
    w.Header().Set("Access-Control-Allow-Origin", "*")
    
    // 从 URL 路径获取文件名
    path := r.URL.Path
    prefix := "/api/files/download/"
    
    if !strings.HasPrefix(path, prefix) {
        http.Error(w, `{"error":"invalid path"}`, http.StatusBadRequest)
        return
    }
    
    fileName := path[len(prefix):]
    if fileName == "" {
        http.Error(w, `{"error":"filename required"}`, http.StatusBadRequest)
        return
    }

    // URL 解码文件名
    decodedFileName, err := url.QueryUnescape(fileName)
    if err == nil {
        fileName = decodedFileName
    }

    filePath := filepath.Join("./uploads", fileName)

    // 检查文件是否存在
    if _, err := os.Stat(filePath); os.IsNotExist(err) {
        http.Error(w, `{"error":"file not found"}`, http.StatusNotFound)
        return
    }

    // 设置下载头信息
    w.Header().Set("Content-Disposition", "attachment; filename="+fileName)
    w.Header().Set("Content-Type", "application/octet-stream")
    
    // 提供文件下载
    http.ServeFile(w, r, filePath)
}

// 文件列表处理
func handleFileList(w http.ResponseWriter, r *http.Request) {
	log.Printf("📁 返回文件列表")
	
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	
	uploadDir := "./uploads"
	files, err := os.ReadDir(uploadDir)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data":    []string{},
		})
		return
	}

	var fileList []map[string]interface{}
	for _, file := range files {
		if !file.IsDir() {
			info, err := file.Info()
			if err != nil {
				continue
			}
			
			fileList = append(fileList, map[string]interface{}{
				"name": file.Name(),
				"size": info.Size(),
				"time": info.ModTime().Format("2006-01-02 15:04:05"),
			})
		}
	}

	log.Printf("📁 返回文件列表: %d 个文件", len(fileList))

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    fileList,
	})
}

// 文件删除处理
func handleFileDelete(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimPrefix(r.URL.Path, "/api/files/delete/")
	if filename == "" {
		http.Error(w, "文件名不能为空", http.StatusBadRequest)
		return
	}

	log.Printf("🗑️ 文件删除: %s", filename)
	
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "DELETE" {
		http.Error(w, "只支持DELETE请求", http.StatusMethodNotAllowed)
		return
	}
	
	filepath := filepath.Join("./uploads", filename)
	
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		http.Error(w, "文件不存在", http.StatusNotFound)
		return
	}

	err := os.Remove(filepath)
	if err != nil {
		log.Printf("❌ 文件删除失败: %v", err)
		http.Error(w, "文件删除失败", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ 文件删除成功: %s", filename)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "文件删除成功",
	})
}

func main() {
	// 初始化数据库
	initDB()
	defer db.Close()
        handlers.SetDB(db)
        // handlers.SetShareDB(db)
	// 初始化目录
	os.MkdirAll("./uploads", 0755)
	os.MkdirAll("./uploads/voices", 0755)

        // router := gin.Default()
	// router.LoadHTMLGlob("templates/*")
	
	// 添加欢迎消息
	welcomeMessage := ChatMessage{
		ID:        1,
		UserID:    1,
		Username:  "🏠 家庭网盘",
		Content:   "👋 欢迎使用家庭网盘！开始聊天和分享文件吧！",
		Type:      "system",
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	}
	chatMessages = append(chatMessages, welcomeMessage)

	// 创建路由
        mux := http.NewServeMux()

        // 注册路由 - 全部使用本地函数 + CORS
        mux.HandleFunc("/", handleRoot)
        mux.HandleFunc("/api/auth/login", middleware.CORS(handleLogin))
        mux.HandleFunc("/api/auth/register", middleware.CORS(handleRegister))
        mux.HandleFunc("/api/chat/send", middleware.CORS(handleSendMessage))
        mux.HandleFunc("/api/chat/messages", middleware.CORS(handleGetMessages))
        mux.HandleFunc("/api/chat/clear", middleware.CORS(handleClearMessages))
        mux.HandleFunc("/api/chat/voice", middleware.CORS(handleVoiceUpload))
        mux.HandleFunc("/api/chat/voice/", middleware.CORS(handleVoiceDownload))
        mux.HandleFunc("/api/files/upload", middleware.CORS(handleFileUpload))
        mux.HandleFunc("/api/files/download/", middleware.CORS(handleFileDownload))
        mux.HandleFunc("/api/files/list", middleware.CORS(handleFileList))
        mux.HandleFunc("/api/files/delete/", middleware.CORS(handleFileDelete))
        mux.HandleFunc("/api/files/share/", middleware.CORS(handlers.CreateShare))
        mux.HandleFunc("/api/files/shared/", middleware.CORS(handlers.GetSharedFile))
        // mux.HandleFunc("/api/files/shares", middleware.CORS(handlers.GetShareLinks))
	// mux.HandleFunc("/api/files/share/delete/", middleware.CORS(handlers.DeleteShareLink))
        mux.HandleFunc("/ws", handleWebSocket)

        // 这个需要特殊处理 - 使用 handlers 包函数 + 认证中间件
        mux.HandleFunc("/api/auth/me", middleware.AuthMiddleware(handlers.HandleGetCurrentUser))	

	// 静态文件服务
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	
	log.Println("🚀 家庭网盘 HTTPS 服务器启动成功!")
	log.Printf("📍 服务地址: https://localhost:%s", port)
	log.Printf("🔗 WebSocket: wss://localhost:%s/ws", port)
	log.Printf("💬 聊天接口: https://localhost:%s/api/chat/messages", port)
	log.Printf("📁 文件接口: https://localhost:%s/api/files/list", port)
	log.Printf("🔐 使用安全连接 (HTTPS)")
	log.Printf("⏰ 启动时间: %s", time.Now().Format("2006-01-02 15:04:05"))
	log.Println("==================================================")
	
        // 证书文件路径
        certFile := "localhost+2.pem"      // 证书文件
        keyFile := "localhost+2-key.pem"   // 密钥文件
        
	// 使用 HTTPS
	err := http.ListenAndServeTLS(":"+port, certFile, keyFile, mux)
	if err != nil {
		log.Fatal("HTTPS 服务器启动失败:", err)
	}
}