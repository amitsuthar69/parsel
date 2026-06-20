package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"path/filepath"
	"time"

	config "github.com/amitsuthar69/parsel/internal/config"
	"github.com/amitsuthar69/parsel/internal/models"
	Log "github.com/amitsuthar69/parsel/internal/models"
)

func makeDummyLog() Log.Log {
	levels := []string{"INFO", "ERROR", "WARN"}
	services := []string{"auth", "payment", "inventory", "gateway"}

	randLvl := levels[rand.IntN(len(levels))]
	randService := services[rand.IntN(len(services))]

	return Log.Log{
		Service:   randService,
		Message:   fmt.Sprintf("log from %v", randService),
		Level:     randLvl,
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

func writeToLogFile(entry models.Log, logDir string) error {
	innerJSON, _ := json.Marshal(entry)

	outer := map[string]string{
		"log":    string(innerJSON),
		"stream": "stdout",
		"time":   entry.Timestamp.UTC().Format(time.RFC3339Nano),
	}

	line, err := json.Marshal(outer)
	if err != nil {
		return err
	}

	filePath := filepath.Join(logDir, entry.Service+".log")
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintln(f, string(line))
	return err
}

func main() {
	cfg := config.Load()

	logsChan := make(chan models.Log)
	go tickLogs(logsChan)

	log.Printf("producer started, writing to dir: %s", cfg.LogDir)

	for entry := range logsChan {
		if err := writeToLogFile(entry, cfg.LogDir); err != nil {
			log.Printf("failed to write log: %v", err)
			continue
		}
		log.Printf("[%s] %s | %s: %s", entry.Timestamp.Format("15:04:05"), entry.Level, entry.Service, entry.Message)
	}
}
