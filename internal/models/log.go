package models

import "time"

type Log struct {
	Service   string    `json:"service"`
	Message   string    `json:"message"`
	Level     string    `json:"level"`
	Timestamp time.Time `json:"timestamp"`
}
