package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

type Client struct {
	ID     string
	Name   string
	Socket *websocket.Conn
}

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

var clients = make(map[string]*Client)

func main() {
	rand.Seed(time.Now().UnixNano())
	http.HandleFunc("/websocket", handleWebSocket)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Ошибка при обновлении соединения:", err)
		return
	}
	defer conn.Close()

	// Генерация уникального идентификатора для клиента
	clientID := generateClientID()

	// Создание клиента
	client := &Client{
		ID:     clientID,
		Socket: conn,
	}
	// Чтение имени отправителя
	_, name, err := conn.ReadMessage()
	if err != nil {
		log.Println("Ошибка при чтении имени:", err)
		return
	}

	// Установка имени отправителя в клиенте
	client.Name = string(name)

	clients[clientID] = client

	for {
		// Чтение сообщения от клиента
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Ошибка при чтении сообщения:", err)
			break
		}

		// Рассылка сообщения всем клиентам, кроме отправителя
		for _, c := range clients {
			//if c.ID != clientID {
			//messageWithSender := fmt.Sprintf("%s: %s", client.Name, message)
			messageWithSender := fmt.Sprintf("%s", message)
			err := c.Socket.WriteMessage(websocket.TextMessage, []byte(messageWithSender))
			if err != nil {
				log.Println("Ошибка при отправке сообщения:", err)
				break
			}
			//}
		}
	}

	// Удаление клиента при закрытии соединения
	delete(clients, clientID)
}

func generateClientID() string {
	// Реализуйте вашу логику генерации уникального идентификатора
	// В этом примере мы будем использовать случайное число
	return "client_" + strconv.Itoa(rand.Intn(1000))
}
