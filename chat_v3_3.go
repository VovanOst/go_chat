package main

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strings"
	"sync"
)

type Client struct {
	ID       string
	Name     string
	Socket   *websocket.Conn
	Outgoing chan string
}

type ChatChannel struct {
	Name    string
	Clients map[*Client]bool
	mu      sync.Mutex
}

var (
	upgrader = websocket.Upgrader{}
	clients  = struct {
		sync.RWMutex
		m map[string]*Client
	}{m: make(map[string]*Client)}

	channels = struct {
		sync.RWMutex
		m map[string]*ChatChannel
	}{m: make(map[string]*ChatChannel)}
)

func main() {
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}
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
	//_, name, err := conn.ReadMessage()
	//if err != nil {
	//	log.Println("Ошибка при чтении имени:", err)
	//	return
	//	}
	//colonIndex := strings.Index(string(name), ":")
	//if colonIndex != -1 {
	// Разделяем строку на никнейм и сообщение.

	//	client.Name = strings.TrimSpace(string(name)[:colonIndex])
	//	log.Println("Никнейм:", client.Name)
	//}
	for {
		// Чтение сообщения от клиента
		_, rawMessage, err := conn.ReadMessage()
		if err != nil {
			log.Println("Ошибка при чтении сообщения:", err)
			break
		}
		message := string(rawMessage) // Преобразование в строку
		colonIndex := strings.Index(message, ":")
		if colonIndex != -1 {
			// Разделяем строку на никнейм и текст сообщения
			nickname := strings.TrimSpace(message[:colonIndex])
			message := strings.TrimSpace(message[colonIndex+1:])
			client.Name = nickname
			// Выводим никнейм и сообщение в консоль (или выполняем другую обработку)
			fmt.Printf("Никнейм: %s, Сообщение: %s\n", nickname, message)

			//message = strings.TrimPrefix(message, client.Name)
			log.Println("message:", message)
			log.Println("has join:", strings.Contains(message, "/join"))
			log.Println("has private:", strings.Contains(message, "/private"))
			// Проверяем, начинается ли сообщение с команды /join
			if strings.Contains(message, "/join ") {
				channelName := strings.TrimLeft(message, "/join ")
				log.Printf("Клиент %s присоединился к каналу %s", client.Name, channelName)
				client.JoinChatChannel(channelName)
			} else if strings.Contains(message, "/private") {

				log.Printf("Cообщение в привате:%s ", message)
				// Если сообщение начинается с /private, это приватное сообщение

				parts := strings.SplitN(strings.TrimLeft(message, "/private "), " ", 2)
				recipientName := parts[0]
				privateMessage := parts[1]
				log.Printf("Клиент %s написал сообщение клиенту %s .Сообщение:%s ", client.Name, recipientName, privateMessage)
				client.SendPrivateMessage(recipientName, privateMessage)
			} else {
				client.SendToChannel(nickname, message)
			}
		} else {
			// Если нет двоеточия, обрабатываем как обычное сообщение
			fmt.Printf("Обычное сообщение: %s\n", message)
		}
	}

	client.LeaveAllChatChannels()

	clients.Lock()
	delete(clients.m, clientID)
	clients.Unlock()
}

// Методы для структуры Client

func (c *Client) SendMessage() {
	defer func() {
		clients.Lock()
		delete(clients.m, c.ID)
		clients.Unlock()

		close(c.Outgoing)
	}()

	for message := range c.Outgoing {
		err := c.Socket.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			log.Println("Ошибка при отправке сообщения:", err)
			break
		}
	}
}

func (c *Client) JoinChatChannel(channelName string) {
	// Логика добавления клиента в канал
	channels.Lock()
	defer channels.Unlock()

	// Проверяем, существует ли канал
	channel, exists := channels.m[channelName]
	if !exists {
		// Если канала нет, создаем новый и добавляем клиента
		channel = &ChatChannel{
			Name:    channelName,
			Clients: make(map[*Client]bool),
		}
		channels.m[channelName] = channel
	}

	// Добавляем клиента в канал
	channel.mu.Lock()
	defer channel.mu.Unlock()
	channel.Clients[c] = true
}

func (c *Client) SendPrivateMessage(recipientName, message string) {
	clients.RLock()
	defer clients.RUnlock()

	// Поиск получателя сообщения
	var recipient *Client
	for _, client := range clients.m {
		if client.Name == recipientName {
			recipient = client
			break
		}
	}

	if recipient != nil {
		select {
		case recipient.Outgoing <- "Приватное от " + c.Name + ": " + message:
		default:
			close(recipient.Outgoing)
		}
	}
}

func (c *Client) SendToChannel(nickname, message string) {
	channels.RLock()
	defer channels.RUnlock()

	// Поиск канала, в котором находится клиент
	var channel *ChatChannel
	for _, ch := range channels.m {
		ch.mu.Lock()
		_, exists := ch.Clients[c]
		ch.mu.Unlock()

		if exists {
			channel = ch
			break
		}
	}

	if channel != nil {
		channel.mu.Lock()
		defer channel.mu.Unlock()

		// Отправка сообщения всем клиентам в канале
		for client := range channel.Clients {
			select {
			case client.Outgoing <- nickname + ": " + message:
			default:
				close(client.Outgoing)
				delete(channel.Clients, client)
			}
		}
	}
}

func (c *Client) LeaveAllChatChannels() {
	// Логика выхода клиента из всех каналов
	channels.Lock()
	defer channels.Unlock()

	for _, channel := range channels.m {
		channel.mu.Lock()
		_, exists := channel.Clients[c]
		if exists {
			delete(channel.Clients, c)
		}
		channel.mu.Unlock()
	}
}

// Остальной код остается без изменений
