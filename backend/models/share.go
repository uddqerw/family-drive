package models

import (
	"time"
)

// ShareLink 文件分享链接
type ShareLink struct {
	ID          string    `json:"id"`
	Filename    string    `json:"filename"`
	Filepath    string    `json:"filepath"`
	ShareURL    string    `json:"share_url"`
	CreatedBy   int       `json:"created_by"`   // 创建者用户ID
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	Password    string    `json:"password,omitempty"` // 可选密码
	AccessCount int       `json:"access_count"`
	MaxAccess   int       `json:"max_access"` // 最大访问次数，0表示无限制
	IsActive    bool      `json:"is_active"`
}

// ShareLinkResponse 分享链接响应
type ShareLinkResponse struct {
	ID          string    `json:"id"`
	Filename    string    `json:"filename"`
	ShareURL    string    `json:"share_url"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	AccessCount int       `json:"access_count"`
	MaxAccess   int       `json:"max_access"`
	IsProtected bool      `json:"is_protected"` // 是否有密码保护
}