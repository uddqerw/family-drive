package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// 聊天消息存储
var (
	chatMessages []map[string]interface{}
	chatMutex    sync.RWMutex
	messageID    = 1
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

	response := map[string]interface{}{
		"success": true,
		"message": "消息发送成功",
		"data":    newMessage,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	log.Printf("💬 新消息: %s: %s", request.Username, request.Content)
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
	chatMessages = append(chatMessages, map[string]interface{}{
		"id":        messageID,
		"user_id":   1,
		"username":  "🏠 家庭网盘",
		"content":   "💬 聊天记录已清空，开始新的对话吧！",
		"type":      "system",
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
	})
	messageID++
	chatMutex.Unlock()

	response := map[string]interface{}{
		"success": true,
		"message": "聊天记录已清空",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	log.Printf("🗑️ 聊天记录已清空")
}

// 健康检查
func handleHealth(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"status":    "running",
		"service":   "家庭网盘完整服务器",
		"version":   "1.0.0",
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
		"features":  []string{"认证", "文件管理", "实时聊天"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func main() {
	// 设置路由
	http.HandleFunc("/api/chat/messages", corsMiddleware(loggingMiddleware(handleChatMessages)))
	http.HandleFunc("/api/chat/send", corsMiddleware(loggingMiddleware(handleChatSend)))
	http.HandleFunc("/api/chat/clear", corsMiddleware(loggingMiddleware(handleChatClear)))
	http.HandleFunc("/", corsMiddleware(loggingMiddleware(handleHealth)))

	port := ":8000"

	fmt.Println("🚀 家庭网盘完整服务器启动成功!")
	fmt.Println("📍 服务地址: http://localhost" + port)
	fmt.Println("💬 聊天接口: http://localhost" + port + "/api/chat/messages")
	fmt.Println("🗑️  清除聊天: http://localhost" + port + "/api/chat/clear")
	fmt.Println("⏰ 启动时间:", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println("==================================================")

	log.Fatal(http.ListenAndServe(port, nil))
}