package main

import (
	"bufio"
	"log"
	"net/http"
	"time"
)

const url = "http://localhost:3000/handshake"

func reqSSE(client *http.Client) <-chan string {
	// создаем канал для сообщений
	messages := make(chan string)

	go func() {
		defer close(messages)

		log.Println("Клиент запущен")

		// цикл где крутяться запросы реконнекты
		for {
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				log.Println("ошибка при создании запроса:", err)
				time.Sleep(time.Second)
				continue
			}

			log.Println("попытка отправить запрос...")

			res, err := client.Do(req)
			if err != nil {
				log.Println(err)
				time.Sleep(1 * time.Second)
				continue
			}

			log.Println("установлено соединение с сервером..")

			// переходим к чтению потока
			scanner := bufio.NewScanner(res.Body)
			for scanner.Scan() {
				line := scanner.Text()
				messages <- line
			}
			if err := scanner.Err(); err != nil {
				log.Println("ошибка чтения:", err)
			} else {
				log.Println("сервер закрыл соединение")
			}
			// если соединение умерло, идем на реконект
			res.Body.Close()

			log.Println("соединение закрыто, идем на реконект...")
			time.Sleep(time.Second)
		}
	}()
	return messages
}

func main() {
	client := new(http.Client)
	// бесконечный цикл обработки сообщений
	for msg := range reqSSE(client) {
		log.Println("recv:", msg)
	}
}
