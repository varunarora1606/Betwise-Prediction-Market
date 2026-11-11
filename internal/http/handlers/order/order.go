package order

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/varunarora1606/Probo/internal/database"
	"github.com/varunarora1606/Probo/internal/models"
	"github.com/varunarora1606/Probo/internal/types"
)

type HandlerReq struct {
	Symbol    string          `json:"symbol" binding:"required"`        //symbol
	Quantity  int             `json:"quantity" binding:"required,gt=0"` //Greater than 0 check
	Price     int             `json:"price"`
	StockSide types.Side      `json:"stockSide" binding:"required,oneof=yes no"`       //side
	StockType types.OrderType `json:"stockType" binding:"required,oneof=market limit"` //ordertype
}

type CreateMarketHandlerReq struct {
	Title     	 string `json:"title" binding:"required"`
	Question     string `json:"question" binding:"required"`
	EndTimeMilli int64  `json:"endTime" binding:"required,gt=0"`
	Symbol       string `json:"symbol" binding:"required"`
}

func GetUserID(c *gin.Context) string {
	userId, _ := c.Get("userId")
	if userIdStr, ok := userId.(string); ok {
		return userIdStr
	}
	fmt.Println("Failed to retrieve userId from context")
	c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
	return ""
}

func BuyHandler(c *gin.Context) {
	var req HandlerReq

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Validation error", "error": err.Error()})
		return
	}

	if req.StockType == types.Limit && (req.Price < 1 || req.Price > 99) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Validation error", "error": "invalid price"})
		return
	}

	result, err := worker(types.Input{
		Fnx:             "order_engine",
		Symbol:          req.Symbol,
		StockSide:       types.Side(req.StockSide),
		Price:           req.Price,
		UserId:          GetUserID(c),
		Quantity:        req.Quantity,
		StockType:       types.OrderType(req.StockType),
		TransactionType: types.Buy,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Error", "error": err.Error()})
		return
	}
	if result.Err != "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Error", "error": result.Err})
		return
	}

	if req.StockType == types.Market {
		c.JSON(http.StatusCreated, gin.H{
			"message": "Buy request completed",
			"data": gin.H{
				"trades": result.Trade.MicroTrades,
			},
		})
	} else {
		c.JSON(http.StatusCreated, gin.H{"message": "Buy request completed", "data": gin.H{
			"completed": result.Trade.TotalQuantity,
			"pending":   req.Quantity - result.Trade.TotalQuantity,
		},
		})
	}
}

func SellHandler(c *gin.Context) {
	var req HandlerReq

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Validation error", "error": err.Error()})
		return
	}

	result, err := worker(types.Input{
		Fnx:             "order_engine",
		Symbol:          req.Symbol,
		StockSide:       types.Side(req.StockSide),
		Price:           req.Price,
		UserId:          GetUserID(c),
		Quantity:        req.Quantity,
		StockType:       types.OrderType(req.StockType),
		TransactionType: types.Sell,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Error", "error": err.Error()})
		return
	}
	if result.Err != "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Error", "error": result.Err})
		return
	}

	fmt.Println(result.Trade)

	if req.StockType == types.Market {
		c.JSON(http.StatusCreated, gin.H{
			"message": "Sell request completed",
			"data": gin.H{
				"trades": result.Trade.MicroTrades,
			},
		})
	} else {
		c.JSON(http.StatusCreated, gin.H{"message": "Sell request completed", "data": gin.H{
			"completed": result.Trade.TotalQuantity,
			"pending":   req.Quantity - result.Trade.TotalQuantity,
		},
		})
	}
}

func GetMarketHandler(c *gin.Context) {
	var req struct {
		Symbol string `json:"symbol" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Validation error", "error": err.Error()})
		return
	}

	result, err := worker(types.Input{
		Fnx:    "get_market",
		Symbol: req.Symbol,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Error", "error": err.Error()})
		return
	}
	if result.Err != "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Error", "error": result.Err})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Market fetched successfully",
		"data":    gin.H{"market": result.Market},
	})

}

func GetMarketsHandler(c *gin.Context) {
	result, err := worker(types.Input{
		Fnx: "get_markets",
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Error", "error": err.Error()})
		return
	}
	if result.Err != "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Error", "error": result.Err})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Markets fetched successfully",
		"data":    gin.H{"markets": result.Markets},
	})
}

func GetOrderBookHandler(c *gin.Context) {
	symbol := c.Query("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Missing symbol"})
		return
	}

	result, err := worker(types.Input{
		Fnx:    "get_orderbook",
		Symbol: symbol,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Error", "error": err.Error()})
		return
	}
	if result.Err != "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Error", "error": result.Err})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "OrderBook successfully",
		"data":    gin.H{"orderBook": result.StockBook},
	})
}

func CreateMarketHandler(c *gin.Context) {
	var req CreateMarketHandlerReq

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Validation error", "error": err.Error()})
		return
	}

	endTime := time.Unix(0, req.EndTimeMilli*int64(time.Millisecond))
	serverTime := time.Now()

	if endTime.Before(serverTime.Add(5 * time.Second)) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Error", "error": "endTime should be in far future"})
		return
	}

	if result := database.DB.First(&models.Market{}, "symbol = ?", req.Symbol); result.Error == nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Error", "error": "market alreasy exists"})
		return
	}

	market := models.Market{ // TODO: Add userId and superAdmin support
		Symbol: req.Symbol,
		Title: req.Title,
		Question: req.Question,
		EndTime: endTime.UnixNano(),
	}

	if err := database.DB.Create(&market).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Error", "error": "error while creating the market"})
		return
	}

	result, err := worker(types.Input{
		Fnx:      "create_market",
		Symbol:   req.Symbol,
		Title: req.Title,
		Question: req.Question,
		EndTime:  endTime.UnixNano(),
	})
	
	if err != nil || (result.Err != "" && result.Err != "symbol's market already exists") {
		if err := database.DB.Where("symbol = ?", req.Symbol).Delete(&models.Market{}).Error; err != nil{
			c.JSON(http.StatusBadRequest, gin.H{"message": "Error", "error": "error while undoing the market creation in db"})
			return // It a glitch and it should not happen at any case (TODO: Much better way is to do the same thing with in-memory rather than db and if db failes revert back in-memory data (You can use dissolve market functionality which you will make to distribute prizes))
		}
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Error", "error": err.Error()})
		return
	}
	if result.Err != "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Error", "error": result.Err})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Market created successfully",
	})
}

func OnRampInrHandler(c *gin.Context) {
	var req struct {
		Quantity int `json:"quantity" binding:"required,gt=0"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Validation error", "error": err.Error()})
		return
	}

	var inrBalance models.InrBalance

	if result := database.DB.First(&inrBalance, "user_id = ?", GetUserID(c)); result.Error != nil {
		inrBalance = models.InrBalance{
			UserId: GetUserID(c),
			Quantity: req.Quantity,
		}
		if err := database.DB.Create(&inrBalance).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Error", "error": "error while creating the account for user"})
			return
		}
	} else {
		if err := database.DB.Model(&models.InrBalance{}).Where("user_id = ?", GetUserID(c)).Updates(models.InrBalance{
			Quantity: inrBalance.Quantity + req.Quantity,
		}).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Error", "error": "error while updating inrBalnace in DB"})
			return
		}
	}

	result, err := worker(types.Input{
		Fnx:      "on_ramp_inr",
		Quantity: req.Quantity,
		UserId:   GetUserID(c),
	})

	// TODO: If Db fails here then it is really bad as then market in memory will have money without actually deducting money from bank and adding to db. If db fails then in memory inr value should revert back. (Think about it)
	if err != nil || result.Err != "" {
		if err := database.DB.Where("user_id = ?", GetUserID(c)).Delete(&models.InrBalance{}).Error; err != nil{
			c.JSON(http.StatusBadRequest, gin.H{"message": "Error", "error": "error while undoing the balance transaction in db"})
			return // It a glitch and it should not happen at any case (TODO: Much better way is to do the same thing with in-memory rather than db and if db failes revert back in-memory data (You can use dissolve market functionality which you will make to distribute prizes))
		}
	}
	
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Error", "error": err.Error()})
		return
	}
	if result.Err != "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Error", "error": result.Err})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Inr balance ramped successfully",
		"data":    gin.H{"balance": result.InrBalance},
	})
}

func GetInrBalanceHandler(c *gin.Context) {

	result, err := worker(types.Input{
		Fnx:    "get_inr_balance",
		UserId: GetUserID(c),
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Error", "error": err.Error()})
		return
	}
	if result.Err != "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Error", "error": result.Err})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Inr balance fetched successfully",
		"data":    gin.H{"balance": result.InrBalance},
	})
}

func GetStockBalanceHandler(c *gin.Context) {

	result, err := worker(types.Input{
		Fnx:    "get_stock_balance",
		UserId: GetUserID(c),
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Error", "error": err.Error()})
		return
	}
	if result.Err != "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Error", "error": result.Err})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Stock balance fetched successfully",
		"data":    gin.H{"balance": result.StockBalance},
	})
}

func GetMeHandler(c *gin.Context) {

	result, err := worker(types.Input{
		Fnx:    "get_me",
		UserId: GetUserID(c),
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Error", "error": err.Error()})
		return
	}
	if result.Err != "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Error", "error": result.Err})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Stock balance fetched successfully",
		"data":    gin.H{"inrBalance": result.InrBalance, "portfolioItems": result.PortfolioItems},
	})

}

func CancelBuyOrderHandler(c *gin.Context) {

}

func CancelSellOrderHandler(c *gin.Context) {

}
