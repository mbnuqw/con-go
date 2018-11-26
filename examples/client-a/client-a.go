package main

import (
	con "con/con-go"
	"fmt"
	"log"
	"time"
)

func main() {
	fmt.Println(" → Example of client")

	client := con.Client{}
	err := client.Connect("/tmp/con-examples.sock", "client-a")
	// err := client.Connect("127.0.0.1:1234", "client-a")
	if err != nil {
		log.Fatal("Cannot connect to server:\n    ", err)
	}

	client.On("disconnect", func(msg con.Msg) (ans []byte) {
		fmt.Println(" → DISCOnnect (simple on)")
		return
	})

	// Send message
	fmt.Println(" → Send 'msg-A' with body 'Just body'")
	err = client.Send("msg-A", []byte("Just body..."))
	if err != nil {
		fmt.Println(" → got error", err.Error() == "disconnected")
	}

	// Request
	fmt.Println(" → Request 'repeat' with body 'this'")
	ansChan := make(chan con.Msg)
	client.Req("repeat", []byte("this"), ansChan)
	answer := <-ansChan
	fmt.Println(" → Got answer", string(answer.Body))

	// Subscribe
	client.On("", func(msg con.Msg) (ans []byte) {
		fmt.Println(" → Got msg from server:", string(msg.Body))
		return
	})

	for i := 0; i < 100; i++ {
		time.Sleep(1000 * time.Millisecond)
		if i > 10 {
			client.Disconnect()
			break
		}
	}
}
