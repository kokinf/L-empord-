package main

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"strings"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/sha3"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "your-password"
	dbname   = "lempordb"
)

var db *sql.DB

func initDB() {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var err error
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Успешное подключение к базе данных")

	// Создание таблицы, если её нет
	createTableQuery := `
    CREATE TABLE IF NOT EXISTS users (
        username VARCHAR(255) PRIMARY KEY,
        hash TEXT NOT NULL,
        counter INTEGER NOT NULL
    );`
	_, err = db.Exec(createTableQuery)
	if err != nil {
		log.Fatal(err)
	}
}

func hash(data []byte) []byte {
	h := sha3.New256()
	h.Write(data)
	return h.Sum(nil)
}

func registerUser(username string, initialHash []byte) bool {
	// Проверяем, существует ли пользователь
	query := "SELECT username FROM users WHERE username = $1;"
	var existingUsername string
	err := db.QueryRow(query, username).Scan(&existingUsername)
	if err == nil {
		fmt.Println("Пользователь уже существует:", username)
		return false
	}

	insertQuery := `
    INSERT INTO users (username, hash, counter)
    VALUES ($1, $2, 100);`

	_, err = db.Exec(insertQuery, username, hex.EncodeToString(initialHash))
	if err != nil {
		log.Println("Ошибка при регистрации пользователя:", err)
		return false
	}

	fmt.Println("Пользователь успешно зарегистрирован.")
	return true
}

func authenticateUser(username string, providedHash []byte) bool {
	var storedHash string
	var counter int

	query := "SELECT hash, counter FROM users WHERE username = $1;"
	err := db.QueryRow(query, username).Scan(&storedHash, &counter)
	if err != nil {
		log.Println("Ошибка при получении данных пользователя:", err)
		return false
	}

	storedHashBytes, _ := hex.DecodeString(storedHash)
	expectedHash := hash(providedHash)

	if !compareHashes(expectedHash, storedHashBytes) {
		return false
	}

	// Обновление хэша и счётчика
	newCounter := counter - 1
	newHash := hex.EncodeToString(providedHash)

	updateQuery := "UPDATE users SET hash = $1, counter = $2 WHERE username = $3;"
	_, err = db.Exec(updateQuery, newHash, newCounter, username)
	if err != nil {
		log.Println("Ошибка при обновлении данных пользователя:", err)
		return false
	}

	return true
}

func compareHashes(hash1, hash2 []byte) bool {
	return string(hash1) == string(hash2)
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		log.Println("Ошибка при чтении данных из соединения:", err)
		return
	}

	message := string(buffer[:n])
	parts := splitMessage(message)

	switch parts[0] {
	case "REGISTER":
		if len(parts) != 3 {
			conn.Write([]byte("Неверный формат регистрации\n"))
			return
		}
		username := parts[1]
		initialHash, err := hex.DecodeString(parts[2])
		if err != nil {
			conn.Write([]byte("Неверный формат хэша\n"))
			return
		}
		if registerUser(username, initialHash) {
			conn.Write([]byte("Регистрация успешна\n"))
		} else {
			conn.Write([]byte("Пользователь уже существует\n"))
		}
	case "AUTHENTICATE":
		if len(parts) != 3 {
			conn.Write([]byte("Неверный формат аутентификации\n"))
			return
		}
		username := parts[1]
		providedHash, err := hex.DecodeString(parts[2])
		if err != nil {
			conn.Write([]byte("Неверный формат хэша\n"))
			return
		}
		if authenticateUser(username, providedHash) {
			conn.Write([]byte("Аутентификация успешна\n"))
		} else {
			conn.Write([]byte("Аутентификация не удалась\n"))
		}
	case "GET_COUNTER":
		if len(parts) != 2 {
			conn.Write([]byte("Неверный формат запроса\n"))
			return
		}
		username := parts[1]

		query := "SELECT counter FROM users WHERE username = $1;"
		var counter int
		err := db.QueryRow(query, username).Scan(&counter)
		if err != nil {
			conn.Write([]byte("Пользователь не найден\n"))
			return
		}

		conn.Write([]byte(fmt.Sprintf("Текущий счётчик: %d\n", counter)))
	default:
		conn.Write([]byte("Неверная команда\n"))
	}
}

func splitMessage(message string) []string {
	return strings.Split(message, " ")
}

func main() {
	initDB()
	defer db.Close()

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal("Ошибка при запуске сервера:", err)
	}
	defer listener.Close()

	fmt.Println("Сервер запущен на порту 8080...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Ошибка при принятии соединения:", err)
			continue
		}
		go handleConnection(conn)
	}
}
