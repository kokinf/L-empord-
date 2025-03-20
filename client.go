package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"

	"golang.org/x/crypto/sha3"
)

func generateInitialHash() ([]byte, [][]byte) {
	secret := make([]byte, 32)
	rand.Read(secret)

	hashChain := make([][]byte, 101)
	hashChain[0] = secret

	for i := 1; i <= 100; i++ {
		hashChain[i] = hash(hashChain[i-1])
	}

	return hashChain[100], hashChain
}

func hash(data []byte) []byte {
	h := sha3.New256()
	h.Write(data)
	return h.Sum(nil)
}

func saveHashChainToFile(username string, hashChain [][]byte) error {
	fileName := username + "_chain.txt"
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("ошибка создания файла: %v", err)
	}
	defer file.Close()

	for _, hash := range hashChain {
		_, err = file.WriteString(hex.EncodeToString(hash) + "\n")
		if err != nil {
			return fmt.Errorf("ошибка записи в файл: %v", err)
		}
	}
	return nil
}

func loadHashChainFromFile(username string) ([][]byte, error) {
	fileName := username + "_chain.txt"
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("файл хэш-цепочки не найден: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	hashChain := make([][]byte, 0)
	for _, line := range lines {
		if line == "" {
			continue
		}
		hashBytes, err := hex.DecodeString(line)
		if err != nil {
			return nil, fmt.Errorf("ошибка декодирования хэша: %v", err)
		}
		hashChain = append(hashChain, hashBytes)
	}

	return hashChain, nil
}

func main() {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("Ошибка при подключении к серверу:", err)
		return
	}
	defer conn.Close()

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Выбери опцию:")
	fmt.Println("1. Регистрация")
	fmt.Println("2. Аутентификация")
	fmt.Println("3. Просмотр счётчика")
	fmt.Println("4. Выход")

	option, _ := reader.ReadString('\n')
	option = option[:len(option)-1] // Удаление символа новой строки

	switch option {
	case "1":
		fmt.Print("Введите имя пользователя: ")
		username, _ := reader.ReadString('\n')
		username = username[:len(username)-1]

		initialHash, hashChain := generateInitialHash()
		err := saveHashChainToFile(username, hashChain)
		if err != nil {
			fmt.Println("Ошибка при сохранении хэш-цепочки:", err)
			return
		}

		message := "REGISTER " + username + " " + hex.EncodeToString(initialHash)
		conn.Write([]byte(message))

		response := make([]byte, 1024)
		n, _ := conn.Read(response)
		fmt.Println(string(response[:n]))
	case "2":
		fmt.Print("Введите имя пользователя: ")
		username, _ := reader.ReadString('\n')
		username = username[:len(username)-1]

		hashChain, err := loadHashChainFromFile(username)
		if err != nil {
			fmt.Println("Ошибка при загрузке хэш-цепочки:", err)
			return
		}

		fmt.Print("Введите текущее значение счётчика: ")
		counterStr, _ := reader.ReadString('\n')
		counterStr = counterStr[:len(counterStr)-1]
		counter, _ := strconv.Atoi(counterStr)

		if counter < 1 || counter > 100 || len(hashChain) == 0 {
			fmt.Println("Неверное значение счётчика или хэш-цепочка не загружена.")
			return
		}

		providedHash := hashChain[counter-1]
		message := "AUTHENTICATE " + username + " " + hex.EncodeToString(providedHash)
		conn.Write([]byte(message))

		response := make([]byte, 1024)
		n, _ := conn.Read(response)
		fmt.Println(string(response[:n]))
	case "3":
		fmt.Print("Введите имя пользователя: ")
		username, _ := reader.ReadString('\n')
		username = username[:len(username)-1]

		message := "GET_COUNTER " + username
		conn.Write([]byte(message))

		response := make([]byte, 1024)
		n, _ := conn.Read(response)
		fmt.Println(string(response[:n]))
	case "4":
		fmt.Println("Выход...")
		return
	default:
		fmt.Println("Неверная опция")
	}

	// Завершение программы после выполнения операции
	return
}
