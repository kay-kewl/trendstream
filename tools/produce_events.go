package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/kay-kewl/trendstream/internal/contract"
	"github.com/kay-kewl/trendstream/internal/normalize"
)

var defaultQueries = []string{
	"iphone 15",
	"кроссовки женские",
	"ноутбук",
	"платье летнее",
	"рюкзак",
	"наушники bluetooth",
	"детские игрушки",
	"кофемашина",
	"смартфон samsung",
	"куртка мужская",
}

func main() {
	var (
		brokersRaw string
		topic      string
		rate       int
		duration   time.Duration
		clientID   string
	)

	flag.StringVar(&brokersRaw, "brokers", "localhost:9092", "comma-separated Kafka brokers")
	flag.StringVar(&topic, "topic", "search-events", "Kafka topic")
	flag.IntVar(&rate, "rate", 100, "events per second")
	flag.DurationVar(&duration, "duration", 30*time.Second, "produce duration")
	flag.StringVar(&clientID, "client-id", "trendstream-producer", "Kafka client id")
	flag.Parse()

	if rate <= 0 {
		log.Fatalf("rate must be positive")
	}

	brokers := splitCSV(brokersRaw)
	if len(brokers) == 0 {
		log.Fatalf("at least one broker is required")
	}

	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ClientID(clientID),
	)
	if err != nil {
		log.Fatalf("failed to create kafka client: %v", err)
	}
	defer client.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if duration > 0 {
		var cancelByTimeout context.CancelFunc
		ctx, cancelByTimeout = context.WithTimeout(ctx, duration)
		defer cancelByTimeout()
	}

	interval := time.Second / time.Duration(rate)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	startedAt := time.Now()
	produced := 0

	for {
		select {
		case <-ctx.Done():
			elapsed := time.Since(startedAt).Round(time.Millisecond)
			log.Printf("stopped producer: produced=%d elapsed=%s", produced, elapsed)
			return

		case now := <-ticker.C:
			event := generateEvent(produced, now.UTC())
			payload, err := json.Marshal(event)
			if err != nil {
				log.Fatalf("failed to marshal event: %v", err)
			}

			key, ok := normalize.Query(event.Query)
			if !ok {
				key = event.Query
			}

			record := &kgo.Record{
				Topic: topic,
				Key:   []byte(key),
				Value: payload,
			}

			if err := client.ProduceSync(ctx, record).FirstErr(); err != nil {
				log.Fatalf("failed to produce event: %v", err)
			}

			produced++
		}
	}
}

func generateEvent(index int, now time.Time) contract.SearchEvent {
	query := defaultQueries[rand.IntN(len(defaultQueries))]
	userHash := hashString(fmt.Sprintf("user-%d", rand.IntN(1000)))

	return contract.SearchEvent{
		SchemaVersion: contract.SearchEventSchemaVersion,
		EventID:       fmt.Sprintf("event-%d-%d", now.UnixNano(), index),
		OccurredAt:    now,
		Query:         query,
		UserIDHash:    userHash,
		SessionID:     fmt.Sprintf("session-%d", rand.IntN(10_000)),
		DeviceIDHash:  hashString(fmt.Sprintf("device-%d", rand.IntN(2_000))),
		IPHash:        hashString(fmt.Sprintf("ip-%d", rand.IntN(5_000))),
		UserAgentHash: hashString(fmt.Sprintf("ua-%d", rand.IntN(100))),
		Region:        "local",
		Locale:        "ru-RU",
		Platform:      "web",
		IsBot:         false,
	}
}

func hashString(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:8])
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))

	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}

		values = append(values, value)
	}

	return values
}
