package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/amitsuthar69/parsel/internal/config"
	shared "github.com/amitsuthar69/parsel/internal/consumer"
	models "github.com/amitsuthar69/parsel/internal/models"

	"github.com/redis/go-redis/v9"
)

func createWebhookPayload(webhookURL, message string) []byte {
	if strings.Contains(webhookURL, "discord.com") {
		body, _ := json.Marshal(map[string]string{"content": message})
		return body
	}
	body, _ := json.Marshal(map[string]string{"text": message})
	return body
}

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

			body := createWebhookPayload(cfg.WebhookURL, fmt.Sprintf("%s : [%s] - %s", entry.Level, entry.Service, entry.Message))
			resp, err := http.Post(cfg.WebhookURL, "application/json", bytes.NewBuffer(body))
			if err != nil {
				log.Printf("failed to send webhook: %v", err)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode >= 300 {
				log.Printf("webhook rejected with status %d", resp.StatusCode)
			}
		}
	}

	log.Println("Alerter started, watching for ERROR logs...")
	shared.StartConsumer(ctx, rdb, cfg.StreamName, "alerter-group", "alerter-1", handler)
}
