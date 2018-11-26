package main

import (
	con "con/con-go"
	"fmt"
	"time"
)

func main() {
	fmt.Println(" → Example of client")

	client := con.Client{}
	err := client.Connect("/tmp/con-examples.sock", "client-b")
	// err := client.Connect("127.0.0.1:1234", "client-b")
	if err != nil {
		fmt.Println(" → Cannot connect to server", err)
	}

	// Send messages
	client.Send("msg-A", nil)
	client.Send("another msg", []byte("with body"))

	// Subscribe
	client.On("msg-from-server", func(msg con.Msg) (ans []byte) {
		fmt.Println(" → Got msg from server:", string(msg.Body))
		return
	})

	for {
		time.Sleep(1000 * time.Millisecond)
	}
}
