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
		var openOrders []types.Order = []types.Order{}
		var updateOrders []types.Order = []types.Order{}
		var matchedOrders []types.Order = []types.Order{}
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

		var deltas []types.Delta

		for _, message := range stream.Messages {
			msg := message.Values["events"].(string)

			var orderEvents []types.Delta

			if err := json.Unmarshal([]byte(msg), &orderEvents); err != nil {
				fmt.Printf("error during unmarshalling in 'output': %v", err)
				continue
			}
			deltas = append(deltas, orderEvents...)
		}

		// TODO: Check from here.

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

		if len(openOrders) > 0 {
			openOrders := convertTypesInModels(openOrders)
			if err := DB.Create(&openOrders).Error; err != nil {
				fmt.Println("Error inserting orders:", err)
			} else {
				fmt.Println("Bulk insert successful")
			}
		}
		if len(updateOrders) > 0 {
			updateOrders := convertTypesInModels(updateOrders)
			for _, order := range updateOrders {
				// TODO: Check if the EventId are different before updating so that any 2 worker might not update it twice.
				err := DB.Transaction(func(tx *gorm.DB) error {
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
				})

				if err != nil {
					fmt.Println("error while updating order: ", order, "\nerror: ", err.Error())
				}
			}
		}
		if len(matchedOrders) > 0 {
			matchedOrders := convertTypesInModels(matchedOrders)
			for _, order := range matchedOrders {
				err := DB.Transaction(func(tx *gorm.DB) error {
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
				})

				if err != nil {
					fmt.Println("error while deleting order: ", order, "\nerror: ", err.Error())
				}
			}
		}
	}

	// switch delta.Msg {
	// case "open":
	// 	openOrders = append(openOrders, delta.Order)
	// case "update":
	// 	updateOrders = append(updateOrders, delta.Order)
	// case "matched":
	// 	matchedOrders = append(matchedOrders, delta.Order)
	// default:
	// 	fmt.Println("Default delta:", delta.Msg)
	// }

	// TODO: Continue from here.

	// for i := 0; i < batchSize; i++ {
	// 	data, err = RClient.RPop(Ctx, "order_events").Result()
	// 	if err == redis.Nil {
	// 		break
	// 	} else if err != nil {
	// 		fmt.Println("RPOP error:", err)
	// 		break
	// 	}
	// 	var delta types.Delta
	// 	if err := json.Unmarshal([]byte(data), &delta); err != nil {
	// 		fmt.Printf("error during unmarshalling of %s in 'order_events': %v", data, err)
	// 		continue
	// 	}

	// }
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
