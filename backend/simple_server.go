package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// 全局变量
var (
	messages   []map[string]interface{}
	mutex      sync.RWMutex
	messageID  = 1
)

func init() {
	// 初始化一些欢迎消息
	messages = []map[string]interface{}{
		{
			"id":        messageID,
			"user_id":   1,
			"username":  "🏠 家庭网盘",
			"content":   "🎉 欢迎来到家庭聊天室！",
			"type":      "system",
			"timestamp": time.Now().Format("2006-01-02 15:04:05"),
		},
	}
	messageID++
}

// CORS 中间件
func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next(w, r)
	}
}

// 获取所有消息
func getMessagesHandler(w http.ResponseWriter, r *http.Request) {
	mutex.RLock()
	defer mutex.RUnlock()
	
	response := map[string]interface{}{
		"success": true,
		"data":    messages,
		"count":   len(messages),
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// 发送新消息
func sendMessageHandler(w http.ResponseWriter, r *http.Request) {
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
	
	mutex.Lock()
	newMessage := map[string]interface{}{
		"id":        messageID,
		"user_id":   request.UserID,
		"username":  request.Username,
		"content":   request.Content,
		"type":      "user",
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
	}
	messageID++
	messages = append(messages, newMessage)
	mutex.Unlock()
	
	response := map[string]interface{}{
		"success": true,
		"message": "消息发送成功",
		"data":    newMessage,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	
	log.Printf("💬 新消息: %s: %s", request.Username, request.Content)
}

// 健康检查
func healthHandler(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"status":    "running",
		"service":   "家庭网盘聊天服务器",
		"version":   "1.0.0",
		"messages":  len(messages),
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
		"endpoints": map[string]string{
			"GET /api/chat/messages": "获取聊天消息",
			"POST /api/chat/send":    "发送新消息",
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func main() {
	// 设置路由
	http.HandleFunc("/api/chat/messages", enableCORS(getMessagesHandler))
	http.HandleFunc("/api/chat/send", enableCORS(sendMessageHandler))
	http.HandleFunc("/", enableCORS(healthHandler))
	
	port := ":8000"
	
	fmt.Println("🚀 家庭聊天服务器启动成功!")
	fmt.Println("📍 服务地址: http://localhost" + port)
	fmt.Println("💬 聊天端点: http://localhost" + port + "/api/chat/messages")
	fmt.Println("📨 发送消息: POST http://localhost" + port + "/api/chat/send")
	fmt.Println("📊 初始消息数:", len(messages))
	fmt.Println("⏰ 启动时间:", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println("==================================================")
	
	log.Fatal(http.ListenAndServe(port, nil))
}