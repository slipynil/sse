<div align="center">

# SSE GUIDE

**Dead simply guide how is SSE works**

[🇬🇧 English](README.md) • [🇷🇺 Russian](docs/README_RU.md)

Made with ❤️ by [@Slipynil](https://github.com/slipynil)

[![Telegram](https://img.shields.io/badge/Telegram-2CA5E0?style=for-the-badge&logo=telegram&logoColor=white)](https://t.me/G0ndonClub)

</div>

## What is *SSE*?

***SSE (Server-Sent Events)*** is a technology that allows a server to send data to a client in a unidirectional mode. In the classic HTTP protocol, the server cannot initiate data transmission on its own, as it only responds to incoming requests. *SSE* solves this problem by allowing the server to push messages over an open connection.

## Why use *SSE*?
*SSE* is optimal when unidirectional data transmission from server to client in real-time is required, and a reverse message flow from the client is not needed for the task.

## When should you use *SSE*?
Choose *SSE* if you need simple one-way communication (for example, for notification feeds, currency rates, or progress bars). However, if your application requires full-fledged bidirectional real-time data exchange (chat, online game), it's better to use *WebSocket*.

## How does the *SSE* system work?
Since the server cannot initiate transmission first, the connection is always **initialized by the client** (**1**). It sends a special HTTP request indicating that it's ready to receive an event stream (*Event Stream*). The server doesn't close the connection after the first response but keeps it open, "holding" the channel for transmitting future messages.

## What does this mean?
Instead of sending data and breaking the connection, the server switches the HTTP connection to **streaming mode**. Within this "eternal" connection, the server can send a new message to the client at any time (**2, 3, 4**) without needing to receive a new request.
In fact, the client is in a state of waiting for the response to continue. This continues until the server itself decides to end the session or until the connection is broken from the client side (**5**). In case of an accidental disconnection, the client typically automatically attempts to reconnect.

<p align="center">
  <img src="docs/sse-diagram.png" alt="Sequence Diagram">
</p>

## How to break the connection?
The server can close the connection at any time by simply terminating the data stream transmission. On its side, the client can also easily interrupt the connection by canceling the active HTTP request. As soon as the client breaks the connection, the server receives a notification about it and releases the resources allocated for this channel.

## Why not use WebSocket in all cases?
Although WebSocket provides full-fledged bidirectional communication, SSE has its advantages:

* Implementation simplicity: SSE works on top of the regular HTTP protocol and doesn't require setting up special proxy servers or complex protocols.
* Resource savings (RPS): For simple notifications, SSE consumes fewer server resources as it works within the standard HTTP stack.
* Automatic reconnection: SSE standard has built-in support for connection recovery on disconnection — the browser will do it itself, without writing extra code.
* Ease of debugging: SSE message stream is plain text that's easy to view in the browser's developer console.

Conclusion: If you don't need to send data from client to server every 100ms (like in games), SSE will be a lighter and more reliable solution.

# How to implement an *SSE* server in Go?
**This is a simplified example for one active connection**

## System Architecture
This implementation uses two endpoints:
* **`/handshake`** - endpoint for establishing SSE connection. The client connects to it and receives an open event stream.
* **`/sendmessage`** - endpoint for triggering message sending. An external request to this endpoint places a message in the `messageChan` channel, from where it's transmitted to the active client.

The global `messageChan` channel serves as a link between the sending endpoint and the active SSE connection.

⚠️ **Important limitations:**
* The global channel is recreated with each new connection
* With multiple simultaneous connections, race conditions are possible
* For production environments, a map of channels with mutexes is needed to support multiple clients

We will use the built-in ***net/http*** library to work with *HTTP*. You can use other libraries such as *gin* or *echo*.
```go
func (w http.ResponseWriter, r *http.Request)
```
Create a channel named `messageChan`. The message to the client will be delivered through it. The channel variable should be defined somewhere where we can access it. Let's say we're just sending a string message for now.
```go
messageChan = make(chan string)
```
For the "*trap-request*" part of the code, we will use an infinite loop. Inside this loop, the process will be blocked by the `messageChan` channel. The `messageChan` channel continues to wait and listen for message readiness. If a message is available, it will be sent to the HTTP writer. We also need to send the message to the cache so the client can see it. In case of receiving a connection close signal from the client, the `r.Context().Done()` function will issue a "*close*" signal, and we can exit the loop (actually, we exit the function, not just the loop). Here we maintain the connection.
```go
rc := http.NewResponseController(w)

for {
	select {
		// message send handler
		case message := <- messageChan:
			fmt.Fprintf(w, "data: %s\n\n", message)
			rc.Flush()
		// signal close handler
		case <-r.Context().Done():
			return
	}
}
```
Right before exiting the function, you need to close the channel to avoid memory leaks. This can be done via `defer`.
```go
defer func() {
	close(messageChan)
	messageChan = nil
}()
```
## How to start sending a message to the client?
We can simply put a message in the channel. Make sure the `messageChan` channel was created before this.
```go
messageChan <- "Hello world!"
```

# How to implement an SSE client in Go?

## 1. Client and Request Initialization
First, we initialize the HTTP client and prepare a GET request to the server endpoint.
```go
client := new(http.Client)
req, _ := http.NewRequest("GET", "http://localhost:3000/handshake", nil)
```
### 2. Connection Establishment Loop (Reconnect)
To make the client fault-tolerant, we use an infinite loop. If the server is unavailable or the connection breaks, the client pauses and tries to connect again.
```go
for {
	res, err := client.Do(req)
	if err != nil {
		log.Println("Connection error, retrying in 1 sec...")
		time.Sleep(time.Second)
		continue
	}
	log.Println("Connection established..")

	// proceed to reading the stream
	scanner := bufio.NewScanner(res.Body)
}
```

### 3. Reading Streaming Data (Stream Processing)
*SSE* transmits data in a stream. The most efficient way to read such a stream in Go is to use `bufio.NewScanner`. It automatically blocks execution and waits for a new line from the server.
```go
scanner := bufio.NewScanner(res.Body)
for scanner.Scan() {
	// Each new line from the server ends up here
	line := scanner.Text()
	messages <- line
}
```
### 4. Handling Stream Termination
The `scanner.Scan()` loop will terminate itself if:
1. The server properly closed the connection.
2. A network error occurred (can be checked via `scanner.Err()`).
After exiting the inner reading loop, `defer res.Body.Close()` will trigger, and the outer `for` loop will send the client to a new reconnection attempt.

### 5. Channel Wrapper
To conveniently use the received data in other parts of the program, we wrap all the logic in a goroutine that returns a read-only channel:
```go
func reqSSE(client *http.Client) <-chan string {
    messages := make(chan string)
    go func() {
        defer close(messages)

        // Outer loop for reconnect
        for {
            req, err := http.NewRequest("GET", url, nil)
            if err != nil {
                log.Println("error creating request:", err)
                time.Sleep(time.Second)
                continue
            }

            res, err := client.Do(req)
            if err != nil {
                log.Println("connection error:", err)
                time.Sleep(time.Second)
                continue
            }

            // Inner loop for reading the stream
            scanner := bufio.NewScanner(res.Body)
            for scanner.Scan() {
                line := scanner.Text()
                messages <- line
            }

            res.Body.Close()
            time.Sleep(time.Second) // pause before reconnect
        }
    }()
    return messages
}
```

Usage in main:
```go
func main() {
    client := new(http.Client)
    for msg := range reqSSE(client) {
        log.Println("recv:", msg)
    }
}
```

# How to test the system?

To verify SSE functionality, follow these steps:

### Step 1: Start the server
```bash
go run server/main.go
```
You will see a message that the server is running on `localhost:3000`.

### Step 2: Start the client
In another terminal, execute:
```bash
go run client/main.go
```
The client will establish a connection with the server and wait for messages.

### Step 3: Send a message
In a third terminal, execute:
```bash
curl localhost:3000/sendmessage
```
The client should receive and output the message "Hello world!".

### Expected Result
In the client terminal, you will see:
```
recv: data: Hello world!
recv:
```

### Why does the client output two lines?

This is related to the SSE protocol format and how the client reads the stream data:

1. **Server sends** (server/main.go:53):
   ```go
   fmt.Fprintf(w, "data: %s\n\n", message)
   ```
   The SSE format requires a double newline `\n\n` after each event. This means the server sends:
   ```
   data: Hello world!\n\n
   ```

2. **Client reads line by line** (client/main.go:43-46):
   ```go
   scanner := bufio.NewScanner(res.Body)
   for scanner.Scan() {
       line := scanner.Text()
       messages <- line
   }
   ```
   `bufio.Scanner` by default splits the stream by newline characters `\n`. Therefore, it reads:
   - **First line**: `data: Hello world!` (up to the first `\n`)
   - **Second line**: empty string `""` (between the two `\n`)

3. **Console output**:
   ```go
   log.Println("recv:", msg)
   ```
   Outputs both read lines:
   - `recv: data: Hello world!` (first message)
   - `recv:` (second message — empty line)

**Conclusion**: The double newline `\n\n` is the SSE standard for separating events. The client, reading line by line, interprets this as two separate lines: one with data and one empty.

You can send messages multiple times by executing the `curl` command repeatedly.
