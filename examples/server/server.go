package main

import (
	con "con/con-go"
	"fmt"
	"log"
	"strconv"
	"time"
)

func main() {
	fmt.Println(" → Server")

	server := con.Server{}

	// Subscribe: all clients - msg 'msg-A'
	server.On("", "msg-A", func(msg con.Msg) (ans []byte) {
		var clientName string
		for i := range server.Clients {
			if server.Clients[i].ID == msg.Author {
				clientName = server.Clients[i].Name
			}
		}

		fmt.Println(" → Got message 'msg-A' from:", clientName)
		if msg.Body != nil {
			fmt.Println("    with body:", string(msg.Body))
		}
		return
	})

	// Subscribe: client 'client-a' - msg 'msg-A'
	server.On("client-a", "msg-A", func(msg con.Msg) (ans []byte) {
		fmt.Println(" → Got message 'msg-a' from 'client-a'")
		return
	})

	// Subscribe for next msg: client 'client-b' - msg 'msg-A'
	server.Once("client-b", "msg-A", func(msg con.Msg) (ans []byte) {
		fmt.Println(" → This will trigger only once")
		return
	})

	// Subscribe: client 'client-b' - all messages
	server.On("client-b", "", func(msg con.Msg) (ans []byte) {
		fmt.Println(" → Message from 'client-b' with name:", msg.Name)
		return
	})

	// Handle request
	server.On("client-a", "repeat", func(msg con.Msg) (ans []byte) {
		fmt.Println(" → Request from 'client-a' - 'repeat'")
		if msg.Body != nil {
			return append(msg.Body, msg.Body...)
		}
		return
	})

	go func() {
		err := server.Listen("/tmp/con-examples.sock")
		// go server.Listen("127.0.0.1:1234")
		if err != nil {
			log.Fatal("Cannot listen")
		}
	}()

	count := 0
	for {
		time.Sleep(160 * time.Millisecond)
		err := server.Broadcast("msg-from-server", []byte("Olala, message from server: "+strconv.Itoa(count)))
		if err != nil {
			log.Fatal("Cannot broadcast")
		}
		err = server.Send("client-a", "some-msg", []byte("This is special message for client-A: "+strconv.Itoa(count)))
		if err != nil {
			log.Fatal("Cannot send msg")
		}
		count++
	}
}
