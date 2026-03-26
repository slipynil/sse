package main

import (
	"fmt"
	"log"
	"net/http"
)

var messageChan chan string

func handleSSE() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log.Println("Get handshake from client")

		// prepare the header for browser (if you want to use browser as client)
		// this header can be use by browser so that it will directly print the message
		// without need to waiting the end of response from server
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// prepare the flusher
		rc := http.NewResponseController(w)

		// instantiate the channel
		messageChan = make(chan string)

		// close the channel after exit the function
		defer func() {
			if messageChan != nil {
				close(messageChan)
				messageChan = nil
			}
			log.Println("client connection is closed")
		}()

		// trap the request under loop forever
		for {

			select {

			// message will received here and printed
			case message := <-messageChan:

				// write the message to buffer
				if _, err := fmt.Fprintf(w, "data: %s\n\n", message); err != nil {
					log.Println("write error:", err)
					return
				}
				// send the buffer to client
				if err := rc.Flush(); err != nil {
					log.Println(err)
				}

			// connection is closed then defer will be executed
			case <-r.Context().Done():
				return

			}
		}

	}
}

func sendMessage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if messageChan != nil {
			log.Println("print message to client")
			messageChan <- "Hello Client"
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func main() {

	log.Println(
		"\n Server is running,\n",
		"makesure you already run the client\n",
		"open another console and call\n",
		"curl localhost:3000/sendmessage\n",
	)

	http.HandleFunc("/handshake", handleSSE())
	http.HandleFunc("/sendmessage", sendMessage())

	err := http.ListenAndServe("localhost:3000", nil)
	if err != nil {
		log.Fatal("HTTP server error: %w", err)
	}

}
