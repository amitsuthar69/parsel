package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	config "github.com/amitsuthar69/parsel/internal/config"
	models "github.com/amitsuthar69/parsel/internal/models"
	"github.com/fsnotify/fsnotify"
	"github.com/redis/go-redis/v9"
)

var cfg config.Config

type ContainerdLog struct {
	Log    string `json:"log"`
	Stream string `json:"stream"`
	Time   string `json:"time"`
}

func lineToLog(line string, filename string) (models.Log, error) {
	var cl ContainerdLog
	if err := json.Unmarshal([]byte(line), &cl); err != nil {
		return models.Log{}, err
	}

	service := strings.SplitN(filename, "-", 2)[0]

	t, err := time.Parse(time.RFC3339Nano, cl.Time)
	if err != nil {
		t = time.Now()
	}

	return models.Log{
		Service:   service,
		Message:   strings.TrimSpace(cl.Log),
		Level:     "INFO",
		Timestamp: t,
	}, nil
}

func pushToRedis(ctx context.Context, rdb *redis.Client, entry models.Log) {
	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		log.Printf("marshal error: %v", err)
		return
	}

	if err := rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: cfg.StreamName,
		ID:     "*",
		MaxLen: 10000,
		Approx: true,
		Values: map[string]any{
			"logData": string(jsonBytes),
		},
	}).Err(); err != nil {
		log.Printf("XAdd error: %v", err)
	}
}

func startWatcher(logFile string, rdb *redis.Client, ctx context.Context) {
	stat, err := os.Stat(logFile)
	var offset int64 = 0
	if err == nil {
		offset = stat.Size()
	}

	partial := []byte{}

	parts := strings.Split(logFile, "/")
	filename := parts[len(parts)-1]
	filename = strings.TrimSuffix(filename, ".log")

	readNew := func() {
		f, err := os.Open(logFile)
		if err != nil {
			return
		}
		defer f.Close()

		st, err := f.Stat()
		if err != nil {
			return
		}

		newSize := st.Size()
		if newSize <= offset {
			return
		}

		n64 := newSize - offset
		buf := make([]byte, int(n64))
		nread, _ := f.ReadAt(buf, offset)
		offset += int64(nread)

		partial = append(partial, buf[:nread]...)
		chunks := bytes.Split(partial, []byte{'\n'})

		for i := 0; i < len(chunks)-1; i++ {
			line := strings.TrimSpace(string(chunks[i]))
			if line == "" {
				continue
			}

			entry, err := lineToLog(line, filename)
			if err != nil {
				log.Printf("parse error for line %q: %v", line, err)
				continue
			}

			pushToRedis(ctx, rdb, entry)
			log.Printf("pushed | %s | %s: %s", entry.Level, entry.Service, entry.Message)
		}

		partial = []byte(chunks[len(chunks)-1])
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println("fsnotify.NewWatcher error:", err)
		return
	}

	if err := watcher.Add(logFile); err != nil {
		log.Printf("watcher.Add error for %s: %v", logFile, err)
		return
	}

	go func() {
		readNew()

		for {
			select {
			case ev := <-watcher.Events:
				if ev.Op&fsnotify.Write == fsnotify.Write {
					time.Sleep(20 * time.Millisecond)
					readNew()
				}
			case err := <-watcher.Errors:
				log.Println("watcher error:", err)
			}
		}
	}()
}

func watchDir(dirPath string, rdb *redis.Client, ctx context.Context) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println("fsnotify.NewWatcher error:", err)
		return
	}

	watcher.Add(dirPath)

	matches, err := filepath.Glob(filepath.Join(dirPath, "*.log"))
	if err != nil {
		log.Println("filepath glob error:", err)
		return
	}

	for _, logFile := range matches {
		go startWatcher(logFile, rdb, ctx)
	}

	go func() {
		for {
			select {
			case ev := <-watcher.Events:
				if ev.Op&fsnotify.Create == fsnotify.Create {
					if strings.HasSuffix(ev.Name, ".log") {
						log.Printf("new log file detected: %s", ev.Name)
						go startWatcher(ev.Name, rdb, ctx)
					}
				}
			case err := <-watcher.Errors:
				log.Println("dir watcher error:", err)
			}
		}
	}()
}

func main() {
	cfg = config.Load()
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: "",
		DB:       0,
		Protocol: 2,
	})
	defer rdb.Close()

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("could not connect to Redis: %v", err)
		os.Exit(1)
	}

	dirPath := cfg.LogDir
	if len(os.Args) > 1 {
		dirPath = os.Args[1]
	}

	log.Printf("agent started, watching dir: %s", dirPath)
	watchDir(dirPath, rdb, ctx)

	select {}
}
