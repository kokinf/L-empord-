package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

func main() {
	for {
		// Отображаем меню
		fmt.Println("1. Register")
		fmt.Println("2. Authenticate")
		fmt.Println("3. Exit")
		fmt.Print("Choose an option: ")

		// Читаем выбор пользователя
		var choice int
		_, err := fmt.Scan(&choice)
		if err != nil {
			fmt.Println("Invalid input, please enter a number.")
			continue
		}

		// Обрабатываем выбор
		switch choice {
		case 1:
			register()
		case 2:
			authenticate()
		case 3:
			fmt.Println("Exiting...")
			os.Exit(0)
		default:
			fmt.Println("Invalid option, please try again.")
		}
	}
}

func register() {
	var username string
	fmt.Print("Enter username: ")
	_, err := fmt.Scan(&username)
	if err != nil {
		fmt.Println("Error reading username:", err)
		return
	}

	// Генерируем начальный хэш
	initialHash := generateHashChain(username, 1000)

	// Отправляем запрос на сервер для регистрации
	resp, err := http.Get("http://localhost:8080/register?username=" + username + "&initialHash=" + initialHash)
	if err != nil {
		fmt.Println("Error registering user:", err)
		return
	}
	defer resp.Body.Close()

	// Читаем ответ от сервера
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(body))
}

func authenticate() {
	var username string
	var password string
	fmt.Print("Enter username: ")
	_, err := fmt.Scan(&username)
	if err != nil {
		fmt.Println("Error reading username:", err)
		return
	}

	fmt.Print("Enter password: ")
	_, err = fmt.Scan(&password)
	if err != nil {
		fmt.Println("Error reading password:", err)
		return
	}

	// Отправляем запрос на сервер для аутентификации
	resp, err := http.Get("http://localhost:8080/authenticate?username=" + username + "&hash=" + password)
	if err != nil {
		fmt.Println("Error authenticating user:", err)
		return
	}
	defer resp.Body.Close()

	// Читаем ответ от сервера
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(body))
}

func generateHashChain(seed string, iterations int) string {
	hash := sha256.Sum256([]byte(seed))
	for i := 1; i < iterations; i++ {
		hash = sha256.Sum256(hash[:])
	}
	return hex.EncodeToString(hash[:])
}
