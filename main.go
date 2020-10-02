package main

import (
	"os"

	"elai.com/socket-chat/client"
	"elai.com/socket-chat/server"
)

func main() {
	if os.Args[1] == "server" {
		server.InitServer()
	} else {
		client.InitClient()
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
