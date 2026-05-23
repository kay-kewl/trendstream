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

type ConsumerObserver interface {
	ObserveKafkaRecordsPolled(count int)
	ObserveKafkaRecordsCommitted(count int)
	ObserveKafkaFetchError(topic string, partition int32)
	ObserveKafkaCommitError()
	ObserveKafkaDecodeError()
}

type noopConsumerObserver struct{}

func (noopConsumerObserver) ObserveKafkaRecordsPolled(count int)                  {}
func (noopConsumerObserver) ObserveKafkaRecordsCommitted(count int)               {}
func (noopConsumerObserver) ObserveKafkaFetchError(topic string, partition int32) {}
func (noopConsumerObserver) ObserveKafkaCommitError()                             {}
func (noopConsumerObserver) ObserveKafkaDecodeError()                             {}

type Consumer struct {
	client    *kgo.Client
	cfg       ConsumerConfig
	processor EventProcessor
	logger    *slog.Logger
	observer  ConsumerObserver
}

func NewConsumer(
	cfg ConsumerConfig,
	processor EventProcessor,
	logger *slog.Logger,
	observers ...ConsumerObserver,
) (*Consumer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	if cfg.ClientID == "" {
		cfg.ClientID = "trendstream"
	}

	observer := ConsumerObserver(noopConsumerObserver{})
	if len(observers) > 0 && observers[0] != nil {
		observer = observers[0]
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
		observer:  observer,
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

		if fetchErrors := fetches.Errors(); len(fetchErrors) > 0 {
			for _, fetchErr := range fetchErrors {
				c.observer.ObserveKafkaFetchError(fetchErr.Topic, fetchErr.Partition)

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
		recordsPolled := 0

		fetches.EachRecord(func(record *kgo.Record) {
			recordsPolled++
			c.handleRecord(ctx, record)
			recordsToCommit = append(recordsToCommit, record)
		})

		c.observer.ObserveKafkaRecordsPolled(recordsPolled)

		if len(recordsToCommit) == 0 {
			continue
		}

		if err := c.client.CommitRecords(ctx, recordsToCommit...); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}

			c.observer.ObserveKafkaCommitError()
			c.logger.Error("failed to commit kafka records", slog.Any("error", err))
			continue
		}

		c.observer.ObserveKafkaRecordsCommitted(len(recordsToCommit))
	}
}

func (c *Consumer) Close() {
	c.client.Close()
}

func (c *Consumer) handleRecord(ctx context.Context, record *kgo.Record) {
	event, err := DecodeSearchEvent(record.Value)
	if err != nil {
		c.observer.ObserveKafkaDecodeError()
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
