package models

import (
	"time"

	"github.com/varunarora1606/Probo/internal/types"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Name     string
	Email    string
	Password string
}

type Order struct {
	BetId           string                `gorm:"type:text;primaryKey"`
	EventId         string                `gorm:"type:text;not null;unique"` //Lets see if I use it
	UserId          string                `gorm:"type:text;not null"`
	Symbol          string                `gorm:"type:text;not null"`
	Side            types.Side            `gorm:"type:text;check:side IN ('yes','no');not null"`
	TransactionType types.TransactionType `gorm:"column:transaction_type;type:text;check:transaction_type In ('buy','sell');not null"`
	Price           int                   `gorm:"type:numeric"`
	Quantity        int                   `gorm:"type:numeric;not null"`
	// Status   string		`gorm:"type:text;check:status IN ('open','matched', 'cancelled')"`
	CreatedAt time.Time
}

type InrBalance struct {
	UserId   string `gorm:"type:text;primaryKey"`
	Quantity int    `gorm:"type:numeric;default:0"` //Locked + quantity
}

type StockBalance struct {
	UserId string `gorm:"type:text;primaryKey"`
	Symbol string `gorm:"type:text;primaryKey"`
	YesQty int    `gorm:"default:0"` //Locked + quantity
	NoQty  int    `gorm:"default:0"` //Locked + quantity
}

type Market struct {
	Symbol     string `gorm:"type:text;primaryKey"`
	Title      string `gorm:"type:text;not null"`
	Question   string `gorm:"type:text;not null"`
	EndTime    int64  `gorm:"type:bigint;not null"`
	YesClosing int    `gorm:"type:numeric;default:50"` // Dont use YesClosing instead store orders in matched, open, cancelled type with timestamp and there see the last closed order
	// Volume     int    `gorm:"type:numeric;default:0"`	 // Same calculate ut during seedingData time instead.
}

// type Order struct {
// 	types.Order
// }
