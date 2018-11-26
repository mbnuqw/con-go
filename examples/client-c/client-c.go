package main

import (
	con "con/con-go"
	"fmt"
	"log"
	"strconv"
	"time"
)

func main() {
	fmt.Println(" → Client 'C'")

	client := con.Client{}
	// err := client.Connect("/tmp/con-examples.sock", "client-c")
	err := client.Connect("127.0.0.1:1234", "client-c")
	if err != nil {
		log.Fatal("Cannot connect to server:\n    ", err)
	}

	// Request
	fmt.Println(" → Request 'repeat' with body 'this'")
	answers := make([]chan con.Msg, 10000)
	for i := range answers {
		answers[i] = make(chan con.Msg, 1)
	}
	startTS := time.Now()
	// for i := 0; i < 1000; i++ {
	for i := range answers {
		// client.Send("repeat", []byte(con.UID()))
		fmt.Println(" → Make req", i)
		client.Send("repeat", []byte(strconv.Itoa(i)))
		// <-answers[i]
		// time.Sleep(1 * time.Microsecond)
	}
	endTS := time.Now()
	fmt.Println(" → ", endTS.Sub(startTS))

	// for i := range answers {
	// 	<-answers[i]
	// }
	time.Sleep(100 * time.Millisecond)
}
