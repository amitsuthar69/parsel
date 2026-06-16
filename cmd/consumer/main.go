package main

import (
	"context"
	"fmt"
	"os"

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

	readArgs := &redis.XReadGroupArgs{
		Streams:  []string{"parsel:logs", ">"},
		Group:    "group-1",
		Consumer: "consumer-1",
		Block:    0,
	}

	rdb.XGroupCreateMkStream(ctx, "parsel:logs", "group-1", "0")

	for {
		streams, err := rdb.XReadGroup(ctx, readArgs).Result()
		if err != nil {
			fmt.Printf("Failed to read XReadGroup: %v", err)
			continue
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {
				fmt.Printf("Processing message %s: %v\n", msg.ID, msg.Values)
				// deserialzie the Value into the json Log struct...

				rdb.XAck(ctx, "parsel:logs", "group-1", msg.ID)
			}
		}
	}
}
