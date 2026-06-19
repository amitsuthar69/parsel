package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	config "github.com/amitsuthar69/parsel/internal/config"
	shared "github.com/amitsuthar69/parsel/internal/consumer"
	models "github.com/amitsuthar69/parsel/internal/models"
	"github.com/gorilla/websocket"

	"github.com/redis/go-redis/v9"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func handleConn(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error while upgrading connection:", err)
		return
	}
	defer conn.Close()

	log.Printf("Client %v connected", conn.RemoteAddr().String())
	hub.register <- conn

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			hub.unregister <- conn
			break
		}
	}
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

	hub := NewHub()
	go hub.Run()

	handler := func(msg redis.XMessage) {
		entry, err := models.XMessageToLog(msg)
		if err != nil {
			log.Printf("unmarshal failed for message %s: %v", msg.ID, err)
			return
		}

		jsonBytes, err := json.Marshal(entry)
		if err != nil {
			log.Printf("Couldn't marshal message to json: %v", err)
			return
		}

		hub.broadcast <- jsonBytes
	}

	go shared.StartConsumer(ctx, rdb, cfg.StreamName, "wsgateway-group", "wsgateway-1", handler)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleConn(hub, w, r)
	})

	log.Println("WebSocket gateway listening on :8080")
	if err := http.ListenAndServe(cfg.WSAddr, nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
