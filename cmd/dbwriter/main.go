package main

import (
	"context"
	"log"

	config "github.com/amitsuthar69/parsel/internal/config"
	shared "github.com/amitsuthar69/parsel/internal/consumer"
	models "github.com/amitsuthar69/parsel/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"

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
		log.Fatalf("Could not connect to Redis: %v", err)
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Could not connect to Postgres: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Postgres ping failed: %v", err)
	}

	_, err = pool.Exec(ctx, `
	CREATE TABLE IF NOT EXISTS logs (
		id BIGSERIAL PRIMARY KEY,
		msg_id TEXT UNIQUE,
		service TEXT,
		level TEXT,
		message TEXT,
		timestamp TIMESTAMPTZ,
		node_name TEXT
	)
	`)
	if err != nil {
		log.Fatalf("Could not create logs table: %v", err)
	}

	handler := func(msg redis.XMessage) {
		entry, err := models.XMessageToLog(msg)
		if err != nil {
			log.Printf("unmarshal failed for message %s: %v", msg.ID, err)
			return
		}

		_, err = pool.Exec(ctx, `
		INSERT INTO logs (msg_id, service, level, message, timestamp, node_name)
		VALUES ($1, $2, $3, $4, $5, $6) 
		ON CONFLICT (msg_id) DO NOTHING`,
			msg.ID,
			entry.Service,
			entry.Level,
			entry.Message,
			entry.Timestamp,
			cfg.NodeName)

		if err != nil {
			log.Printf("db insert failed for message %s: %v", msg.ID, err)
			return
		}

		log.Printf("written to db: [%s] %s | %s: %s",
			entry.Timestamp.Format("15:04:05"),
			entry.Level,
			entry.Service,
			entry.Message,
		)
	}

	log.Println("db writer started...")
	shared.StartConsumer(ctx, rdb, cfg.StreamName, "db-group", "dbwriter-1", handler)
}
