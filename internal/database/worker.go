package database

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/varunarora1606/Probo/internal/models"
	"github.com/varunarora1606/Probo/internal/types"
	"gorm.io/gorm"
)

var batchSize int = 20
var blockTime time.Duration = 5 * time.Second

func Worker() {
	for {
		// TODO: XREADGROUP with > won't automatically retry the failed messages.
		// First I should implement XAUTOCLAIM to recover messages from crashed workers
		// And next stream should not be read till this stream is persisted as there might be some orders which are dependent on 1st stream orders. (Like matching of 2nd stream order with 1st stream open order.)

		result, err := RClient.XReadGroup(Ctx, &redis.XReadGroupArgs{
			Group:    DBGroup,
			Consumer: "db-worker-1", // I think this should not be hard coded as we can scale this also with multiple workers
			Streams:  []string{TradeStream, ">"},
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

		// Batch Process (if 1 fails all 20 message streams are failled to maintain atomicity in txn.)
		if err := processBatch(stream.Messages); err != nil {
			fmt.Println("Batch processing failed:", err)
			// No ack here.
			continue
		}

		// Acknowledge (After db commits every msg)
		ids := make([]string, 0, len(stream.Messages))

		for _, id := range stream.Messages {
			ids = append(ids, id.ID)
		}

		if err := RClient.XAck(Ctx, TradeStream, DBGroup, ids...).Err(); err != nil {
			// TODO: If db succeds but ack only fails then what. So I should create a database table for processed redis message ids. So that first we check that and then just ack that and move forward if it already exist. 
			fmt.Println("XACK failed:", err)
		}

	}

}

func processBatch(messages []redis.XMessage) error {
	var deltas []types.Delta

	for _, message := range messages {
		msg, ok := message.Values["events"].(string)
		if !ok {
			return fmt.Errorf("missing events in message %s", message.ID)
		}

		var orderEvents []types.Delta

		if err := json.Unmarshal([]byte(msg), &orderEvents); err != nil {
			return fmt.Errorf("error during unmarshalling in 'output': %v", err)

		}
		deltas = append(deltas, orderEvents...)
	}

	// TODO: Check from here.

	var openOrders []types.Order = []types.Order{}
	var updateOrders []types.Order = []types.Order{}
	var matchedOrders []types.Order = []types.Order{}
	for _, delta := range deltas {
		switch delta.Msg {
		case "open":
			openOrders = append(openOrders, delta.Order)
		case "update":
			updateOrders = append(updateOrders, delta.Order)
		case "matched":
			matchedOrders = append(matchedOrders, delta.Order)
		default:
			fmt.Println("Default delta:", delta.Msg)
		}
	}
	return DB.Transaction(func(tx *gorm.DB) error {

		if len(openOrders) > 0 {
			openOrders := convertTypesInModels(openOrders)
			if err := tx.Create(&openOrders).Error; err != nil {
				return fmt.Errorf("Insert failed : %w",
					err)
			}
		}
		if len(updateOrders) > 0 {
			updateOrders := convertTypesInModels(updateOrders)
			for _, order := range updateOrders {
				if err := applyUpdate(tx, order); err != nil {
					return fmt.Errorf(
						"update failed for bet %s: %w",
						order.BetId,
						err,
					)
				}

			}
		}
		if len(matchedOrders) > 0 {
			matchedOrders := convertTypesInModels(matchedOrders)
			for _, order := range matchedOrders {
				if err := applyMatch(tx, order); err != nil {
					return fmt.Errorf(
						"Match failed for bet %s: %w",
						order.BetId,
						err,
					)
				}
			}
		}
		return nil
	})
}

func applyUpdate(tx *gorm.DB, order models.Order) error {
	var oldOrder models.Order
	// First, fetch the existing order
	if err := tx.Where("bet_id = ?", order.BetId).First(&oldOrder).Error; err != nil {
		return err
	}

	if err := tx.Model(&models.Order{}).Where("bet_id = ?", order.BetId).Updates(models.Order{
		Quantity: order.Quantity,
		EventId:  order.EventId,
	}).Error; err != nil {
		fmt.Println("Error inserting orders:", err)
		return err
	}

	if order.TransactionType == types.Buy {
		if err := tx.Model(&models.InrBalance{}).
			Where("user_id = ?", order.UserId).
			Update("quantity", gorm.Expr("quantity - ?", order.Price*(oldOrder.Quantity-order.Quantity))).Error; err != nil {
			return err
		}

		// Second update
		var stock models.StockBalance
		if err := tx.FirstOrCreate(&stock, models.StockBalance{
			UserId: order.UserId,
			Symbol: order.Symbol,
		}).Error; err != nil {
			return err
		}

		if order.Side == types.Yes {
			stock.YesQty += oldOrder.Quantity - order.Quantity
		} else {
			stock.NoQty += oldOrder.Quantity - order.Quantity
		}

		if err := tx.Save(&stock).Error; err != nil {
			return err
		}

	} else {
		if err := tx.Model(&models.InrBalance{}).
			Where("user_id = ?", order.UserId).
			Update("quantity", gorm.Expr("quantity + ?", order.Price*(oldOrder.Quantity-order.Quantity))).Error; err != nil {
			return err
		}

		// Second update
		var stock models.StockBalance
		if err := tx.FirstOrCreate(&stock, models.StockBalance{
			UserId: order.UserId,
			Symbol: order.Symbol,
		}).Error; err != nil {
			return err
		}

		if order.Side == types.Yes {
			stock.YesQty -= oldOrder.Quantity - order.Quantity
		} else {
			stock.NoQty -= oldOrder.Quantity - order.Quantity
		}

		if err := tx.Save(&stock).Error; err != nil {
			return err
		}
	}
	return nil
}

func applyMatch(tx *gorm.DB, order models.Order) error {
	if err := tx.Where("bet_id = ?", order.BetId).Delete(&models.Order{}).Error; err != nil {
		return err
	}

	if order.TransactionType == types.Buy {
		if err := tx.Model(&models.InrBalance{}).
			Where("user_id = ?", order.UserId).
			Update("quantity", gorm.Expr("quantity - ?", order.Price*(order.Quantity))).Error; err != nil {
			return err
		}

		// Second update
		var stock models.StockBalance
		if err := tx.FirstOrCreate(&stock, models.StockBalance{
			UserId: order.UserId,
			Symbol: order.Symbol,
		}).Error; err != nil {
			return err
		}

		if order.Side == types.Yes {
			stock.YesQty += order.Quantity
		} else {
			stock.NoQty += order.Quantity
		}

		if err := tx.Save(&stock).Error; err != nil {
			return err
		}
	} else {
		if err := tx.Model(&models.InrBalance{}).
			Where("user_id = ?", order.UserId).
			Update("quantity", gorm.Expr("quantity + ?", order.Price*(order.Quantity))).Error; err != nil {
			return err
		}

		// Second update
		var stock models.StockBalance
		if err := tx.FirstOrCreate(&stock, models.StockBalance{
			UserId: order.UserId,
			Symbol: order.Symbol,
		}).Error; err != nil {
			return err
		}

		if order.Side == types.Yes {
			stock.YesQty -= order.Quantity
		} else {
			stock.NoQty -= order.Quantity
		}

		if err := tx.Save(&stock).Error; err != nil {
			return err
		}
	}

	return nil
}

func convertTypesInModels(typesOrders []types.Order) []models.Order {
	var modelOrders []models.Order
	for _, o := range typesOrders {
		modelOrders = append(modelOrders, models.Order{
			BetId:           o.BetId,
			EventId:         o.EventId,
			UserId:          o.UserID,
			Symbol:          o.Symbol,
			Side:            o.Side,
			Price:           o.Price,
			Quantity:        o.Quantity,
			TransactionType: o.TransactionType,
			CreatedAt:       o.CreatedAt,
		})
	}
	return modelOrders
}
