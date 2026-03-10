package postgresDB

import (
	"database/sql"
	"fmt"
	"log"
	"os"
)

func InitDB() *sql.DB {
	connStr := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Ошибка подключения к базе данных: ", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal("База не ответила на пинг: ", err)
	}

	fmt.Println("Бот подключен к базе данных!")
	return db
}

func CreateTable(db *sql.DB) {
	query := `
	CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    track_code TEXT UNIQUE NOT NULL,
    status TEXT NOT NULL DEFAULT 'В обработке',
    user_email TEXT,
    tg_chat_id BIGINT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );	`
	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Не удалось создать/найти таблицу:", err)
	}
}

func GetOrderStatus(db *sql.DB, tracker string) (string, error) {
	var status string
	query := `SELECT status FROM orders WHERE track_code = $1`
	err := db.QueryRow(query, tracker).Scan(&status)
	return status, err
}

func RegisterUser(db *sql.DB, tracker string, userID int64) {
	query := `UPDATE orders SET tg_chat_id = $2, updated_at = CURRENT_TIMESTAMP WHERE track_code = $1`
	_, err := db.Exec(query)
	if err != nil {
		log.Println("Не удалось добавить пользователя:", err)
	}
}
