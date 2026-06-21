package models

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type Log struct {
	Service   string    `json:"service"`
	Message   string    `json:"message"`
	Level     string    `json:"level"`
	Timestamp time.Time `json:"timestamp"`
}

type LogRow struct {
	ID        int64     `json:"id"`
	MsgID     string    `json:"msg_id"`
	Service   string    `json:"service"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	NodeName  string    `json:"node_name"`
}

func XMessageToLog(msg redis.XMessage) (Log, error) {
	raw, ok := msg.Values["logData"].(string)
	if !ok {
		return Log{}, fmt.Errorf("type assertion failed for message %s", msg.ID)
	}

	var entry Log
	if err := json.Unmarshal([]byte(raw), &entry); err != nil {
		return Log{}, err
	}

	return entry, nil
}

func QueryBuilder(service, level, from, to string, limit, offset int) (string, []any) {
	query := `SELECT id, msg_id, service, level, message, timestamp, node_name FROM logs WHERE 1=1`
	args := []any{}
	i := 1

	if service != "" {
		query += ` AND service = $` + strconv.Itoa(i)
		args = append(args, service)
		i++
	}

	if level != "" {
		query += ` AND level = $` + strconv.Itoa(i)
		args = append(args, level)
		i++
	}

	if from != "" {
		t, err := time.Parse(time.RFC3339, from)
		if err == nil {
			query += ` AND timestamp >= $` + strconv.Itoa(i)
			args = append(args, t)
			i++
		}
	}

	if to != "" {
		t, err := time.Parse(time.RFC3339, to)
		if err == nil {
			query += ` AND timestamp <= $` + strconv.Itoa(i)
			args = append(args, t)
			i++
		}
	}

	query += ` ORDER BY timestamp DESC LIMIT $` + strconv.Itoa(i) + ` OFFSET $` + strconv.Itoa(i+1)
	args = append(args, limit, offset)

	return query, args
}
