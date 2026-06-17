package main

import (
	"context"
	"fmt"
	"os"

	shared "github.com/amitsuthar69/parsel/internal/consumer"

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
		fmt.Printf("Could not connect to Redis: %v", err)
		os.Exit(1)
	}

	handler := func(msg redis.XMessage) {}
	shared.StartConsumer(ctx, rdb, "parsel:logs", "group-1", "consumer-1", handler)
}
