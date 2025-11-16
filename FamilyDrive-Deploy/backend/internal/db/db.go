package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

var conn *sql.DB

func Init(path string) error {
	// sqlite DSN: file:path?_foreign_keys=1
	dsn := fmt.Sprintf("file:%s?_foreign_keys=1", path)
	var err error
	conn, err = sql.Open("sqlite", dsn)
	if err != nil {
		return err
	}
	// set sensible limits
	conn.SetMaxOpenConns(1)
	// migrate: create tables
	return migrate()
}

func Close() {
	if conn != nil {
		_ = conn.Close()
	}
}

func DB() *sql.DB {
	return conn
}

func migrate() error {
	s := `
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    token TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL,
    expires_at DATETIME NOT NULL,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);`
	_, err := conn.Exec(s)
	return err
}
