package main

import (
	"fmt"

	"golang.org/x/net/websocket"
)

func main() {
	origin := "http://localhost:3022/"
	url := "ws://localhost:3024/ws"

	numconns := 5000
	bufsz := 65536

	done := make(chan bool)

	for i := 0; i < numconns; i++ {
		go func(i int) {
			defer func() { done <- true }()
			ws, err := websocket.Dial(url, "", origin)
			if err != nil {
					fmt.Printf("err dialing: %v", err)
					return
			}
			var msg = make([]byte, bufsz)
			var n int

			for {
				if n, err = ws.Read(msg); err != nil {
					fmt.Printf("err, closing: %s", err)
					return
				}
				fmt.Printf("%d received: %d bytes\n", i, n)
			}
		}(i)
	}


	for i := 0; i < numconns; i++ {
		<-done
	}
}
