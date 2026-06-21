package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/amitsuthar69/parsel/internal/config"
	"github.com/amitsuthar69/parsel/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

var pool *pgxpool.Pool

func handleQuery(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	service := q.Get("service")
	level := q.Get("level")
	from := q.Get("from")
	to := q.Get("to")
	limitStr := q.Get("limit")
	offsetStr := q.Get("offset")

	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	offset := 0
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o > 0 {
			offset = o
		}
	}

	query, args := models.QueryBuilder(service, level, from, to, limit, offset)

	rows, err := pool.Query(r.Context(), query, args...)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	logs := []models.LogRow{}
	for rows.Next() {
		var row models.LogRow
		if err := rows.Scan(
			&row.ID,
			&row.MsgID,
			&row.Service,
			&row.Level,
			&row.Message,
			&row.Timestamp,
			&row.NodeName,
		); err != nil {
			continue
		}
		logs = append(logs, row)
	}

	writeJSON(w, http.StatusOK, logs)
}

func getServicesHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := pool.Query(r.Context(), "SELECT DISTINCT service FROM logs ORDER BY service")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	services := []string{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err == nil {
			services = append(services, s)
		}
	}

	writeJSON(w, http.StatusOK, services)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func main() {
	cfg := config.Load()
	ctx := context.Background()

	var err error
	pool, err = pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("could not connect to Postgres: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Postgres ping failed: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("./web")))
	mux.HandleFunc("/api/logs", handleQuery)
	mux.HandleFunc("/api/services", getServicesHandler)

	fmt.Println("API server listening on: ", cfg.ApiAddr)
	if err := http.ListenAndServe(cfg.ApiAddr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
