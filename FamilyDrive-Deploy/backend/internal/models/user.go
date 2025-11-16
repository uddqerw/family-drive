package models

import (
	"database/sql"
	"time"

	"familydrive/internal/db"
)

type User struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

func CreateUser(name, email, passwordHash string) (*User, error) {
	res, err := db.DB().Exec("INSERT INTO users(name,email,password_hash) VALUES(?,?,?)", name, email, passwordHash)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return GetUserByID(id)
}

func GetUserByEmail(email string) (*User, error) {
	row := db.DB().QueryRow("SELECT id,name,email,password_hash,created_at FROM users WHERE email = ?", email)
	var u User
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func GetUserByID(id int64) (*User, error) {
	row := db.DB().QueryRow("SELECT id,name,email,password_hash,created_at FROM users WHERE id = ?", id)
	var u User
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func StoreRefreshToken(token string, userID int64, expiresAt time.Time) error {
	_, err := db.DB().Exec("INSERT INTO refresh_tokens(token,user_id,expires_at) VALUES(?,?,?)", token, userID, expiresAt)
	return err
}

func GetRefreshTokenOwner(token string) (int64, time.Time, error) {
	row := db.DB().QueryRow("SELECT user_id,expires_at FROM refresh_tokens WHERE token = ?", token)
	var uid int64
	var expStr string
	err := row.Scan(&uid, &expStr)
	if err == sql.ErrNoRows {
		return 0, time.Time{}, sql.ErrNoRows
	}
	if err != nil {
		return 0, time.Time{}, err
	}
	exp, err := time.Parse(time.RFC3339, expStr)
	if err != nil {
		return 0, time.Time{}, err
	}
	return uid, exp, nil
}

func RevokeRefreshToken(token string) error {
	_, err := db.DB().Exec("DELETE FROM refresh_tokens WHERE token = ?", token)
	return err
}
