package main

import (
	"context"
	"log"
	"os"

	shared "github.com/amitsuthar69/parsel/internal/consumer"
	models "github.com/amitsuthar69/parsel/internal/models"

	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
		Protocol: 2,
	})
	defer rdb.Close()

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("Could not connect to Redis: %v", err)
		os.Exit(1)
	}

	handler := func(msg redis.XMessage) {
		entry, err := models.XMessageToLog(msg)
		if err != nil {
			log.Printf("unmarshal failed for message %s: %v", msg.ID, err)
			return
		}

		log.Printf("[%s] %s | %s: %s",
			entry.Timestamp.Format("15:04:05"),
			entry.Level,
			entry.Service,
			entry.Message,
		)
	}

	log.Println("Logger consumer started...")
	shared.StartConsumer(ctx, rdb, "parsel:logs", "logger-group", "logger-1", handler)
}
