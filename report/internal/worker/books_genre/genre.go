package books_genre

import (
	"encoding/json"
	"github.com/Shopify/sarama"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
)

type Genre struct {
	Genre string `json:"genre"`
}

type Consumer struct {
	db *sqlx.DB
	kafka sarama.SyncProducer
	dlqTopic string
}

func New(db *sqlx.DB, kafka sarama.Client, dlqTopic string) (*Consumer, error) {
	producer, err := sarama.NewSyncProducerFromClient(kafka)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		db: db,
		kafka: producer,
		dlqTopic: dlqTopic,
	}, nil
}

func (c *Consumer) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (c *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (c *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		var genre Genre

		err := json.Unmarshal(message.Value, &genre)
		if err != nil {
			log.Printf("books_genre consumer: invalid message: %v error: %v\n", message.Value, err)
			c.SendToDlq(message.Value)
			session.MarkMessage(message, "")
			continue
		}

		if genre.Genre == "" {
			log.Printf("books_genre consumer: empty genre for message with id: %v partition: %v\n", message.Offset, message.Partition)
			continue
		}

		insertGenreQuery := `INSERT INTO genres (genre) VALUES ($1);`
		_, err = c.db.Exec(insertGenreQuery, genre.Genre)
		if err != nil {
			log.Printf("books_genre consumer: failed to insert books_genre to db: %v\n", err)
			c.SendToDlq(message.Value)
			session.MarkMessage(message, "")
			continue
		}

		session.MarkMessage(message, "")
	}

	return nil
}

func (c *Consumer) SendToDlq(message []byte) {
	_, _, err := c.kafka.SendMessage(&sarama.ProducerMessage{
		Topic: c.dlqTopic,
		Value: sarama.StringEncoder(message),
	})
	if err != nil {
		log.Printf("error while producing to dlq: %v\n", err)
	}
}