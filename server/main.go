package main

import (
	"fmt"
	"log"
	"net/http"
)

// channel for transmitting messages from server to active client
var messageChan chan string

func handleSSE() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log.Println("received connection request from client")

		// Set headers so the browser understands:
		// this is not a regular page, but streaming data (SSE)
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache") // Disable stream caching
		w.Header().Set("Connection", "keep-alive")  // Keep connection open
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// initialize controller that allows pushing data to client
		// without closing the stream
		rc := http.NewResponseController(w)

		// create channel for messages
		messageChan = make(chan string)

		// --- CLEANUP RESOURCES ---
		// on function exit (connection break)
		defer func() {
			if messageChan != nil {
				close(messageChan)
				messageChan = nil
			}
			log.Println("connection with client closed")
		}()

		// --- MAIN LOOP --- ("trap-request")
		// hold connection and wait for events
		for {

			// --- CHANNEL CHECK ---
			select {

			// --- SEND MESSAGE TO CLIENT ---
			// if a message arrived in the channel
			case message := <-messageChan:

				// SSE format: data: <message>\n\n
				if _, err := fmt.Fprintf(w, "data: %s\n\n", message); err != nil {
					log.Println("write error:", err)
					return
				}
				// Force send data from buffer to network
				if err := rc.Flush(); err != nil {
					log.Println("flush error:", err)
					return
				}

			// If client closed the stream
			// signal close handler
			case <-r.Context().Done():
				return

			}
		}

	}
}

func sendMessage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if there is an active SSE connection
		if messageChan != nil {
			log.Println("sending message to client...")
			messageChan <- "Hello world!"
		}
		// Return 204 (success, no response body)
		w.WriteHeader(http.StatusNoContent)
	}
}

func main() {

	log.Println(
		"\n Server started.\n",
		"First open a connection (curl or browser),\n",
		"then in another terminal execute:\n",
		"curl localhost:3000/sendmessage\n",
	)

	http.HandleFunc("/handshake", handleSSE())     // endpoint for connection
	http.HandleFunc("/sendmessage", sendMessage()) // endpoint for sending message

	err := http.ListenAndServe("localhost:3000", nil)
	if err != nil {
		log.Fatal("server error: %w", err)
	}

}
