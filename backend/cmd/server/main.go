package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
        "io"
)

// WebSocket 升级器
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有跨域请求
	},
}

// 客户端连接
type Client struct {
	conn *websocket.Conn
	send chan []byte
}

// 聊天消息存储
var (
	chatMessages []map[string]interface{}
	chatMutex    sync.RWMutex
	messageID    = 1

	// WebSocket Hub
	clients    = make(map[*Client]bool)
	broadcast  = make(chan []byte)
	register   = make(chan *Client)
	unregister = make(chan *Client)
)

func init() {
	// 初始化聊天消息
	chatMessages = []map[string]interface{}{
		{
			"id":        messageID,
			"user_id":   1,
			"username":  "🏠 家庭网盘",
			"content":   "🎉 欢迎使用家庭网盘和聊天室！",
			"type":      "system",
			"timestamp": time.Now().Format("2006-01-02 15:04:05"),
		},
	}
	messageID++
}

// WebSocket Hub 管理
func runHub() {
	for {
		select {
		case client := <-register:
			clients[client] = true
			log.Printf("👤 新客户端连接，当前客户端数: %d", len(clients))

		case client := <-unregister:
			if _, ok := clients[client]; ok {
				delete(clients, client)
				close(client.send)
			}
			log.Printf("👤 客户端断开，当前客户端数: %d", len(clients))

		case message := <-broadcast:
			// 广播消息给所有客户端
			for client := range clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(clients, client)
				}
			}
		}
	}
}

// 处理 WebSocket 连接
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket 升级失败: %v", err)
		return
	}

	client := &Client{
		conn: conn,
		send: make(chan []byte, 256),
	}

	register <- client

	// 启动读写协程
	go client.writePump()
	go client.readPump()
}

// 写入消息到客户端
func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
		unregister <- c
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		}
	}
}

// 从客户端读取消息
func (c *Client) readPump() {
	defer func() {
		c.conn.Close()
		unregister <- c
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		// 处理收到的消息（可选，用于客户端直接通过 WebSocket 发送消息）
		var msgData map[string]interface{}
		if err := json.Unmarshal(message, &msgData); err == nil {
			if action, ok := msgData["action"].(string); ok && action == "ping" {
				// 处理心跳
				response := map[string]interface{}{
					"action":    "pong",
					"timestamp": time.Now().Unix(),
				}
				responseBytes, _ := json.Marshal(response)
				c.send <- responseBytes
			}
		}
	}
}

// 广播新消息给所有客户端
func broadcastNewMessage(message map[string]interface{}) {
	messageData := map[string]interface{}{
		"type":    "new_message",
		"message": message,
	}

	messageBytes, err := json.Marshal(messageData)
	if err != nil {
		log.Printf("广播消息编码失败: %v", err)
		return
	}

	broadcast <- messageBytes
	log.Printf("📢 广播消息给 %d 个客户端", len(clients))
}

// CORS 中间件
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	}
}

// 聊天API - 获取消息
func handleChatMessages(w http.ResponseWriter, r *http.Request) {
	chatMutex.RLock()
	defer chatMutex.RUnlock()

	response := map[string]interface{}{
		"success":   true,
		"data":      chatMessages,
		"count":     len(chatMessages),
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// 聊天API - 发送消息
func handleChatSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Username string `json:"username"`
		Content  string `json:"content"`
		UserID   int    `json:"user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if request.Content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}

	if request.Username == "" {
		request.Username = "匿名用户"
	}

	chatMutex.Lock()
	newMessage := map[string]interface{}{
		"id":        messageID,
		"user_id":   request.UserID,
		"username":  request.Username,
		"content":   request.Content,
		"type":      "user",
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
	}
	messageID++
	chatMessages = append(chatMessages, newMessage)
	chatMutex.Unlock()

	// 🆕 广播新消息给所有连接的客户端
	broadcastNewMessage(newMessage)

	response := map[string]interface{}{
		"success": true,
		"message": "消息发送成功",
		"data":    newMessage,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	log.Printf("💬 新消息: %s: %s (广播给 %d 客户端)", request.Username, request.Content, len(clients))
}

// 聊天API - 清除消息
func handleChatClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	chatMutex.Lock()
	// 清空消息但保留一条系统消息
	chatMessages = []map[string]interface{}{}
	messageID = 1

	// 添加一条新的系统消息
	systemMessage := map[string]interface{}{
		"id":        messageID,
		"user_id":   1,
		"username":  "🏠 家庭网盘",
		"content":   "💬 聊天记录已清空，开始新的对话吧！",
		"type":      "system",
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
	}
	messageID++
	chatMessages = append(chatMessages, systemMessage)
	chatMutex.Unlock()

	// 🆕 广播清除消息通知
	broadcastNewMessage(systemMessage)

	response := map[string]interface{}{
		"success": true,
		"message": "聊天记录已清空",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	log.Printf("🗑️ 聊天记录已清空 (通知 %d 客户端)", len(clients))
}

// 文件列表处理 - 返回真实文件数据
func handleFileList(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 🆕 扫描 uploads 目录，返回真实文件列表
	files, err := os.ReadDir("uploads")
	if err != nil {
		// 如果目录不存在，返回空数组
		response := map[string]interface{}{
			"success": true,
			"data":    []interface{}{},
			"message": "文件目录为空",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// 🆕 构建文件信息列表
	fileList := make([]map[string]interface{}, 0)
	
	for _, file := range files {
		if !file.IsDir() {
			info, _ := file.Info()
			fileList = append(fileList, map[string]interface{}{
				"id":         strings.ReplaceAll(file.Name(), ".", ""), // 简单ID生成
				"name":       file.Name(),
				"size":       info.Size(),
				"type":       "file", // 可以根据扩展名判断具体类型
				"uploadTime": info.ModTime().Format("2006-01-02 15:04:05"),
			})
		}
	}

	response := map[string]interface{}{
		"success": true,
		"data":    fileList,
		"message": fmt.Sprintf("找到 %d 个文件", len(fileList)),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	log.Printf("📁 返回文件列表: %d 个文件", len(fileList))
}

// 文件上传处理 - 终极修复版本
func handleFileUpload(w http.ResponseWriter, r *http.Request) {
    log.Printf("🔍 开始处理文件上传请求")
    
    if r.Method != "POST" {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // 确保上传目录存在
    if err := os.MkdirAll("uploads", 0755); err != nil {
        log.Printf("❌ 创建上传目录失败: %v", err)
        http.Error(w, "创建上传目录失败", http.StatusInternalServerError)
        return
    }

    // 🆕 完全手动解析 multipart
    // 不要使用 ParseMultipartForm，直接读取 body
    reader, err := r.MultipartReader()
    if err != nil {
        log.Printf("❌ 创建 multipart reader 失败: %v", err)
        http.Error(w, "无效的multipart数据", http.StatusBadRequest)
        return
    }

    // 读取第一个 part
    part, err := reader.NextPart()
    if err != nil {
        log.Printf("❌ 读取 part 失败: %v", err)
        http.Error(w, "无法读取文件部分", http.StatusBadRequest)
        return
    }

    // 获取文件名
    filename := part.FileName()
    if filename == "" {
        http.Error(w, "文件名不能为空", http.StatusBadRequest)
        return
    }

    log.Printf("📤 开始上传文件: %s", filename)

    // 创建目标文件
    filePath := "uploads/" + filename
    dst, err := os.Create(filePath)
    if err != nil {
        log.Printf("❌ 创建文件失败: %v", err)
        http.Error(w, "无法创建文件", http.StatusInternalServerError)
        return
    }
    defer dst.Close()

    // 🆕 手动逐块读取和写入
    buffer := make([]byte, 4096)
    totalWritten := 0
    
    for {
        n, readErr := part.Read(buffer)
        if n > 0 {
            written, writeErr := dst.Write(buffer[:n])
            if writeErr != nil {
                log.Printf("❌ 写入文件失败: %v", writeErr)
                dst.Close()
                os.Remove(filePath)
                http.Error(w, "写入文件失败", http.StatusInternalServerError)
                return
            }
            totalWritten += written
            log.Printf("📝 已写入 %d 字节，累计 %d 字节", written, totalWritten)
        }
        
        if readErr == io.EOF {
            break
        }
        if readErr != nil {
            log.Printf("❌ 读取数据失败: %v", readErr)
            dst.Close()
            os.Remove(filePath)
            http.Error(w, "读取文件数据失败", http.StatusInternalServerError)
            return
        }
    }

    // 强制同步到磁盘
    if err := dst.Sync(); err != nil {
        log.Printf("⚠️ 同步文件失败: %v", err)
    }

    // 验证文件
    fileInfo, err := os.Stat(filePath)
    if err != nil {
        log.Printf("❌ 无法验证文件: %v", err)
        http.Error(w, "文件验证失败", http.StatusInternalServerError)
        return
    }

    log.Printf("✅ 文件上传完成: %s (总大小: %d 字节)", filename, fileInfo.Size())

    response := map[string]interface{}{
        "success": true,
        "message": "文件上传成功",
        "data": map[string]interface{}{
            "filename": filename,
            "size":     fileInfo.Size(),
        },
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
    
    log.Printf("📨 返回上传成功响应")
}
// 文件下载处理 - 正确版本（返回文件内容）
func handleFileDownload(w http.ResponseWriter, r *http.Request) {
    // 从URL路径中提取文件名
    filename := strings.TrimPrefix(r.URL.Path, "/api/files/download/")
    
    if filename == "" {
        http.Error(w, "文件名不能为空", http.StatusBadRequest)
        return
    }

    // 检查文件是否存在
    filePath := "uploads/" + filename
    fileInfo, err := os.Stat(filePath)
    if os.IsNotExist(err) {
        http.Error(w, "文件不存在", http.StatusNotFound)
        return
    }

    // 🆕 打开文件
    file, err := os.Open(filePath)
    if err != nil {
        log.Printf("❌ 无法打开文件: %v", err)
        http.Error(w, "无法打开文件", http.StatusInternalServerError)
        return
    }
    defer file.Close()

    // 🆕 设置正确的 HTTP 头部（关键！）
    w.Header().Set("Content-Disposition", "attachment; filename="+filename)
    w.Header().Set("Content-Type", "application/octet-stream")
    w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

    // 🆕 直接发送文件内容到响应
    _, err = io.Copy(w, file)
    if err != nil {
        log.Printf("❌ 发送文件失败: %v", err)
        return
    }

    log.Printf("✅ 文件下载成功: %s (大小: %d bytes)", filename, fileInfo.Size())
}
// 文件删除处理
func handleFileDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 从URL路径中提取文件名
	filename := strings.TrimPrefix(r.URL.Path, "/api/files/delete/")
	
	if filename == "" {
		http.Error(w, "文件名不能为空", http.StatusBadRequest)
		return
	}

	// 删除文件
	filePath := "uploads/" + filename
	err := os.Remove(filePath)
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"message": "文件删除失败: " + err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "文件删除成功",
		"data": map[string]interface{}{
			"filename": filename,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	log.Printf("🗑️ 文件删除: %s", filename)
}

// 健康检查
func handleHealth(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"status":    "running",
		"service":   "家庭网盘完整服务器",
		"version":   "1.0.0",
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
		"features":  []string{"认证", "文件管理", "实时聊天"},
		"clients":   len(clients), // 🆕 显示当前连接客户端数
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func main() {
	// 🆕 启动 WebSocket Hub
	go runHub()

	// 🆕 使用 ServeMux 明确路由
	mux := http.NewServeMux()
	
	mux.HandleFunc("/api/files/list", corsMiddleware(loggingMiddleware(handleFileList)))
	mux.HandleFunc("/api/files/upload", corsMiddleware(loggingMiddleware(handleFileUpload)))
	mux.HandleFunc("/api/files/download/", corsMiddleware(loggingMiddleware(handleFileDownload)))
	mux.HandleFunc("/api/files/delete/", corsMiddleware(loggingMiddleware(handleFileDelete)))
	mux.HandleFunc("/api/chat/messages", corsMiddleware(loggingMiddleware(handleChatMessages)))
	mux.HandleFunc("/api/chat/send", corsMiddleware(loggingMiddleware(handleChatSend)))
	mux.HandleFunc("/api/chat/clear", corsMiddleware(loggingMiddleware(handleChatClear)))
	mux.HandleFunc("/ws", corsMiddleware(loggingMiddleware(handleWebSocket)))
	mux.HandleFunc("/", corsMiddleware(loggingMiddleware(handleHealth)))

	addr := "0.0.0.0:8000"

	fmt.Println("🚀 家庭网盘完整服务器启动成功!")
	fmt.Println("📍 服务地址: http://localhost:8000")
	fmt.Println("🔗 WebSocket: ws://localhost:8000/ws")
	fmt.Println("💬 聊天接口: http://localhost:8000/api/chat/messages")
	fmt.Println("📁 文件接口: http://localhost:8000/api/files/list")
	fmt.Println("🗑️  清除聊天: http://localhost:8000/api/chat/clear")
	fmt.Println("⏰ 启动时间:", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println("==================================================")

	log.Fatal(http.ListenAndServe(addr, mux)) // 🆕 使用 mux
}