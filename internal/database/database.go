package database

import (
	"errors"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
	"github.com/varunarora1606/Probo/internal/models"
	"golang.org/x/net/context"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB
var RClient *redis.Client
var Ctx = context.Background()

const (
	TradeStream = "DbAndWsStream"
	DBGroup     = "db-workers"
	WSGroup     = "ws-workers"
)

func createGroups() error {
	groups := []string{DBGroup, WSGroup}

	for _, group := range groups {
		err := RClient.XGroupCreateMkStream(
			Ctx,
			TradeStream,
			group,
			"$",
		).Err()

		if err != nil && !errors.Is(err, redis.Nil) {
			if err.Error() == "BUSYGROUP Consumer Group name already exists" {
				continue
			}
			return err
		}
	}

	return nil
}

func Connect(db_url string, redis_url string) {
	dsn := db_url

	var err error

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		log.Fatal("Database error:", err.Error()) //It should be Panic
	}

	fmt.Println("🚀 Connected to PostgreSQL successfully!")

	err = DB.AutoMigrate(&models.Order{}, &models.Market{}, &models.InrBalance{}, &models.StockBalance{})
	if err != nil {
		log.Fatal("Failed to auto-migrate:", err)
	}

	opt, err := redis.ParseURL(redis_url)
	if err != nil {
		panic(err)
	}

	RClient = redis.NewClient(opt)

	fmt.Println("🚀 Connected to Redis successfully!")

	if err = createGroups(); err != nil {
		panic(err)
	}
}
