package engine

import (
	"fmt"

	"github.com/varunarora1606/Probo/internal/database"
	"github.com/varunarora1606/Probo/internal/models"
	"github.com/varunarora1606/Probo/internal/types"
)

func seedEngine() {
	var markets []models.Market
	if result := database.DB.Find(&markets); result.Error != nil {
		panic("Failed to seed engine")
	}
	for _, market := range markets {
		if err := CreateMarket(market.Symbol, market.Title, market.Question, market.EndTime); err != nil {
			fmt.Println("Error while seeding market to the engine: ", market)
			fmt.Println("Error: ", err.Error())
		}
	}


	var inrBalances []models.InrBalance
	if result := database.DB.Find(&inrBalances); result.Error != nil {
		panic("Failed to seed engine")
	}
	for _, inrBalance := range inrBalances {
		OnRampInr(inrBalance.UserId, inrBalance.Quantity)
	}

	
	var stockBalances []models.StockBalance
	if result := database.DB.Find(&stockBalances); result.Error != nil {
		panic("Failed to seed engine")
	}
	for _, stockBalance := range stockBalances {
		seedStockBalance(stockBalance.UserId, stockBalance.Symbol, stockBalance.YesQty, stockBalance.NoQty)
	}


	var orders []models.Order
	if result := database.DB.Find(&orders); result.Error != nil {
		panic("Failed to seed engine")
	}
	for _, order := range orders {
		if _, _, err := OrderEngine(order.Symbol, order.Side, order.Price, order.UserId, order.Quantity, types.Limit, order.TransactionType, order.BetId); err != nil {
			fmt.Println("Error while seeding order to the engine: ", order)
			fmt.Println("Error: ", err.Error())
		}
	}
}