package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
	
	"github.com/gorilla/websocket"
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
					"action": "pong",
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

	// 设置路由
	http.HandleFunc("/ws", corsMiddleware(loggingMiddleware(handleWebSocket))) // 🆕 WebSocket 端点
	http.HandleFunc("/api/chat/messages", corsMiddleware(loggingMiddleware(handleChatMessages)))
	http.HandleFunc("/api/chat/send", corsMiddleware(loggingMiddleware(handleChatSend)))
	http.HandleFunc("/api/chat/clear", corsMiddleware(loggingMiddleware(handleChatClear)))
	http.HandleFunc("/", corsMiddleware(loggingMiddleware(handleHealth)))

	port := ":8000"

	fmt.Println("🚀 家庭网盘完整服务器启动成功!")
	fmt.Println("📍 服务地址: http://localhost" + port)
	fmt.Println("🔗 WebSocket: ws://localhost" + port + "/ws")
	fmt.Println("💬 聊天接口: http://localhost" + port + "/api/chat/messages")
	fmt.Println("🗑️  清除聊天: http://localhost" + port + "/api/chat/clear")
	fmt.Println("⏰ 启动时间:", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println("==================================================")

	log.Fatal(http.ListenAndServe(port, nil))
}