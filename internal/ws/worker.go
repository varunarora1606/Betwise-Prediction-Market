package ws

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/varunarora1606/Probo/internal/database"
	"github.com/varunarora1606/Probo/internal/memory"
	"github.com/varunarora1606/Probo/internal/types"
)

// Check These 2 values
var batchSize int = 20
var blockTime time.Duration = 5 * time.Second

func Worker() {
	for {

		result, err := database.RClient.XReadGroup(database.Ctx, &redis.XReadGroupArgs{
			Group:    database.WSGroup,
			Consumer: "ws-worker-1",
			Streams:  []string{database.TradeStream, ">"},
			Count:    int64(batchSize),
			Block:    blockTime,
		}).Result()
		// result, err := RClient.BRPop(Ctx, 0, "order_events").Result()
		if err != nil {
			if err == redis.Nil {
				// No new messages.
				continue
			}

			fmt.Println("Error reading from Redis Stream: ", err)
			time.Sleep(time.Second)
			continue
		}

		if len(result) == 0 {
			continue
		}
		stream := result[0]

		var deltas []types.Delta

		for _, message := range stream.Messages {
			msg, ok := message.Values["events"].(string)
			if !ok {
				// TODO: Handle Error
				fmt.Println("missing events in message ", message.ID)
				continue
			}

			var orderEvents []types.Delta

			if err := json.Unmarshal([]byte(msg), &orderEvents); err != nil {
				// TODO: Handle Error
				fmt.Println("error during unmarshalling in 'output': ", err)
				continue

			}
			deltas = append(deltas, orderEvents...)
		}
		// fmt.Println("WS: ", deltas)

		for _, delta := range deltas {
			memory.Server.Mu.RLock()
			conns, exists := memory.Server.Conns[delta.Order.Symbol]
			if !exists {
				fmt.Println("Currently no user is listening to this market")
				memory.Server.Mu.RUnlock()
				continue
			}
			fmt.Println(conns)
			for conn, _ := range conns {
				conn.WriteJSON(delta)
			}
			memory.Server.Mu.RUnlock()
		}

		// Acknowledge
		ids := make([]string, 0, len(stream.Messages))

		for _, id := range stream.Messages {
			ids = append(ids, id.ID)
		}

		if err := database.RClient.XAck(database.Ctx, database.TradeStream, database.WSGroup, ids...).Err(); err != nil {
			fmt.Println("XACK failed:", err)
		}

	}
}
