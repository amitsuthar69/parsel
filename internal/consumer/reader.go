package consumer

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

func StartConsumer(
	ctx context.Context,
	rdb *redis.Client,
	streamName string,
	groupName string,
	consumerName string,
	handlerFunc func(msg redis.XMessage)) {

	err := rdb.XGroupCreateMkStream(ctx, streamName, groupName, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		log.Fatalf("Failed to create consumer group: %v", err)
	}

	go recoverPending(ctx, rdb, streamName, groupName, consumerName, handlerFunc)

	readArgs := &redis.XReadGroupArgs{
		Streams:  []string{streamName, ">"},
		Group:    groupName,
		Consumer: consumerName,
		Block:    0,
	}

	for {
		streams, err := rdb.XReadGroup(ctx, readArgs).Result()
		if err != nil {
			log.Printf("Failed to read XReadGroup: %v", err)
			continue
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {
				handlerFunc(msg)
				if err := rdb.XAck(ctx, streamName, groupName, msg.ID).Err(); err != nil {
					log.Printf("Failed to ACK message %s: %v", msg.ID, err)
				}
			}
		}
	}
}

// PEL recovery, runs every 60s, reclaims messages stuck for >30s
func recoverPending(
	ctx context.Context,
	rdb *redis.Client,
	streamName string,
	groupName string,
	consumerName string,
	handlerFunc func(msg redis.XMessage)) {

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			claimArgs := &redis.XAutoClaimArgs{
				Stream:   streamName,
				Group:    groupName,
				Consumer: consumerName + "-recovery",
				MinIdle:  30 * time.Second,
				Start:    "0-0",
				Count:    100,
			}

			msgs, _, err := rdb.XAutoClaim(ctx, claimArgs).Result()
			if err != nil {
				log.Printf("[PEL recovery] XAutoClaim failed: %v", err)
				continue
			}

			if len(msgs) == 0 {
				continue
			}

			log.Printf("[PEL recovery] reclaiming %d stuck messages", len(msgs))
			for _, msg := range msgs {
				handlerFunc(msg)
				if err := rdb.XAck(ctx, streamName, groupName, msg.ID).Err(); err != nil {
					log.Printf("[PEL recovery] failed to ACK %s: %v", msg.ID, err)
				}
			}
		}
	}
}
