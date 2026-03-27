package main

import (
	"bufio"
	"log"
	"net/http"
	"time"
)

const url = "http://localhost:3000/handshake"

func reqSSE(client *http.Client) <-chan string {
	// create channel for messages
	messages := make(chan string)

	go func() {
		defer close(messages)

		log.Println("Client started")

		// reconnect loop
		for {
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				log.Println("error creating request:", err)
				time.Sleep(time.Second)
				continue
			}

			log.Println("attempting to send request...")

			res, err := client.Do(req)
			if err != nil {
				log.Println(err)
				time.Sleep(1 * time.Second)
				continue
			}

			log.Println("connection established with server..")

			// proceed to reading the stream
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
			// if connection died, go to reconnect
			res.Body.Close()

			log.Println("connection closed, reconnecting...")
			time.Sleep(time.Second)
		}
	}()
	return messages
}

func main() {
	client := new(http.Client)
	// infinite message processing loop
	for msg := range reqSSE(client) {
		log.Println("recv:", msg)
	}
}
