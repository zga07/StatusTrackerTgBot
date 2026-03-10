package postgresDB

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"
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
	    description TEXT,
	    status TEXT NOT NULL DEFAULT 'В обработке',
	    tg_chat_id BIGINT,
	    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

    CREATE TABLE IF NOT EXISTS order_history (
        id SERIAL PRIMARY KEY,
        track_code TEXT NOT NULL,
        status TEXT NOT NULL,
        changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );`
	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Не удалось создать/найти таблицу:", err)
	}
}

func RegisterUser(db *sql.DB, tracker string, userID int64) {
	query := `UPDATE orders SET tg_chat_id = $2, updated_at = CURRENT_TIMESTAMP WHERE track_code = $1`
	_, err := db.Exec(query, tracker, userID)
	if err != nil {
		log.Println("Не удалось добавить пользователя:", err)
	}
}

func AddOrder(db *sql.DB, trackCode, description string) error {
	query := `INSERT INTO orders (track_code, description) VALUES ($1, $2)`
	_, err := db.Exec(query, trackCode, description)
	if err == nil {
		addHistoryRecord(db, trackCode, "В обработке")
	}
	return err
}

func addHistoryRecord(db *sql.DB, trackCode, status string) {
	query := `INSERT INTO order_history (track_code, status) VALUES ($1, $2)`
	_, err := db.Exec(query, trackCode, status)
	if err != nil {
		log.Println("Ошибка записи истории:", err)
	}
}

func GetOrderStatus(db *sql.DB, tracker string) (string, error) {
	var status string
	query := `SELECT status FROM orders WHERE track_code = $1`
	err := db.QueryRow(query, tracker).Scan(&status)
	return status, err
}

func GetOrderHistory(db *sql.DB, trackCode string) ([]string, error) {
	query := `SELECT status, changed_at FROM order_history
              WHERE track_code = $1 ORDER BY changed_at ASC`
	rows, err := db.Query(query, trackCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []string
	for rows.Next() {
		var status string
		var changedAt time.Time
		if err := rows.Scan(&status, &changedAt); err != nil {
			return nil, err
		}
		history = append(history, fmt.Sprintf("%s — %s", changedAt.Format("02.01.2006 15:04"), status))
	}
	return history, nil
}

func UpdateOrderStatus(db *sql.DB, trackCode, newStatus string) (int64, error) {
	var chatID sql.NullInt64
	query := `UPDATE orders SET status = $2, updated_at = CURRENT_TIMESTAMP
              WHERE track_code = $1 RETURNING tg_chat_id`

	err := db.QueryRow(query, trackCode, newStatus).Scan(&chatID)
	if err != nil {
		return 0, err
	}

	addHistoryRecord(db, trackCode, newStatus)
	return chatID.Int64, nil
}
