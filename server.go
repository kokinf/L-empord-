package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	_ "github.com/lib/pq" // Импортируем драйвер PostgreSQL
	"log"
	"net/http"
)

var db *sql.DB

// Инициализация базы данных PostgreSQL
func initDB() {
	var err error
	connStr := "user=postgres dbname=lempordb sslmode=disable password=yourpassword"
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	// Создаем таблицу users, если она не существует
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		hash TEXT NOT NULL,
		counter INTEGER NOT NULL
	);`
	_, err = db.Exec(createTableQuery)
	if err != nil {
		log.Fatalf("Error creating table: %v", err)
	}
}

// Регистрация пользователя
func registerUser(username string, initialHash string) {
	query := `INSERT INTO users (username, hash, counter) VALUES ($1, $2, $3)`
	_, err := db.Exec(query, username, initialHash, 1000)
	if err != nil {
		log.Fatalf("Error registering user: %v", err)
	}
	fmt.Printf("User %s registered successfully\n", username)
}

// Аутентификация пользователя
func authenticateUser(username string, hash string) bool {
	var storedHash string
	var counter int
	query := `SELECT hash, counter FROM users WHERE username = $1`
	err := db.QueryRow(query, username).Scan(&storedHash, &counter)
	if err != nil {
		log.Printf("Error fetching user data: %v", err)
		return false
	}

	// Проверяем хэш
	newHash := sha256.Sum256([]byte(hash))
	newHashStr := hex.EncodeToString(newHash[:])
	if newHashStr == storedHash {
		// Обновляем хэш и счетчик
		updateQuery := `UPDATE users SET hash = $1, counter = $2 WHERE username = $3`
		_, err = db.Exec(updateQuery, hash, counter-1, username)
		if err != nil {
			log.Printf("Error updating user data: %v", err)
		}
		return true
	}
	return false
}

func main() {
	initDB()
	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		username := r.URL.Query().Get("username")
		initialHash := r.URL.Query().Get("initialHash")
		registerUser(username, initialHash)
		fmt.Fprintf(w, "User %s registered successfully", username)
	})
	http.HandleFunc("/authenticate", func(w http.ResponseWriter, r *http.Request) {
		username := r.URL.Query().Get("username")
		hash := r.URL.Query().Get("hash")
		if authenticateUser(username, hash) {
			fmt.Fprintf(w, "User %s authenticated successfully", username)
		} else {
			fmt.Fprintf(w, "Authentication failed for user %s", username)
		}
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
