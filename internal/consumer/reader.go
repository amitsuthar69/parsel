package consumer

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

func StartConsumer(
	ctx context.Context,
	rdb *redis.Client,
	streamName string,
	groupName string,
	consumerName string,
	handlerFunc func(msg redis.XMessage)) error {

	readArgs := &redis.XReadGroupArgs{
		Streams:  []string{streamName, ">"},
		Group:    groupName,
		Consumer: consumerName,
		Block:    0,
	}

	rdb.XGroupCreateMkStream(ctx, streamName, groupName, "0")

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
