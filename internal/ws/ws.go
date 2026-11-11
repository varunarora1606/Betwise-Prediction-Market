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
	fmt.Println("Entered")
	memory.Server.Mu.Lock()
    defer memory.Server.Mu.Unlock()
	memory.Server.Conns[string(message)[10:]][conn] = true
	conn.WriteMessage(websocket.TextMessage, message)
}

func unsubscribeAll(conn *websocket.Conn) {
    memory.Server.Mu.Lock()
    defer memory.Server.Mu.Unlock()

    for symbol, conns := range memory.Server.Conns {
        delete(conns, conn)

        if len(conns) == 0 {
            delete(memory.Server.Conns, symbol)
        }
    }
}

func WsHandler(c *gin.Context) {
	log.Printf("ws entered")
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	defer unsubscribeAll(conn)
	defer conn.Close()

	for {
		msgType, message, err := conn.ReadMessage()

		if err != nil {
			fmt.Println("Error reading message:", err)
			break
		}

		if msgType == websocket.TextMessage && string(message)[:9] == "SUBSCRIBE" {
			memory.Server.Mu.Lock()
			if _, exists := memory.Server.Conns[string(message)[10:]]; !exists {
				// Temp way, Change it in future and add marketr from backend, do not allow user to create it
				fmt.Println("Creating market for ws layer")
				memory.Server.Conns[string(message)[10:]] = make(map[*websocket.Conn]bool)
			}
			if _, exists := memory.Server.Conns[string(message)[10:]][conn]; exists {
				fmt.Println("user is already subscribed to this market")
				continue
			}
			memory.Server.Mu.Unlock()
			go subscribe(conn, message)
		}
	}
}
