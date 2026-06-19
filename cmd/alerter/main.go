package main

import (
	"context"
	"log"
	"os"

	config "github.com/amitsuthar69/parsel/internal/config"
	shared "github.com/amitsuthar69/parsel/internal/consumer"
	models "github.com/amitsuthar69/parsel/internal/models"

	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.Load()

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
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

		if entry.Level == "ERROR" {
			log.Printf("ALERT | %s | %s: %s",
				entry.Timestamp.Format("15:04:05"),
				entry.Service,
				entry.Message,
			)
		}
	}

	log.Println("Alerter started, watching for ERROR logs...")
	shared.StartConsumer(ctx, rdb, cfg.StreamName, "alerter-group", "alerter-1", handler)
}
