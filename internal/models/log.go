package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Log struct {
	Service   string    `json:"service"`
	Message   string    `json:"message"`
	Level     string    `json:"level"`
	Timestamp time.Time `json:"timestamp"`
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
