package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/varunarora1606/Probo/internal/config"
	"github.com/varunarora1606/Probo/internal/database"
	"github.com/varunarora1606/Probo/internal/engine"
	"github.com/varunarora1606/Probo/internal/http/handlers/order"
	"github.com/varunarora1606/Probo/internal/middlewares"
	"github.com/varunarora1606/Probo/internal/models"
	"github.com/varunarora1606/Probo/internal/ws"
)

func main() {
	cfg := config.MustLoad()

	// Db setup
	database.Connect(cfg.DBUrl, cfg.RedisUrl)
	if err := database.DB.AutoMigrate(&models.User{}); err != nil {
		slog.Error("Failed to migrate database", "error", err.Error())
		os.Exit(1)
	}

	// Setup router
	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			// Allow all origins dynamically
			return true
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// router.POST("/api/v1/user/signup", user.Signup)
	// router.POST("/api/v1/user/login", user.Signin)
	// router.POST("/api/v1/user/logout", user.Logout)

	if cfg.Env == "dev" {
		router.POST("/api/v1/order/buy", middlewares.DevAuth(), order.BuyHandler)
		router.POST("/api/v1/order/sell", middlewares.DevAuth(), order.SellHandler)
		router.POST("/api/v1/order/on-ramp-inr", middlewares.DevAuth(), order.OnRampInrHandler)
		router.GET("/api/v1/order/inr-balance", middlewares.DevAuth(), order.GetInrBalanceHandler)
		router.GET("/api/v1/order/stock-balance", middlewares.DevAuth(), order.GetStockBalanceHandler)
		router.GET("/api/v1/order/me", middlewares.DevAuth(), order.GetMeHandler)
	} else {
		router.POST("/api/v1/order/buy", middlewares.VerifyJWT(cfg.ClerkPubKey), order.BuyHandler)
		router.POST("/api/v1/order/sell", middlewares.VerifyJWT(cfg.ClerkPubKey), order.SellHandler)
		router.POST("/api/v1/order/on-ramp-inr", middlewares.VerifyJWT(cfg.ClerkPubKey), order.OnRampInrHandler)
		router.GET("/api/v1/order/inr-balance", middlewares.VerifyJWT(cfg.ClerkPubKey), order.GetInrBalanceHandler)
		router.GET("/api/v1/order/stock-balance", middlewares.VerifyJWT(cfg.ClerkPubKey), order.GetStockBalanceHandler)
		router.GET("/api/v1/order/me", middlewares.VerifyJWT(cfg.ClerkPubKey), order.GetMeHandler)
	}

	router.POST("/api/v1/order/create-market", order.CreateMarketHandler)


	router.GET("/api/v1/order/orderbook", order.GetOrderBookHandler)
	router.GET("/api/v1/order/market", order.GetMarketHandler)
	router.GET("/api/v1/order/markets", order.GetMarketsHandler)
	router.GET("/ws", ws.WsHandler)

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Setup server
	server := http.Server{
		Addr:    cfg.Address,
		Handler: router,
	}

	slog.Info("Server started on", slog.String("Address", cfg.Address))

	done := make(chan os.Signal, 1)

	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go engine.Worker()
	go database.Worker()
	go func() {
		fmt.Println("Server started")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server", err.Error())
		}
	}()

	<-done

	slog.Info("Received shutdown signal, shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Error shutting down server", slog.String("error", err.Error()))
	} else {
		slog.Info("Server shutdown successfully")
	}

}
