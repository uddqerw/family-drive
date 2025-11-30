// handlers/chat.go
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"familydrive/models"
	"familydrive/websocket"
)

// 添加全局消息存储
var (
	chatMessages   []models.Message
	chatMutex      sync.RWMutex
	messageCounter = 1000
)

// 初始化一些消息
func init() {
	chatMessages = []models.Message{
		{
			ID:        1,
			UserID:    1,
			Username:  "系统消息",
			Content:   "🎉 欢迎来到家庭聊天室！",
			Type:      "text",
			Room:      "general",
			CreatedAt: time.Now(),
		},
		{
			ID:        2,
			UserID:    2,
			Username:  "家庭助手",
			Content:   "💬 这是一个家庭专用的聊天室，可以在这里分享文件和交流",
			Type:      "text",
			Room:      "general",
			CreatedAt: time.Now().Add(-time.Minute * 5),
		},
	}
	messageCounter = 3
	
	fmt.Printf("💾 聊天系统初始化完成，初始消息数: %d\n", len(chatMessages))
}

func HandleGetMessages(w http.ResponseWriter, r *http.Request) {
	chatMutex.RLock()
	defer chatMutex.RUnlock()
	
	// 添加调试日志
	fmt.Printf("📨 [%s] 处理消息请求，返回 %d 条消息\n", 
		time.Now().Format("15:04:05"), len(chatMessages))
	
	for i, msg := range chatMessages {
		fmt.Printf("   %d. [%s] %s: %s\n", 
			i+1, msg.CreatedAt.Format("15:04:05"), msg.Username, msg.Content)
	}
	
	// 转换为前端期望的格式
	formattedMessages := make([]map[string]interface{}, len(chatMessages))
	for i, msg := range chatMessages {
		formattedMessages[i] = map[string]interface{}{
			"id":        msg.ID,
			"user_id":   msg.UserID,
			"username":  msg.Username,
			"content":   msg.Content,
			"type":      "user",
			"timestamp": msg.CreatedAt.Format(time.RFC3339),
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    formattedMessages,
	})
}

func HandleChatSend(hub *websocket.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Username string `json:"username"`
			Content  string `json:"content"`
			UserID   int    `json:"user_id"`
		}
		
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "无效请求", http.StatusBadRequest)
			return
		}
		
		if request.Content == "" {
			http.Error(w, "消息内容不能为空", http.StatusBadRequest)
			return
		}
		
		// 创建新消息
		newMessage := models.Message{
			ID:        messageCounter,
			UserID:    request.UserID,
			Username:  request.Username,
			Content:   request.Content,
			Type:      "text",
			Room:      "general",
			CreatedAt: time.Now(),
		}
		messageCounter++
		
		// 存储消息
		chatMutex.Lock()
		chatMessages = append(chatMessages, newMessage)
		chatMutex.Unlock()
		
		// 添加调试日志
		fmt.Printf("💾 [%s] 消息已存储，当前总数: %d\n", 
			time.Now().Format("15:04:05"), len(chatMessages))
		fmt.Printf("📝 最新消息: %s - %s\n", newMessage.Username, newMessage.Content)
		
		// 通过 WebSocket 广播消息
		messageData := map[string]interface{}{
			"type":      "chat_message",
			"id":        newMessage.ID,
			"user_id":   newMessage.UserID,
			"username":  newMessage.Username,
			"content":   newMessage.Content,
			"timestamp": newMessage.CreatedAt.Format(time.RFC3339),
		}
		messageBytes, _ := json.Marshal(messageData)
		
		fmt.Printf("📢 准备广播消息到 WebSocket\n")
		hub.Broadcast(messageBytes)
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "消息发送成功",
			"data":    newMessage,
		})
	}
}

// 发送语音消息
func HandleVoiceMessage(w http.ResponseWriter, r *http.Request) {
	// 解析表单
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB
		http.Error(w, "文件太大", http.StatusBadRequest)
		return
	}
	
	username := r.FormValue("username")
	duration := r.FormValue("duration")
	
	// 这里可以处理语音文件上传
	fmt.Printf("🎤 语音消息: %s - %s秒\n", username, duration)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "语音消息接收成功",
		"duration": duration,
	})
}

// 清空消息
func HandleClearMessages(w http.ResponseWriter, r *http.Request) {
	chatMutex.Lock()
	defer chatMutex.Unlock()
	
	// 保留系统消息
	systemMessages := []models.Message{}
	for _, msg := range chatMessages {
		if msg.Username == "系统消息" || msg.Username == "家庭助手" {
			systemMessages = append(systemMessages, msg)
		}
	}
	chatMessages = systemMessages
	
	fmt.Printf("🗑️ [%s] 清空聊天消息，保留 %d 条系统消息\n", 
		time.Now().Format("15:04:05"), len(chatMessages))
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "消息清空成功",
	})
}

// WebSocket 处理器
func HandleWebSocket(hub *websocket.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("🔌 [%s] WebSocket 连接请求\n", time.Now().Format("15:04:05"))
		websocket.ServeWebSocket(hub, w, r)
	}
}