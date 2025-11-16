package handlers

import (
	"encoding/json"
	"family-drive/backend/models"
	"family-drive/backend/websocket"
	"log"
	"net/http"
	"time"
)

func HandleWebSocket(hub *websocket.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		websocket.ServeWebSocket(hub, w, r)
	}
}

func HandleGetMessages(w http.ResponseWriter, r *http.Request) {
	// 获取聊天消息历史
	messages := []models.Message{
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    messages,
	})
}