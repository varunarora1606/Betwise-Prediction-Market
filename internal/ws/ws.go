package ws

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/varunarora1606/Probo/internal/memory"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func subscribe(conn *websocket.Conn, message []byte) {
	memory.Server.Conns[string(message)[10:]][conn] = true
	conn.WriteMessage(websocket.TextMessage, message)
}

func WsHandler(c *gin.Context) {
	log.Printf("ws entered")
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	defer conn.Close()

	for {
		msgType, message, err := conn.ReadMessage()

		if err != nil {
			fmt.Println("Error reading message:", err)
			break
		}

		if msgType == websocket.TextMessage && string(message)[:9] == "SUBSCRIBE" {
			if _, exists := memory.Server.Conns[string(message)[10:]]; !exists {
				// Temp way, Change it in future and add marketr from backend, do not allow user to create it
				fmt.Println("Creating market for ws layer")
				memory.Server.Conns[string(message)[10:]] = make(map[*websocket.Conn]bool)
			}
			if _, exists := memory.Server.Conns[string(message)[10:]][conn]; exists {
				fmt.Println("user is already subscribed to this market")
				continue
			}
			go subscribe(conn, message)
		}
	}
}
