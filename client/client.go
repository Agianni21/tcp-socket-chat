package client

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

// InitClient main function for client
func InitClient() {
	reader := bufio.NewScanner(os.Stdin)
	conn, err := net.Dial("tcp", "127.0.0.1:7777")
	must(err)

	go clientReader(conn)

	for {
		reader.Scan()
		bytes := []byte(reader.Text() + "\n")
		conn.Write(bytes)

		if string(bytes[:]) == "end\n" {
			conn.Close()
			fmt.Println("breaking")
			break
		}

	}
}

// clientReader receives messages from the server and prints them on stdout
func clientReader(conn net.Conn) {
	reader := bufio.NewScanner(conn)
	for {
		ok := reader.Scan()
		if !ok {
			fmt.Println("Server Disconnected")
			conn.Close()
			break
		}
		text := reader.Text()
		fmt.Println(text)
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
