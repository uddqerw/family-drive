package main

import (
	"log"
	"net/http"
	"os"

	"familydrive/internal/db"
	"familydrive/internal/handlers"
)

func main() {
	// DB file path (current dir)
	dbPath := "./family.db"
	if e := os.Getenv("FAMILYDRIVE_DB"); e != "" {
		dbPath = e
	}

	if err := db.Init(dbPath); err != nil {
		log.Fatalf("db.Init: %v", err)
	}
	defer db.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/register", handlers.HandleRegister)
	mux.HandleFunc("/api/auth/login", handlers.HandleLogin)
	mux.HandleFunc("/api/auth/refresh", handlers.HandleRefresh)
	mux.HandleFunc("/api/auth/me", handlers.HandleMe)
	mux.HandleFunc("/api/auth/logout", handlers.HandleLogout)
        mux.HandleFunc("/api/files/upload", handlers.HandleFileUpload)
        mux.HandleFunc("/api/files/list", handlers.HandleFileList)
        mux.HandleFunc("/api/files/download/", handlers.HandleFileDownload)
        mux.HandleFunc("/api/files/delete/", handlers.HandleFileDelete)

	addr := ":8000"
	if v := os.Getenv("FAMILYDRIVE_ADDR"); v != "" {
		addr = v
	}
	log.Printf("server starting at %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("ListenAndServe: %v", err)
	}
}
