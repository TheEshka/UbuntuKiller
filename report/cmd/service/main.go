package main

import (
	"context"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/jmoiron/sqlx"
	"log"
	"net/http"
	"report/internal/worker"
	"report/internal/worker/books_genre"
	"report/internal/worker/books_return"
	"time"

	"github.com/antelman107/net-wait-go/wait"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	ServicePort      int    `default:"3000" split_words:"true"`
	ServiceSecret    string `required:"true" split_words:"true"`
	ServiceAuthMap   map[string]string `required:"true" split_words:"true"`
	DatabaseHost     string `default:"postgres" split_words:"true"`
	DatabasePort     int    `default:"5432" split_words:"true"`
	DatabaseUser     string `default:"postgres" split_words:"true"`
	DatabasePassword string `required:"true" split_words:"true"`
	DatabaseName     string `default:"report" split_words:"true"`
	KafkaHost        string `default:"kafka" split_words:"true"`
	KafkaPort        int    `default:"9092" split_words:"true"`
	KafkaGenresTopic string `default:"books-genres" split_words:"true"`
	KafkaReturnsTopic string `default:"books-returns" split_words:"true"`
	KafkaGenresDlqTopic string `default:"books-genres-dlq" split_words:"true"`
	KafkaReturnsDlqTopic string `default:"books-returns-dlq" split_words:"true"`
	KafkaGenresGroup string `default:"report-genres" split_words:"true"`
	KafkaReturnsGroup string `default:"report-returns" split_words:"true"`
}

func initDatabase(cfg Config) *sqlx.DB {
	if !wait.New(
		wait.WithProto("tcp"),
		wait.WithWait(200*time.Millisecond),
		wait.WithBreak(50*time.Millisecond),
		wait.WithDeadline(15*time.Second),
		wait.WithDebug(true),
	).Do([]string{fmt.Sprintf("%s:%d", cfg.DatabaseHost, cfg.DatabasePort)}) {
		log.Fatal("timeout waiting for database")
	}

	connConfig, err := pgx.ParseConfig(
		fmt.Sprintf(
			"postgres://%s:%s@%s:%d/%s",
			cfg.DatabaseUser,
			cfg.DatabasePassword,
			cfg.DatabaseHost,
			cfg.DatabasePort,
			cfg.DatabaseName,
		),
	)
	if err != nil {
		log.Fatalf("failed to parse pgx config: %v\n", err)
	}

	connConfig.ConnectTimeout = time.Minute

	connString := stdlib.RegisterConnConfig(connConfig)
	db, err := sqlx.Connect("pgx", connString)
	if err != nil {
		log.Fatalf("failed to connect to database: %v\n", err)
	}

	return db
}

func initKafka(cfg Config) sarama.Client {
	if !wait.New(
		wait.WithProto("tcp"),
		wait.WithWait(200*time.Millisecond),
		wait.WithBreak(50*time.Millisecond),
		wait.WithDeadline(15*time.Second),
		wait.WithDebug(true),
	).Do([]string{fmt.Sprintf("%s:%d", cfg.KafkaHost, cfg.KafkaPort)}) {
		log.Fatal("timeout waiting for kafka")
	}

	kafkaConfig := sarama.NewConfig()
	kafkaConfig.Producer.Return.Successes = true
	kafkaClient, err := sarama.NewClient([]string{fmt.Sprintf("%s:%d", cfg.KafkaHost, cfg.KafkaPort)}, kafkaConfig)
	if err != nil {
		log.Fatalf("failed to connect to kafka: %v\n", err)
	}

	return kafkaClient
}

func main() {
	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		log.Fatalf("failed to initialize env config: %v\n", err)
	}

	db := initDatabase(cfg)
	kafka := initKafka(cfg)

	genresConsumer, err := books_genre.New(db, kafka, cfg.KafkaGenresDlqTopic)
	if err != nil {
		log.Fatalf("failed to initialize genres consumer: %v\n", err)
	}

	genresWorker := worker.New(db, kafka, genresConsumer, cfg.KafkaGenresTopic, cfg.KafkaGenresGroup)
	go func() {
		err := genresWorker.Process(context.Background())
		if err != nil {
			log.Fatalf("failed to process in genres worker: %v\n", err)
		}
	}()

	returnsConsumer, err := books_return.New(db, kafka, cfg.KafkaReturnsDlqTopic)
	if err != nil {
		log.Fatalf("failed to initialize genres consumer: %v\n", err)
	}

	returnsWorker := worker.New(db, kafka, returnsConsumer, cfg.KafkaReturnsTopic, cfg.KafkaReturnsGroup)
	go func() {
		err := returnsWorker.Process(context.Background())
		if err != nil {
			log.Fatalf("failed to process in returns worker: %v\n", err)
		}
	}()

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))


	err = http.ListenAndServe(fmt.Sprintf(":%d", cfg.ServicePort), r)
	if err != nil {
		log.Fatalf("failed to serve: %v\n", err)
	}
}