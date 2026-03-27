package main

import (
	"fmt"
	"log"
	"net/http"
)

// канал для передачи сообщений от сервера к активному клиенту
var messageChan chan string

func handleSSE() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log.Println("получен запрос на соединение от клиента")

		// Устанавливаем заголовки, чтобы браузер понял:
		// это не обычная страница, а потоковые данные (SSE)
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache") // Запрещаем кэшировать поток
		w.Header().Set("Connection", "keep-alive")  // Поддерживаем соединение открытым
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// инициализируем контроллер, который позволяет пушить данные клиенту
		// не закрываю поток
		rc := http.NewResponseController(w)

		// создаем канал для сообщений
		messageChan = make(chan string)

		// --- ОЧИЩАЕМ РЕСУРСЫ ---
		// при выходе из функции (разрыве связи)
		defer func() {
			if messageChan != nil {
				close(messageChan)
				messageChan = nil
			}
			log.Println("соединение с клиентом закрыто")
		}()

		// --- ОСНОВНОЙ ЦИКЛ --- ("trap-request")
		// удерживаем соединение и ждем событий
		for {

			// --- ПРОВЕРКА КАНАЛА ---
			select {

			// --- ОТПРАВКА СООБЩЕНИЯ КЛИЕНТУ ---
			// если в канал пришло сообщение
			case message := <-messageChan:

				// Формат SSE: data: <message>\n\n
				if _, err := fmt.Fprintf(w, "data: %s\n\n", message); err != nil {
					log.Println("ошибка записи:", err)
					return
				}
				// Принудительно отправляем данные из буфера в сеть
				if err := rc.Flush(); err != nil {
					log.Println("ошибка Flush:", err)
					return
				}

			// Если клиент сам зыкрыл поток
			// signal close handler
			case <-r.Context().Done():
				return

			}
		}

	}
}

func sendMessage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, есть ли активное SSE-соединение
		if messageChan != nil {
			log.Println("отправляем сообщение клиенту...")
			messageChan <- "Привет мир!"
		}
		// Возвращаем 204 (успешно, без тела ответа)
		w.WriteHeader(http.StatusNoContent)
	}
}

func main() {

	log.Println(
		"\n Сервер запущен.\n",
		"Сначала откройте соединение (curl или браузер),\n",
		"а затем в другом терминале выполните:\n",
		"curl localhost:3000/sendmessage\n",
	)

	http.HandleFunc("/handshake", handleSSE())     // эндпоинт для подключения
	http.HandleFunc("/sendmessage", sendMessage()) // эндпоинт для отправки сообщения

	err := http.ListenAndServe("localhost:3000", nil)
	if err != nil {
		log.Fatal("ошибка сервера: %w", err)
	}

}
