package server

import (
	"log"
	"net"
)

// Chatroom instance of a chatroom
type Chatroom struct {
	Name           string
	Users          map[string]net.Conn
	BroadcastChan  chan Message
	DisconnectChan chan string
	NewUserChan    chan string
	MaxUsers       uint64
}

// InitChatroom creates a Chatroom with default values
func InitChatroom(name string) Chatroom {
	chatroom := Chatroom{
		Name:           name,
		Users:          make(map[string]net.Conn),
		BroadcastChan:  make(chan Message, 128),
		DisconnectChan: make(chan string, 128),
		NewUserChan:    make(chan string, 128),
		MaxUsers:       64,
	}

	return chatroom
}

// ChatroomWorker should be used as a go routine.
// It will handle a single chatroom
func ChatroomWorker(chatroom Chatroom, globalUsers map[string]net.Conn) {
	log.Println("Chatroom " + chatroom.Name + " started")

	shutdownSignal := make(chan bool)

	go Broadcaster(chatroom.BroadcastChan, chatroom.Users, shutdownSignal)

	for {
		select {
		case user := <-chatroom.NewUserChan:
			// setup a worker for a new user on the chatroom
			log.Println(user + " connected to " + chatroom.Name)
			chatroom.Users[user] = globalUsers[user]
			go ClientWorker(user, globalUsers[user], chatroom.BroadcastChan, chatroom.DisconnectChan, globalUsers)
		case user := <-chatroom.DisconnectChan:
			// disconnect a user from the chatroom
			log.Println(user + " disconnected from " + chatroom.Name)
			delete(chatroom.Users, user)
			delete(globalUsers, user) // TODO add a better system for deleting users, probably a global handler

			// always keep the default chatroom open, but close empty chatrooms
			if !(chatroom.Name == "default") && len(chatroom.Users) == 0 {
				log.Println("Shutting down " + chatroom.Name + " chatroom")
				shutdownSignal <- true
				break
			}
		}
	}
}
