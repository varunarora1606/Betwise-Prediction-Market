package types

import (
	"time"
)

type Side string

const (
	Yes Side = "yes"
	No  Side = "no"
)

type OrderType string

const (
	Limit  OrderType = "limit"
	Market OrderType = "market"
)

type TransactionType string

const (
	Buy  TransactionType = "buy"
	Sell TransactionType = "sell"
)

type OrderDetails struct {
	Total  int      `json:"total"`
	Orders []string `json:"orders"` // int to BetDetails
}
type StockBook struct {
	Yes map[int]OrderDetails `json:"yes"`
	No  map[int]OrderDetails `json:"no"`
}

type Balance struct {
	Quantity int `json:"quantity"`
	Locked   int `json:"locked"`
}

type BetDetails struct {
	UserId          string          `json:"userId"`
	Price           int             `json:"price"`
	Quantity        int             `json:"quantity"`
	Side            Side            `json:"side"`
	TransactionType TransactionType `json:"transactionType"`
}

type SymbolBook struct {
	Title      string
	Question   string
	EndTime    int64
	YesClosing int
	Volume     int
}

type Delta struct {
	Msg   string // remove, update, add  // Partially filled/ Fully filled/Unfilled (market)
	Order Order
}

type MicroTrade struct {
	Quantity int `json:"quantity"`
	Price    int `json:"price"`
}

type Trade struct {
	TotalQuantity int
	MicroTrades   []MicroTrade
}

type Input struct {
	ApiId           string //It should be InputId
	Fnx             string
	UserId          string
	Symbol          string
	Title           string
	Question        string
	EndTime         int64
	Quantity        int
	Price           int
	StockSide       Side
	StockType       OrderType
	TransactionType TransactionType
}

type Output struct {
	ForWs          bool
	ApiId          string
	Err            string
	Market         SymbolBook
	Markets        map[string]SymbolBook
	StockBook      StockBook
	InrBalance     Balance
	StockBalance   map[string]map[Side]Balance
	PortfolioItems []PortfolioItem
	Deltas         []Delta
	Trade          Trade
}

type PortfolioItem struct {
	Title    string `json:"title"`
	Symbol   string `json:"symbol"`
	Value    int    `json:"value"`
	Quantity int    `json:"quantity"`
}

type Order struct {
	BetId           string `gorm:"primaryKey"` //BetId
	EventId         string
	UserID          string
	Symbol          string
	Side            Side
	TransactionType TransactionType
	Price           int
	Quantity        int
	// Status   string
	CreatedAt time.Time
}
