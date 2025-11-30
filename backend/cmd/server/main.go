package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"familydrive/handlers"
	"familydrive/internal/auth"

	// "familydrive/middleware"
	"familydrive/websocket"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 用户信息结构体 - 匹配数据库表结构
type User struct {
	ID           int       `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"unique" json:"username"`
	Email        string    `gorm:"unique" json:"email"`
	PasswordHash string    `gorm:"column:password_hash" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}

// 文件信息结构体
type FileInfo struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	Type       string `json:"type"`
	UploadTime string `json:"uploadTime"`
	IsHidden   bool   `json:"isHidden"`
}

// 分享记录结构体
type ShareRecord struct {
	Token       string    `json:"token"`
	Filename    string    `json:"filename"`
	Password    string    `json:"password"` // 加密后的密码
	ExpireTime  time.Time `json:"expireTime"`
	MaxAccess   int       `json:"maxAccess"`
	AccessCount int       `json:"accessCount"`
	CreatedAt   time.Time `json:"createdAt"`
}

var (
	uploadDir     = "./uploads"
	shareRecords  = make(map[string]ShareRecord) // 内存存储分享记录
	files         []FileInfo
	fileIDCounter = 1
	db            *gorm.DB
)

// ==================== 数据库初始化 ====================

func initDB() {
	// 替换为你的MySQL连接信息
	dsn := "root:Root@20160212@tcp(localhost:3306)/family_drive?charset=utf8mb4&parseTime=True&loc=Local"
	var err error
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("MySQL连接失败: " + err.Error())
	}
	fmt.Println("✅ MySQL连接成功")

	// 自动迁移表结构
	err = db.AutoMigrate(&User{})
	if err != nil {
		fmt.Println("⚠️  表迁移警告:", err)
	}
}

// ==================== 认证处理器 ====================

// 登录处理器 - 使用你的JWT中间件
func handleLogin(c *gin.Context) {
	var request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.BindJSON(&request); err != nil {
		c.JSON(400, gin.H{"error": "无效请求"})
		return
	}

	fmt.Printf("🔐 登录尝试: %s\n", request.Email)

	// ✅ 查询真实用户
	var user User
	result := db.Where("email = ?", request.Email).First(&user)
	if result.Error != nil {
		c.JSON(401, gin.H{"error": "邮箱不存在"})
		return
	}

	// ✅ 验证密码（bcrypt加密验证）
	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password))
	if err != nil {
		c.JSON(401, gin.H{"error": "密码错误"})
		return
	}

	fmt.Printf("✅ 登录成功: %s (ID: %d)\n", user.Username, user.ID)

	// 使用你的JWT中间件生成token
	token, err := auth.GenerateUserToken(user.ID, user.Username, user.Email, 24*time.Hour)
	if err != nil {
		c.JSON(500, gin.H{"error": "生成token失败: " + err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"success": true,
		"message": "登录成功",
		"data": gin.H{
			"user": gin.H{
				"id":       user.ID,
				"username": user.Username, // ✅ 真实用户名！
				"email":    user.Email,
			},
			"access_token": token,
		},
	})
}

// 注册处理器
func handleRegister(c *gin.Context) {
	var request struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.BindJSON(&request); err != nil {
		c.JSON(400, gin.H{"error": "无效请求"})
		return
	}

	fmt.Printf("👤 注册用户: %s (%s)\n", request.Username, request.Email)

	// ✅ 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(500, gin.H{"error": "密码加密失败"})
		return
	}

	// ✅ 创建真实用户
	user := User{
		Username:     request.Username,
		Email:        request.Email,
		PasswordHash: string(hashedPassword),
	}

	result := db.Create(&user)
	if result.Error != nil {
		c.JSON(500, gin.H{"error": "用户已存在或创建失败"})
		return
	}

	fmt.Printf("✅ 注册成功: %s (%s) ID: %d\n", user.Username, user.Email, user.ID)

	c.JSON(200, gin.H{
		"success": true,
		"message": "注册成功",
		"data": gin.H{
			"user": gin.H{
				"id":       user.ID,
				"username": user.Username,
				"email":    user.Email,
			},
		},
	})
}

// 获取当前用户 - 使用JWT中间件
func handleGetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(401, gin.H{"error": "未认证"})
		return
	}

	username, _ := c.Get("username")
	email, _ := c.Get("email")

	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"id":       userID,
			"username": username,
			"email":    email,
		},
	})
}

// 登出处理器
func handleLogout(c *gin.Context) {
	fmt.Println("🚪 用户登出")
	c.JSON(200, gin.H{
		"success": true,
		"message": "登出成功",
	})
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:3001")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// ==================== 主函数 ====================
func main() {
	// 初始化数据库连接
	initDB()

	router := gin.Default()

	// 设置信任的代理 - 生产环境用通配符
	router.SetTrustedProxies([]string{"*"})

	// 使用CORS中间件
	// router.Use(wrapMiddleware(middleware.CORS))
	router.Use(CORSMiddleware())

	// 创建 WebSocket Hub
	hub := websocket.NewHub()
	go hub.Run()

	// ==================== 路由注册 ====================

	// 公开路由 - 不需要认证
	public := router.Group("/api")
	{
		// 认证路由
		public.POST("/auth/login", handleLogin)
		public.POST("/auth/register", handleRegister)
		// 健康检查路由
		public.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok", "message": "服务器运行正常"})
		})
		// 添加根路径路由
		public.GET("/", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"service":   "家庭网盘后端",
				"status":    "running",
				"timestamp": time.Now().Unix(),
			})
		})
		// 分享链接访问（公开）
		public.GET("/s/:token", accessSharedFile)
	}

	// 受保护路由 - 需要认证
	protected := router.Group("/api")
	protected.Use(GinAuthMiddleware()) // 使用新的 Gin 认证中间件
	{
		// 用户相关
		protected.GET("/auth/me", handleGetCurrentUser)
		protected.POST("/auth/logout", handleLogout)

		// 文件管理
		protected.POST("/files/upload", uploadFile)
		protected.GET("/files/list", listFiles)
		protected.GET("/files/download/:filename", downloadFile)
		protected.POST("/files/secure-download/:filename", secureDownloadFile)
		protected.DELETE("/files/delete/:filename", deleteFile)
		protected.POST("/files/share/:filename", createShare)

		// 聊天功能
		protected.GET("/chat/messages", gin.WrapH(http.HandlerFunc(handlers.HandleGetMessages)))
		protected.POST("/chat/send", gin.WrapH(handlers.HandleChatSend(hub)))
		protected.POST("/chat/voice", gin.WrapH(http.HandlerFunc(handlers.HandleVoiceMessage)))
		protected.POST("/chat/clear", gin.WrapH(http.HandlerFunc(handlers.HandleClearMessages)))
		protected.GET("/ws", gin.WrapH(handlers.HandleWebSocket(hub)))
	}

	fmt.Println("🚀 文件服务器启动在 https://localhost:8000")
	fmt.Println("🔒 安全模式：JWT认证 + 密码验证 + 分享链接保护")
	fmt.Println("💬 聊天功能：WebSocket 实时聊天已启用")
	fmt.Println("👤 认证系统：MySQL + JWT 完整用户验证")
	fmt.Println("🌐 CORS：已启用中间件")

	// 使用证书文件
	if err := router.RunTLS(":8000", "localhost.crt", "localhost.key"); err != nil {
		log.Fatal("启动服务器失败:", err)
	}
}

// ==================== 文件处理函数 ====================
// 以下是你调试好的文件处理代码，完全保持不变！

// 加密密码
func hashPassword(password string) string {
	if password == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// 验证密码
func verifyPassword(inputPassword, storedHash string) bool {
	if inputPassword == "" && storedHash == "" {
		return true
	}
	return hashPassword(inputPassword) == storedHash
}

// 检查分享是否过期
func isShareExpired(share ShareRecord) bool {
	return time.Now().After(share.ExpireTime)
}

// 检查访问次数是否超限
func isAccessExceeded(share ShareRecord) bool {
	return share.MaxAccess > 0 && share.AccessCount >= share.MaxAccess
}

// 上传文件
func uploadFile(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "获取文件失败"})
		return
	}
	defer file.Close()

	// 获取是否隐藏文件（默认true - 私有网盘模式）
	isHidden := c.Request.FormValue("is_hidden") != "false"

	// 创建目标文件
	filename := header.Filename
	filePath := filepath.Join(uploadDir, filename)
	dst, err := os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建文件失败"})
		return
	}
	defer dst.Close()

	// 复制文件内容
	if _, err := io.Copy(dst, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败"})
		return
	}

	// 保存文件信息
	fileInfo := FileInfo{
		ID:         fileIDCounter,
		Name:       filename,
		Size:       header.Size,
		Type:       header.Header.Get("Content-Type"),
		UploadTime: time.Now().Format(time.RFC3339),
		IsHidden:   isHidden,
	}
	files = append(files, fileInfo)
	fileIDCounter++

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    fileInfo,
		"message": "文件上传成功",
	})
}

// 文件列表 - 修复：确保返回数组格式
func listFiles(c *gin.Context) {
	// 如果 files 是 nil，返回空数组而不是 null
	if files == nil {
		c.JSON(http.StatusOK, []FileInfo{})
		return
	}

	// 返回所有文件（主人视图 - 私有网盘模式）
	c.JSON(http.StatusOK, files)
}

// 在 main.go 中添加这个函数
func GinAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 Header 获取 token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "未提供认证token"})
			c.Abort()
			return
		}

		// 提取 Bearer token
		tokenString := authHeader
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenString = authHeader[7:]
		}

		// 验证并解析 token
		claims, err := auth.ParseUserToken(tokenString)
		if err != nil {
			c.JSON(401, gin.H{"error": "token无效或已过期"})
			c.Abort()
			return
		}

		// 将用户信息存入 Gin 上下文
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)

		fmt.Printf("🔐 用户认证: %s (ID: %d)\n", claims.Username, claims.UserID)
		c.Next()
	}
}

// 文件下载
func downloadFile(c *gin.Context) {
	filename := c.Param("filename")
	filePath := filepath.Join(uploadDir, filename)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	// 提供文件下载
	c.File(filePath)
}

// 安全下载文件（需要密码验证）
func secureDownloadFile(c *gin.Context) {
	filename := c.Param("filename")

	var request struct {
		Password   string `json:"password"`
		ShareToken string `json:"share_token"`
	}

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效请求"})
		return
	}

	// 查找分享记录
	share, exists := shareRecords[request.ShareToken]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "分享链接不存在或已失效"})
		return
	}

	// 验证文件名匹配
	if share.Filename != filename {
		c.JSON(http.StatusForbidden, gin.H{"error": "文件不匹配"})
		return
	}

	// 检查是否过期
	if isShareExpired(share) {
		delete(shareRecords, request.ShareToken)
		c.JSON(http.StatusGone, gin.H{"error": "分享链接已过期"})
		return
	}

	// 检查访问次数
	if isAccessExceeded(share) {
		delete(shareRecords, request.ShareToken)
		c.JSON(http.StatusGone, gin.H{"error": "分享链接访问次数已用完"})
		return
	}

	// 验证密码
	if !verifyPassword(request.Password, share.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}

	// 文件路径
	filePath := filepath.Join(uploadDir, filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	// 更新访问次数
	share.AccessCount++
	shareRecords[request.ShareToken] = share

	// 提供文件下载
	c.File(filePath)
}

// 删除文件
func deleteFile(c *gin.Context) {
	filename := c.Param("filename")
	filePath := filepath.Join(uploadDir, filename)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	// 删除文件
	if err := os.Remove(filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除文件失败"})
		return
	}

	// 从文件列表中移除
	for i, file := range files {
		if file.Name == filename {
			files = append(files[:i], files[i+1:]...)
			break
		}
	}

	// 删除相关的分享记录
	for token, share := range shareRecords {
		if share.Filename == filename {
			delete(shareRecords, token)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "文件删除成功",
	})
}

// 创建分享链接
func createShare(c *gin.Context) {
	filename := c.Param("filename")

	var request struct {
		ExpireHours int    `json:"expire_hours"`
		MaxAccess   int    `json:"max_access"`
		Password    string `json:"password"`
		UserID      int    `json:"user_id"`
	}

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效请求"})
		return
	}

	// 检查文件是否存在
	filePath := filepath.Join(uploadDir, filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	// 生成唯一 token
	token := uuid.New().String()[:8]
	expireTime := time.Now().Add(time.Duration(request.ExpireHours) * time.Hour)

	// 创建分享记录
	shareRecord := ShareRecord{
		Token:       token,
		Filename:    filename,
		Password:    hashPassword(request.Password),
		ExpireTime:  expireTime,
		MaxAccess:   request.MaxAccess,
		AccessCount: 0,
		CreatedAt:   time.Now(),
	}

	// 保存分享记录
	shareRecords[token] = shareRecord

	// 构建分享链接
	shareURL := fmt.Sprintf("https://localhost:8000/api/s/%s", token)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"share_url":    shareURL,
			"expire_time":  expireTime.Format(time.RFC3339),
			"max_access":   request.MaxAccess,
			"has_password": request.Password != "",
			"token":        token,
		},
		"message": "分享链接创建成功",
	})
}

// 通过分享链接访问文件
func accessSharedFile(c *gin.Context) {
	token := c.Param("token")

	// 查找分享记录
	share, exists := shareRecords[token]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "分享链接不存在或已失效"})
		return
	}

	// 检查是否过期
	if isShareExpired(share) {
		delete(shareRecords, token)
		c.JSON(http.StatusGone, gin.H{"error": "分享链接已过期"})
		return
	}

	// 检查访问次数
	if isAccessExceeded(share) {
		delete(shareRecords, token)
		c.JSON(http.StatusGone, gin.H{"error": "分享链接访问次数已用完"})
		return
	}

	// 文件路径
	filePath := filepath.Join(uploadDir, share.Filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	// 如果有密码，返回需要密码的页面
	if share.Password != "" {
		html := `
		<!DOCTYPE html>
		<html>
		<head>
			<title>下载受密码保护的文件</title>
			<style>
				body { 
					font-family: Arial, sans-serif; 
					max-width: 500px; 
					margin: 100px auto; 
					padding: 20px;
					background: #f5f5f5;
				}
				.container {
					background: white;
					padding: 30px;
					border-radius: 10px;
					box-shadow: 0 2px 10px rgba(0,0,0,0.1);
				}
				.form-group { margin-bottom: 20px; }
				label { display: block; margin-bottom: 8px; font-weight: bold; color: #333; }
				input { 
					width: 100%; 
					padding: 12px; 
					border: 1px solid #ddd; 
					border-radius: 6px;
					font-size: 16px;
					box-sizing: border-box;
				}
				button { 
					background: #1890ff; 
					color: white; 
					padding: 12px 24px; 
					border: none; 
					border-radius: 6px; 
					cursor: pointer;
					font-size: 16px;
					width: 100%;
				}
				button:hover { background: #40a9ff; }
				.error { 
					color: #ff4d4f; 
					margin-top: 10px; 
					padding: 10px;
					background: #fff2f0;
					border: 1px solid #ffccc7;
					border-radius: 6px;
					display: none;
				}
				.success {
					color: #52c41a;
					margin-top: 10px;
					padding: 10px;
					background: #f6ffed;
					border: 1px solid #b7eb8f;
					border-radius: 6px;
					display: none;
				}
				.file-info {
					background: #f0f8ff;
					padding: 15px;
					border-radius: 6px;
					margin-bottom: 20px;
					border-left: 4px solid #1890ff;
				}
			</style>
		</head>
		<body>
			<div class="container">
				<h2>🔒 受密码保护的文件下载</h2>
				
				<div class="file-info">
					<strong>文件名称:</strong> ` + share.Filename + `<br>
					<strong>剩余访问次数:</strong> ` + strconv.Itoa(share.MaxAccess-share.AccessCount) + `/` + strconv.Itoa(share.MaxAccess) + `<br>
					<strong>链接过期时间:</strong> ` + share.ExpireTime.Format("2006-01-02 15:04:05") + `
				</div>
				
				<form id="downloadForm">
					<div class="form-group">
						<label for="password">访问密码:</label>
						<input type="password" id="password" name="password" required placeholder="请输入访问密码">
					</div>
					<button type="submit">🔐 验证并下载</button>
				</form>
				
				<div id="error" class="error"></div>
				<div id="success" class="success"></div>
			</div>
			
			<script>
				document.getElementById('downloadForm').addEventListener('submit', async function(e) {
					e.preventDefault();
					const password = document.getElementById('password').value;
					const errorDiv = document.getElementById('error');
					const successDiv = document.getElementById('success');
					
					// 隐藏消息
					errorDiv.style.display = 'none';
					successDiv.style.display = 'none';
					
					if (!password) {
						errorDiv.textContent = '请输入访问密码';
						errorDiv.style.display = 'block';
						return;
					}
					
					try {
						const response = await fetch('/api/files/secure-download/` + share.Filename + `', {
							method: 'POST',
							headers: { 'Content-Type': 'application/json' },
							body: JSON.stringify({
								password: password,
								share_token: '` + token + `'
							})
						});
						
						if (response.ok) {
							const blob = await response.blob();
							const url = window.URL.createObjectURL(blob);
							const a = document.createElement('a');
							a.href = url;
							a.download = '` + share.Filename + `';
							document.body.appendChild(a);
							a.click();
							document.body.removeChild(a);
							window.URL.revokeObjectURL(url);
							
							successDiv.textContent = '✅ 下载成功！文件已开始下载';
							successDiv.style.display = 'block';
						} else {
							const errorData = await response.json();
							errorDiv.textContent = '❌ ' + (errorData.error || '下载失败');
							errorDiv.style.display = 'block';
						}
					} catch (err) {
						errorDiv.textContent = '❌ 网络错误，请重试';
						errorDiv.style.display = 'block';
					}
				});
			</script>
		</body>
		</html>
		`
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
		return
	}

	// 如果没有密码，直接下载并更新访问次数
	share.AccessCount++
	shareRecords[token] = share
	c.File(filePath)
}
