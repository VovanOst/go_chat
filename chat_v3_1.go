package main

import (
	"github.com/google/uuid"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

type Client struct {
	ID       string
	Name     string
	Socket   *websocket.Conn
	Outgoing chan string
}

var (
	upgrader = websocket.Upgrader{}
	clients  = struct {
		sync.RWMutex
		m map[string]*Client
	}{m: make(map[string]*Client)}
)

var upgrader = websocket.Upgrader{}
var channels = make(map[string]*Channel)
var clients = make(map[string]*Client)

func main() {
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

	clientID := uuid.New().String() // Генерация UUID V4

	client := &Client{
		ID:       clientID,
		Socket:   conn,
		Outgoing: make(chan string),
	}

	clients.Lock()
	clients.m[clientID] = client
	clients.Unlock()

	go client.SendMessage()

	// Чтение имени отправителя
	_, name, err := conn.ReadMessage()
	if err != nil {
		log.Println("Ошибка при чтении имени:", err)
		return
	}

	client.Name = string(name)

	for {
		// Чтение сообщения от клиента
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Ошибка при чтении сообщения:", err)
			break
		}

		// Проверяем, начинается ли сообщение с команды /join
		if string(message[0:6]) == "/join " {
			channelName := string(message[6:])
			client.JoinChannel(channelName)
		} else if string(message[0:8]) == "/private" {
			// Если сообщение начинается с /private, это приватное сообщение
			parts := strings.SplitN(string(message[9:]), " ", 2)
			recipientName := parts[0]
			privateMessage := parts[1]
			client.SendPrivateMessage(recipientName, privateMessage)
		} else {
			client.SendToChannel(message)
		}
	}

	client.LeaveAllChannels()

	clients.Lock()
	delete(clients.m, clientID)
	clients.Unlock()
}
