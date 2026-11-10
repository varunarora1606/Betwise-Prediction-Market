package memory

import (
	"sync"

	"github.com/gorilla/websocket"
	"github.com/varunarora1606/Probo/internal/types"
)

type OrderBookType struct {
	Mu   sync.RWMutex
	Data map[string]types.StockBook
}
type InrBalanceType struct {
	Mu   sync.RWMutex
	Data map[string]types.Balance
}
type StockBalanceType struct {
	Mu   sync.RWMutex
	Data map[string]map[string]map[types.Side]types.Balance
}
type BetBookType struct {
	Mu   sync.RWMutex
	Data map[string]types.BetDetails
}

type MarketBookType struct {
	Mu   sync.RWMutex
	Data map[string]types.SymbolBook
}

var OrderBook = OrderBookType{
	Data: make(map[string]types.StockBook),
}

var InrBalance = InrBalanceType{
	Data: make(map[string]types.Balance),
}

var StockBalance = StockBalanceType{
	Data: make(map[string]map[string]map[types.Side]types.Balance),
}

var BetBook = BetBookType{
	Data: make(map[string]types.BetDetails),
}

var MarketBook = MarketBookType{
	Data: make(map[string]types.SymbolBook),
}

type ServerType struct {
	Mu sync.RWMutex
	Conns map[string]map[*websocket.Conn]bool
}

var Server = ServerType{
	Conns: make(map[string]map[*websocket.Conn]bool),
}