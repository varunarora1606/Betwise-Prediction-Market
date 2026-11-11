package engine

import (
	"fmt"

	"github.com/varunarora1606/Probo/internal/memory"
	"github.com/varunarora1606/Probo/internal/types"
)

func CreateMarket(symbol string, title string, question string, endTime int64) error {
	memory.OrderBook.Mu.Lock()
	memory.MarketBook.Mu.Lock()
	defer memory.MarketBook.Mu.Unlock()
	defer memory.OrderBook.Mu.Unlock()

	_, exist := memory.MarketBook.Data[symbol]
	if exist {
		return fmt.Errorf("symbol's market already exists")
	}

	if _, exist := memory.OrderBook.Data[symbol]; exist {
		return fmt.Errorf("symbol's market already exists")
	}

	memory.MarketBook.Data[symbol] = types.SymbolBook{
		Title: title,
		Question: question,
		EndTime:  endTime,
		YesClosing: 50,
	}

	memory.OrderBook.Data[symbol] = types.StockBook{
		Yes: make(map[int]types.OrderDetails),
		No:  make(map[int]types.OrderDetails),
	}

	return nil
}

func OnRampInr(userId string, quantity int) types.Balance {
	memory.InrBalance.Mu.Lock()
	defer memory.InrBalance.Mu.Unlock()

	userBalance := memory.InrBalance.Data[userId]
	userBalance.Quantity += quantity
	memory.InrBalance.Data[userId] = userBalance

	return userBalance
}

func seedStockBalance(userId string, symbol string, yesQty int, noQty int) {
	memory.StockBalance.Mu.Lock()
	defer memory.StockBalance.Mu.Unlock()

	// Ensure top-level map is initialized
	if memory.StockBalance.Data == nil {
		memory.StockBalance.Data = make(map[string]map[string]map[types.Side]types.Balance)
	}

	// Ensure user map exists
	if memory.StockBalance.Data[userId] == nil {
		memory.StockBalance.Data[userId] = make(map[string]map[types.Side]types.Balance)
	}

	// Ensure symbol map exists
	if memory.StockBalance.Data[userId][symbol] == nil {
		memory.StockBalance.Data[userId][symbol] = make(map[types.Side]types.Balance)
	}

	userBalance := memory.StockBalance.Data[userId]
	userBalance[symbol][types.Yes] = types.Balance{
		Quantity: yesQty,
	}
	userBalance[symbol][types.No] = types.Balance{
		Quantity: noQty,
	}
	memory.StockBalance.Data[userId] = userBalance
}
