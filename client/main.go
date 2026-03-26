package main

import (
	"bufio"
	"log"
	"net/http"
	"time"
)

const url = "http://localhost:3000/handshake"

func reqSSE(client *http.Client) <-chan string {
	// prepare the required channel
	messages := make(chan string) // text channel

	go func() {
		defer close(messages)

		log.Println("\nClient is running",
			"\ntrying to initiate the communication to server\n")

		// цикл где крутяться запросы реконнекты
		for {
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				log.Println("failed to create new req:", err)
				time.Sleep(time.Second)
				continue
			}

			log.Println("try to shake hand to server")

			res, err := client.Do(req)
			if err != nil {
				log.Println(err)
				time.Sleep(1 * time.Second)
				continue
			}

			log.Println("connection to server is established..")

			scanner := bufio.NewScanner(res.Body)
			for scanner.Scan() {
				line := scanner.Text()
				messages <- line
			}
			if err := scanner.Err(); err != nil {
				log.Println("read error:", err)
			} else {
				log.Println("server closed connection")
			}
			// если соединение умерло, идем на реконект
			defer res.Body.Close()
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
