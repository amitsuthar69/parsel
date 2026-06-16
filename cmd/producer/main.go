package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"time"

	Log "github.com/amitsuthar69/parsel/internal/models"
	"github.com/redis/go-redis/v9"
)

func makeDummyLog() Log.Log {
	randInt := rand.IntN(100)
	return Log.Log{
		Service:   "auth",
		Message:   fmt.Sprintf("user%d logged in", randInt),
		Level:     "INFO",
		Timestamp: time.Now(),
	}
}

func tickLogs(logsChan chan Log.Log) {
	ticker := time.NewTicker(time.Second * 3)
	defer ticker.Stop()

	for range ticker.C {
		logsChan <- makeDummyLog()
	}
}

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

	logsChan := make(chan Log.Log)
	go tickLogs(logsChan)

	for log := range logsChan {
		jsonLog, err := json.Marshal(log)
		if err != nil {
			fmt.Printf("Failed to marshal log: %v", err)
			continue
		}

		streamArgs := &redis.XAddArgs{
			Stream: "parsel:logs",
			ID:     "*",
			Values: map[string]any{
				"logData": string(jsonLog),
			},
		}

		id, err := rdb.XAdd(ctx, streamArgs).Result()
		if err != nil {
			fmt.Printf("Failed to put log on stream: %v", err)
			continue
		} else {
			fmt.Printf("[%s] Pushed to Stream (ID: %s): %s\n",
				time.Now().Format("15:04:05"), id, log.Message)
		}
	}
}
