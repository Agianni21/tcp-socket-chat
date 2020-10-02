package server

import (
	"bufio"
	"log"
	"net"
	"strconv"
	"strings"
)

// InitServer starts the chat server
func InitServer() {
	log.Println("STARTING SERVER")
	listener, err := net.Listen("tcp", "127.0.0.1:7777")
	must(err)

	defer listener.Close()

	users := make(map[string]net.Conn, 128)
	chatrooms := make(map[string]Chatroom)

	// start default chatroom
	defaultChatroom := InitChatroom("default")
	go ChatroomWorker(defaultChatroom, users)
	chatrooms["default"] = defaultChatroom

	// start the lobby spawner
	lobbyChannel := make(chan string, 128)
	go LobbySpawner(chatrooms, users, lobbyChannel)

	for {
		// for each new connection to the server create a worker that will handle it
		conn, err := listener.Accept()
		must(err)

		NewUserConnectionWorker(conn, lobbyChannel, users)
	}
}

// Broadcaster handles a go routine that broadcasts a message to all the clients on "users"
func Broadcaster(messages <-chan Message, users map[string]net.Conn, shutdown <-chan bool) {
	for {
		select {
		case <-shutdown:
			break
		case message := <-messages:
			for _, connection := range users {
				connection.Write([]byte(message.Content + "\n"))
			}
		}
	}
}

// ClientWorker go routine that handles a client connection on a chatroom
func ClientWorker(username string, conn net.Conn, sendMessage chan<- Message, disconnect chan<- string, users map[string]net.Conn) {
	reader := bufio.NewScanner(conn)
	for {
		ok := reader.Scan()

		if !ok {
			log.Println(username + " disconnected")
			conn.Close()
			disconnect <- username
			break
		}

		text := reader.Text()

		if text == "end" {
			conn.Close()
			disconnect <- username
			break
		} else if text != "" {
			sendMessage <- Message{
				User:    username,
				Content: username + ": " + text,
			}
		}
	}

}

// NewUserConnectionWorker go routine that handles a new connection to the server, setting username
func NewUserConnectionWorker(conn net.Conn, newRegistration chan<- string, users map[string]net.Conn) {
	reader := bufio.NewScanner(conn)
	conn.Write([]byte("Select username\n"))
	for {
		ok := reader.Scan()
		// user closed the connection before registering
		if !ok {
			conn.Close()
			break
		}

		text := reader.Text()

		if text == "end" {
			conn.Close()
			break
		}

		if _, ok := users[text]; ok {
			conn.Write([]byte("Username already in use\n"))
		} else {
			users[text] = conn
			newRegistration <- text
			break
		}
	}
}

// LobbySpawner go routine that creates lobbyworkers for clients
func LobbySpawner(chatrooms map[string]Chatroom, users map[string]net.Conn, newLobbyForUser <-chan string) {
	for {
		user := <-newLobbyForUser
		go LobbyWorker(user, users[user], users, chatrooms)
	}
}

// LobbyWorker go routine that handles joining, creating, listing chatrooms for a client
func LobbyWorker(user string, conn net.Conn, users map[string]net.Conn, chatrooms map[string]Chatroom) {
	reader := bufio.NewScanner(conn)
	conn.Write([]byte("Welcome to the Lobby\n"))
	conn.Write([]byte("actions: list, join <CHATROOM>, create <CHATROOM>, end\n"))
	for {
		ok := reader.Scan()
		// user closed the connection before registering
		if !ok {
			conn.Close()
			break
		}

		text := reader.Text()

		if text == "end" {
			conn.Close()
			break
		}

		switch strings.Split(text, " ")[0] {
		case "create":
			parsed := strings.Split(text, " ")
			if len(parsed) < 2 {
				conn.Write([]byte("Invalid create command\n"))
				continue
			}

			maxUsers := uint64(64)
			if len(parsed) > 2 {
				maxUsers, _ = strconv.ParseUint(parsed[2], 10, 64)
			}

			room := parsed[1]

			if _, ok := chatrooms[room]; ok {
				// chatroom exists
				conn.Write([]byte("Chatroom " + room + "already exists\n"))
				continue
			} else {
				conn.Write([]byte("Creating chatroom " + room + "\n"))
				newChatroom := InitChatroom(room)
				newChatroom.MaxUsers = maxUsers
				newChatroom.NewUserChan <- user
				chatrooms[room] = newChatroom
				go ChatroomWorker(newChatroom, users)
				conn.Write([]byte("Joining " + room + "\n"))
				break
			}

		case "list":
			for chatroom := range chatrooms {
				conn.Write([]byte(chatroom + "\n"))
			}
			continue
		case "join":
			parsed := strings.Split(text, " ")
			if len(parsed) != 2 {
				conn.Write([]byte("Invalid join command\n"))
				continue
			}

			room := parsed[1]
			if _, ok := chatrooms[room]; ok {
				// chatroom exists
				// check if chatroom has capacity
				if chatrooms[room].MaxUsers == uint64(len(chatrooms[room].Users)) {
					conn.Write([]byte(room + "At max capacity, can't join\n"))
					continue
				}
				conn.Write([]byte("Joining " + room + "\n"))
				chatrooms[room].NewUserChan <- user
				break
			} else {
				conn.Write([]byte("Chatroom " + room + " doesn't exist\n"))
				continue
			}
		default:
			conn.Write([]byte("Command " + text + " doesn't exist\n"))
			continue
		}
		break
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
