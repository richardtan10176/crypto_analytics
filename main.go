package main

import (
	"fmt"
	"log"

	"github.com/gorilla/websocket"
)


func main() {
	url := "wss://stream.binance.com:9443/ws/btcusdt@trade"

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer conn.Close()

	fmt.Println("Connected to", url)

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("read error:", err)
			return
		}
		fmt.Println(string(msg))
	}
}
