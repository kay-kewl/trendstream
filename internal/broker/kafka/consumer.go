package kafka

import (
	"context"
	"errors"
	"log/slog"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/kay-kewl/trendstream/internal/contract"
	"github.com/kay-kewl/trendstream/internal/ingest"
)

type EventProcessor interface {
	Process(ctx context.Context, event contract.SearchEvent) ingest.Result
}

type Consumer struct {
	client    *kgo.Client
	cfg       ConsumerConfig
	processor EventProcessor
	logger    *slog.Logger
}

func NewConsumer(cfg ConsumerConfig, processor EventProcessor, logger *slog.Logger) (*Consumer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	if cfg.ClientID == "" {
		cfg.ClientID = "trendstream"
	}

	client, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.ConsumeTopics(cfg.Topic),
		kgo.ConsumerGroup(cfg.GroupID),
		kgo.ClientID(cfg.ClientID),
		kgo.DisableAutoCommit(),
	)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		client:    client,
		cfg:       cfg,
		processor: processor,
		logger:    logger,
	}, nil
}

func (c *Consumer) Run(ctx context.Context) error {
	c.logger.Info(
		"kafka consumer started",
		slog.String("topic", c.cfg.Topic),
		slog.String("group_id", c.cfg.GroupID),
		slog.Any("brokers", c.cfg.Brokers),
	)

	for {
		fetches := c.client.PollFetches(ctx)
		if fetches.IsClientClosed() {
			return nil
		}

		if err := ctx.Err(); err != nil {
			return nil
		}

		if errors := fetches.Errors(); len(errors) > 0 {
			for _, fetchErr := range errors {
				c.logger.Error(
					"kafka fetch error",
					slog.String("topic", fetchErr.Topic),
					slog.Int("partition", int(fetchErr.Partition)),
					slog.Any("error", fetchErr.Err),
				)
			}

			continue
		}

		recordsToCommit := make([]*kgo.Record, 0)

		fetches.EachRecord(func(record *kgo.Record) {
			c.handleRecord(ctx, record)
			recordsToCommit = append(recordsToCommit, record)
		})

		if len(recordsToCommit) == 0 {
			continue
		}

		if err := c.client.CommitRecords(ctx, recordsToCommit...); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}

			c.logger.Error("failed to commit kafka records", slog.Any("error", err))
		}
	}
}

func (c *Consumer) Close() {
	c.client.Close()
}

func (c *Consumer) handleRecord(ctx context.Context, record *kgo.Record) {
	event, err := DecodeSearchEvent(record.Value)
	if err != nil {
		c.logger.Warn(
			"dropped invalid kafka message",
			slog.String("topic", record.Topic),
			slog.Int("partition", int(record.Partition)),
			slog.Int64("offset", record.Offset),
			slog.Any("error", err),
		)

		return
	}

	result := c.processor.Process(ctx, event)
	if !result.Accepted {
		c.logger.Debug(
			"dropped search event",
			slog.String("reason", string(result.Reason)),
			slog.String("topic", record.Topic),
			slog.Int("partition", int(record.Partition)),
			slog.Int64("offset", record.Offset),
		)

		return
	}

	c.logger.Debug(
		"accepted search event",
		slog.String("topic", record.Topic),
		slog.Int("partition", int(record.Partition)),
		slog.Int64("offset", record.Offset),
		slog.Int64("count", result.Count),
	)
}
