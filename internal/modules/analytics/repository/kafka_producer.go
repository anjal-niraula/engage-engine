package repository

import (
	"context"
	"encoding/json"

	"github.com/anjal-niraula/engage-engine/internal/modules/analytics/domain"
	platformKafka "github.com/anjal-niraula/engage-engine/internal/platform/kafka"
)

type KafkaViewProducer struct {
	client *platformKafka.Client
	topic  string
}

func NewKafkaViewProducer(client *platformKafka.Client, topic string) *KafkaViewProducer {
	if topic == "" {
		topic = "engage.views"
	}
	return &KafkaViewProducer{
		client: client,
		topic:  topic,
	}
}

func (p *KafkaViewProducer) PublishViewEvent(ctx context.Context, event domain.ViewEvent) error {
	if p.client == nil {
		return nil
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.client.Publish(ctx, p.topic, []byte(event.PostID.String()), payload)
}
