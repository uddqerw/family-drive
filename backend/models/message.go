package models

import (
	"time"
)

type Message struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	Type      string    `json:"type"` // text, image, file
	Room      string    `json:"room"` // general, private_{userid}
	CreatedAt time.Time `json:"created_at"`
}

type ChatRequest struct {
	Action   string `json:"action"`   // send_message, join_room, leave_room
	Room     string `json:"room"`     // 房间名
	Content  string `json:"content"`  // 消息内容
	Username string `json:"username"` // 用户名
}