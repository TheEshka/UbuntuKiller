package worker

import (
	"context"
	"github.com/Shopify/sarama"
	"github.com/jmoiron/sqlx"
)

type Worker struct {
	groupID string
	topic string
	kafka sarama.Client
	db *sqlx.DB
	consumer sarama.ConsumerGroupHandler
}

func New(db *sqlx.DB, kafka sarama.Client, consumer sarama.ConsumerGroupHandler, topic string, groupID string) *Worker {
	return &Worker{
		groupID:  groupID,
		topic:    topic,
		kafka:    kafka,
		db:       db,
		consumer: consumer,
	}
}

func (w *Worker) Process(ctx context.Context) error {
	consumerGroup, err := sarama.NewConsumerGroupFromClient(w.groupID, w.kafka)
	if err != nil {
		return err
	}

	for {
		err = consumerGroup.Consume(ctx, []string{w.topic}, w.consumer)
		if err != nil {
			return err
		}
	}
}
