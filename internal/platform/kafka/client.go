package kafka

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/platform/config"
	"github.com/rs/zerolog"
	segmentio "github.com/segmentio/kafka-go"
)

// Client encapsulates Kafka producer and consumer lifecycle.
type Client struct {
	config config.KafkaConfig
	log    *zerolog.Logger
	writer *segmentio.Writer
}

// New initializes a Kafka client with connection pooling and producer writer.
func New(cfg config.KafkaConfig, logger *zerolog.Logger) (*Client, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("no kafka brokers configured")
	}

	writer := &segmentio.Writer{
		Addr:         segmentio.TCP(cfg.Brokers...),
		Balancer:     &segmentio.LeastBytes{},
		BatchSize:    1000,
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: segmentio.RequireOne,
		MaxAttempts:  3,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		Async:        true,
	}

	client := &Client{
		config: cfg,
		log:    logger,
		writer: writer,
	}

	logger.Info().Strs("brokers", cfg.Brokers).Str("default_topic", cfg.Topic).Msg("kafka client initialized")
	return client, nil
}

// Publish publishes a message with key and value to a specified topic.
// If topic is empty, the default topic from configuration is used.
func (c *Client) Publish(ctx context.Context, topic string, key []byte, value []byte) error {
	targetTopic := topic
	if targetTopic == "" {
		targetTopic = c.config.Topic
	}

	msg := segmentio.Message{
		Topic: targetTopic,
		Key:   key,
		Value: value,
		Time:  time.Now(),
	}
	return c.writer.WriteMessages(ctx, msg)
}

// PublishBatch publishes a slice of messages in a single high-throughput batch.
func (c *Client) PublishBatch(ctx context.Context, msgs []segmentio.Message) error {
	if len(msgs) == 0 {
		return nil
	}
	return c.writer.WriteMessages(ctx, msgs...)
}

// NewConsumer creates a segmentio Reader for consuming events from a topic.
func (c *Client) NewConsumer(topic string, groupID string) *segmentio.Reader {
	targetTopic := topic
	if targetTopic == "" {
		targetTopic = c.config.Topic
	}
	if groupID == "" {
		groupID = c.config.GroupID
	}
	return segmentio.NewReader(segmentio.ReaderConfig{
		Brokers:        c.config.Brokers,
		Topic:          targetTopic,
		GroupID:        groupID,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		CommitInterval: 1 * time.Second,
		StartOffset:    segmentio.LastOffset,
	})
}

// Ping checks TCP connectivity to the first configured Kafka broker.
func (c *Client) Ping(ctx context.Context) error {
	if len(c.config.Brokers) == 0 {
		return fmt.Errorf("no brokers configured")
	}
	d := net.Dialer{Timeout: 3 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", c.config.Brokers[0])
	if err != nil {
		return err
	}
	return conn.Close()
}

// Close gracefully closes the Kafka producer writer.
func (c *Client) Close() error {
	if c.writer != nil {
		c.log.Info().Msg("closing kafka writer")
		return c.writer.Close()
	}
	return nil
}
