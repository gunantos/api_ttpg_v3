package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: wsclient <ws-url> [timeout]")
		os.Exit(1)
	}
	url := os.Args[1]
	timeout := 15 * time.Second
	if len(os.Args) > 2 {
		if d, err := time.ParseDuration(os.Args[2]); err == nil {
			timeout = d
		}
	}
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		fmt.Println("DIAL_ERROR:", err)
		os.Exit(2)
	}
	defer conn.Close()
	fmt.Println("CONNECTED")
	conn.SetReadDeadline(time.Now().Add(timeout))
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("READ_END")
			return
		}
		var ev struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}
		if json.Unmarshal(msg, &ev) == nil {
			fmt.Printf("EVENT type=%s data=%s\n", ev.Type, string(ev.Data))
		} else {
			fmt.Printf("RAW %s\n", msg)
		}
	}
}
