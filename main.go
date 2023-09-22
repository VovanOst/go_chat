package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
)

type Client struct {
	conn   net.Conn
	name   string
	reader *bufio.Reader
}

var clients []Client

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Не удалось прослушивать порт 8080: %v", err)
	}
	defer listener.Close()

	fmt.Println("Сервер чата запущен. Ожидание подключений...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Ошибка при подключении: %v", err)
			continue
		}

		reader := bufio.NewReader(conn)
		name, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Ошибка при чтении имени клиента: %v", err)
			continue
		}

		name = strings.TrimSpace(name)
		client := Client{
			conn:   conn,
			name:   name,
			reader: reader,
		}

		clients = append(clients, client)
		go handleClient(client)
	}
}

func handleClient(client Client) {
	defer client.conn.Close()

	broadcastMessage(fmt.Sprintf("Клиент %s присоединился к чату.", client.name))

	for {
		message, err := client.reader.ReadString('\n')
		if err != nil {
			log.Printf("Ошибка при чтении сообщения от клиента %s: %v", client.name, err)
			break
		}

		message = strings.TrimSpace(message)
		broadcastMessage(fmt.Sprintf("%s: %s", client.name, message))
	}

	removeClient(client)
	broadcastMessage(fmt.Sprintf("Клиент %s покинул чат.", client.name))
}

func broadcastMessage(message string) {
	fmt.Println(message)

	for _, client := range clients {
		client.conn.Write([]byte(message + "\n"))
	}
}

func removeClient(client Client) {
	for i, c := range clients {
		if c == client {
			clients = append(clients[:i], clients[i+1:]...)
			break
		}
	}
}
