package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	consumerGroup = "events-service"
	commitTimeout = 5 * time.Second
)

type eventConsumer interface {
	PollRecords(context.Context, int) kgo.Fetches
	CommitRecords(context.Context, ...*kgo.Record) error
	AllowRebalance()
}

func consumeEvents(ctx context.Context, consumer eventConsumer, logger *slog.Logger) error {
	for ctx.Err() == nil {
		if err := consumeNext(ctx, consumer, logger); err != nil && ctx.Err() == nil {
			return err
		}
	}
	return nil
}

func consumeNext(ctx context.Context, consumer eventConsumer, logger *slog.Logger) error {
	fetches := consumer.PollRecords(ctx, 1)
	defer consumer.AllowRebalance()
	if ctx.Err() != nil {
		return nil
	}
	if fetches.IsClientClosed() {
		return kgo.ErrClientClosed
	}
	fetches.EachError(func(topic string, partition int32, err error) {
		logger.Warn("fetch Kafka events", "topic", topic, "partition", partition, "error", err)
	})

	records := fetches.RecordIter()
	if records.Done() {
		return nil
	}
	record := records.Next()
	logger = logger.With("topic", record.Topic, "partition", record.Partition, "offset", record.Offset)
	var metadata struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal(record.Value, &metadata); err != nil ||
		metadata.ID == "" || metadata.Type+"-events" != record.Topic {
		logger.Warn("skip Kafka record with invalid event metadata")
	} else {
		logger.Info("event processed", "event_id", metadata.ID, "type", metadata.Type)
	}

	commitCtx, cancel := context.WithTimeout(ctx, commitTimeout)
	defer cancel()
	if err := consumer.CommitRecords(commitCtx, record); err != nil {
		return fmt.Errorf("commit Kafka offset for %s partition %d offset %d: %w", record.Topic, record.Partition, record.Offset, err)
	}
	return nil
}
