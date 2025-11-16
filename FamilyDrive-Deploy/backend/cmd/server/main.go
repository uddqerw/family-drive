package main

import (
	"log"
	"net/http"
	"os"

	"familydrive/internal/db"
	"familydrive/internal/handlers"
)

// CORS中间件
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

func main() {
	// DB初始化
	dbPath := "./family.db"
	if e := os.Getenv("FAMILYDRIVE_DB"); e != "" {
		dbPath = e
	}

	if err := db.Init(dbPath); err != nil {
		log.Fatalf("db.Init: %v", err)
	}
	defer db.Close()

	mux := http.NewServeMux()

	// 健康检查
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"family-drive"}`))
	})

	// API路由（全部包装CORS中间件）
	mux.HandleFunc("/api/auth/register", corsMiddleware(handlers.HandleRegister))
	mux.HandleFunc("/api/auth/login", corsMiddleware(handlers.HandleLogin))
	mux.HandleFunc("/api/auth/refresh", corsMiddleware(handlers.HandleRefresh))
	mux.HandleFunc("/api/auth/me", corsMiddleware(handlers.HandleMe))
	mux.HandleFunc("/api/auth/logout", corsMiddleware(handlers.HandleLogout))
	mux.HandleFunc("/api/files/upload", corsMiddleware(handlers.HandleFileUpload))
	mux.HandleFunc("/api/files/list", corsMiddleware(handlers.HandleFileList))
	mux.HandleFunc("/api/files/download/", corsMiddleware(handlers.HandleFileDownload))
	mux.HandleFunc("/api/files/delete/", corsMiddleware(handlers.HandleFileDelete))

	addr := "0.0.0.0:8000"
	if v := os.Getenv("FAMILYDRIVE_ADDR"); v != "" {
		addr = v
	}
	log.Printf("server starting at %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("ListenAndServe: %v", err)
	}
}