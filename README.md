\# 🏠 Family Drive - 家庭网盘 \& 实时聊天系统



一个功能完整的家庭私有网盘系统，支持文件管理、安全分享和实时聊天。



!\[GitHub](https://img.shields.io/badge/Go-1.21+-blue)

!\[GitHub](https://img.shields.io/badge/React-18-61dafb)

!\[GitHub](https://img.shields.io/badge/Tauri-Rust-orange)

!\[GitHub](https://badgen.net/badge/license/MIT/blue)



\## ✨ 功能特性



\### 📁 文件管理

\- ✅ 文件上传、下载、删除

\- ✅ 文件列表浏览

\- ✅ 私有网盘模式

\- ✅ 文件分享链接生成

\- ✅ 密码保护分享

\- ✅ 分享链接过期设置



\### 🔐 安全认证

\- ✅ JWT 令牌认证

\- ✅ 用户注册登录

\- ✅ bcrypt 密码加密

\- ✅ HTTPS 安全传输

\- ✅ CORS 跨域保护



\### 💬 实时聊天

\- ✅ WebSocket 实时通信

\- ✅ 多用户消息同步

\- ✅ 语音消息支持

\- ✅ 消息历史记录



\## 🏗️ 技术架构



\### 后端 (Go)

```go

Gin + GORM + JWT + MySQL + WebSocket

```



\### 前端 (React + Tauri)

```javascript

React + TypeScript + Tauri + Axios + WebSocket

```



\## 🚀 快速开始



\### 环境要求

\- Go 1.21+

\- Node.js 18+

\- MySQL 8.0+



\### 后端部署



1\. \*\*克隆项目\*\*

```bash

git clone https://github.com/uddqerw/family-drive.git

cd family-drive/backend

```



2\. \*\*数据库配置\*\*

```sql

CREATE DATABASE family\_drive;

```



3\. \*\*环境配置\*\*

```bash

\# 复制并修改配置

cp config.example.env .env

```



4\. \*\*安装依赖 \& 运行\*\*

```bash

go mod tidy

go run cmd/server/main.go

```



\### 前端部署



1\. \*\*安装依赖\*\*

```bash

cd frontend

npm install

```



2\. \*\*开发模式运行\*\*

```bash

\# 前端开发服务器

npm run dev



\# Tauri 桌面应用

npm run tauri dev

```



\## 📁 项目结构



```

family-drive/

├── backend/                 # Go 后端服务

│   ├── cmd/server/         # 应用入口

│   ├── internal/           # 内部核心逻辑

│   │   ├── auth/          # JWT 认证

│   │   ├── db/            # 数据库操作

│   │   └── models/        # 数据模型

│   ├── middleware/         # 中间件层

│   ├── handlers/          # HTTP 处理器

│   └── websocket/         # WebSocket 服务

├── frontend/               # React 前端

│   ├── src/

│   │   ├── components/    # React 组件

│   │   ├── services/      # API 服务

│   │   └── hooks/         # 自定义 Hooks

│   └── src-tauri/         # Tauri 配置

└── README.md

```



\## 🔧 配置说明



\### 数据库表结构

```sql

CREATE TABLE users (

&nbsp;   id INT AUTO\_INCREMENT PRIMARY KEY,

&nbsp;   username VARCHAR(50) UNIQUE NOT NULL,

&nbsp;   email VARCHAR(100) UNIQUE NOT NULL,

&nbsp;   password\_hash VARCHAR(255) NOT NULL,

&nbsp;   created\_at TIMESTAMP DEFAULT CURRENT\_TIMESTAMP,

&nbsp;   updated\_at TIMESTAMP DEFAULT CURRENT\_TIMESTAMP ON UPDATE CURRENT\_TIMESTAMP

);

```



\### 默认测试账户

```

邮箱: test@example.com

密码: 123456

```



\## 🌐 API 文档



\### 认证接口

\- `POST /api/auth/login` - 用户登录

\- `POST /api/auth/register` - 用户注册

\- `GET /api/auth/me` - 获取当前用户



\### 文件接口

\- `POST /api/files/upload` - 文件上传

\- `GET /api/files/list` - 文件列表

\- `GET /api/files/download/:filename` - 文件下载

\- `POST /api/files/share/:filename` - 创建分享



\### 聊天接口

\- `GET /api/chat/messages` - 获取消息

\- `POST /api/chat/send` - 发送消息

\- `GET /ws` - WebSocket 连接



\## 🛡️ 安全特性



\- 🔒 JWT 令牌认证

\- 🔐 bcrypt 密码哈希

\- 🌐 HTTPS 加密传输

\- 🛡️ CORS 安全策略

\- 🗂️ 文件访问权限控制



\## 📈 性能优化



\- ⚡ WebSocket 实时通信

\- 🗃️ 数据库连接池

\- 🔄 前端请求缓存

\- 📦 文件分块上传（规划中）



\## 🤝 贡献指南



欢迎提交 Issue 和 Pull Request！



\## 📄 许可证

MIT License

---

⭐ 如果这个项目对你有帮助，请给它一个 Star！

