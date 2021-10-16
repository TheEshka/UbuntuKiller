package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/go-chi/jwtauth/v5"
	"github.com/jmoiron/sqlx"
	"io/ioutil"
	"log"
	"net/http"
	"report/internal/common"
	"report/internal/handler/auth"
	"report/internal/handler/reports"
	"report/internal/worker"
	"report/internal/worker/books_genre"
	"report/internal/worker/books_return"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	ServicePort      int    `default:"9116" envconfig:"port"`
	ServiceSecret    string `required:"true" split_words:"true"`
	ServiceAuthMap   map[string]string `required:"true" split_words:"true"`

	DatabaseHost     string `default:"postgres" split_words:"true"`
	DatabasePort     int    `default:"5432" split_words:"true"`
	DatabaseUser     string `default:"postgres" split_words:"true"`
	DatabasePassword string `split_words:"true"`
	DatabaseName     string `default:"books" split_words:"true"`

	DatabaseUrl		string  `split_words:"true"`

	CloudkarafkaBrokers        string `default:"kafka:9092" split_words:"true"`
	KafkaSaslLogin   string `default:"njeb2phw" split_words:"true"`
	CloudkarafkaPassword string `required:"true" split_words:"true"`
	CloudkarafkaCa    string `split_words:"true"`
	KafkaGenresTopic string `default:"njeb2phw-books-genres" split_words:"true"`
	KafkaReturnsTopic string `default:"njeb2phw-books-returns" split_words:"true"`
	KafkaGenresDlqTopic string `default:"njeb2phw-books-genres-dlq" split_words:"true"`
	KafkaReturnsDlqTopic string `default:"njeb2phw-books-returns-dlq" split_words:"true"`
	KafkaGenresGroup string `default:"njeb2phw-report-genres" split_words:"true"`
	KafkaReturnsGroup string `default:"njeb2phw-report-returns" split_words:"true"`
}

func initDatabase(cfg Config) *sqlx.DB {
	//if !wait.New(
	//	wait.WithProto("tcp"),
	//	wait.WithWait(200*time.Millisecond),
	//	wait.WithBreak(50*time.Millisecond),
	//	wait.WithDeadline(15*time.Second),
	//	wait.WithDebug(true),
	//).Do([]string{fmt.Sprintf("%s:%d", cfg.DatabaseHost, cfg.DatabasePort)}) {
	//	log.Fatal("timeout waiting for database")
	//}

	var configStr string
	if cfg.DatabaseUrl == "" {
		configStr = fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
			cfg.DatabaseUser,
			cfg.DatabasePassword,
			cfg.DatabaseHost,
			cfg.DatabasePort,
			cfg.DatabaseName,
		)
	} else {
		configStr = cfg.DatabaseUrl
	}

	connConfig, err := pgx.ParseConfig(configStr)
	if err != nil {
		log.Fatalf("failed to parse pgx config: %v\n", err)
	}

	connConfig.ConnectTimeout = time.Minute

	connString := stdlib.RegisterConnConfig(connConfig)
	db, err := sqlx.Connect("pgx", connString)
	if err != nil {
		log.Fatalf("failed to connect to database: %v\n", err)
	}

	initSql, err := ioutil.ReadFile("./deployments/postgres/001-init.sql")
	if err != nil {
		log.Fatalf("failed to init database: %v\n", err)
	}
	requests := strings.Split(string(initSql), ";")
	for _, request := range requests {
		_, err := db.DB.Exec(request)
		if err != nil {
			log.Fatalf("failed to init database: %v\n", err)
		}
	}

	return db
}

func initKafka(cfg Config) sarama.Client {
	addresses := strings.Split(cfg.CloudkarafkaBrokers, ",")
	//for _, address := range addresses {
	//	hostPort := strings.Split(address, ":")
	//	if !wait.New(
	//		wait.WithProto("tcp"),
	//		wait.WithWait(200*time.Millisecond),
	//		wait.WithBreak(50*time.Millisecond),
	//		wait.WithDeadline(15*time.Second),
	//		wait.WithDebug(true),
	//	).Do([]string{fmt.Sprintf("%s:%s", hostPort[0], hostPort[1])}) {
	//		log.Fatal("timeout waiting for kafka")
	//	}
	//}

	//caCert, err := os.ReadFile("../report/deployments/kafkaCA.pem")
	//if err != nil {
	//	log.Fatal(err)
	//}
	caCert := []byte(cfg.CloudkarafkaCa)

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: false,
		ClientAuth: tls.NoClientCert,
	}

	kafkaConfig := sarama.NewConfig()
	kafkaConfig.Producer.Return.Successes = true
	kafkaConfig.Net.SASL.Enable = true
	kafkaConfig.Net.SASL.User = cfg.KafkaSaslLogin
	kafkaConfig.Net.SASL.Password = cfg.CloudkarafkaPassword
	kafkaConfig.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &common.XDGSCRAMClient{HashGeneratorFcn: common.SHA512} }
	kafkaConfig.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
	kafkaConfig.Net.SASL.Handshake = true
	kafkaConfig.Version = sarama.V2_5_0_0
	kafkaConfig.Metadata.Full = false
	kafkaConfig.Net.TLS.Enable = true
	kafkaConfig.Net.TLS.Config = tlsConfig
	kafkaClient, err := sarama.NewClient(addresses, kafkaConfig)
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

	handler := reports.New(db)

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	var tokenAuth = jwtauth.New("HS256", []byte(cfg.ServiceSecret), nil)
	authHandler := auth.New(tokenAuth)

	r.Group(func(r chi.Router) {
		r.Use(middleware.BasicAuth("services", cfg.ServiceAuthMap))
		r.Post("/auth", authHandler.Auth)
	})


	r.Group(func(r chi.Router) {
		r.Use(middleware.BasicAuth("services", cfg.ServiceAuthMap))
		r.Post("/auth", authHandler.Auth)
	})

	r.Group(func(r chi.Router) {
		r.Use(jwtauth.Verifier(tokenAuth))
		r.Use(jwtauth.Authenticator)

		r.Route("/reports", func(r chi.Router) {
			r.Get("/books-return", handler.ReturnsReport)
			r.Get("/books-genre", handler.GenresReport)
		})

	})


	err = http.ListenAndServe(fmt.Sprintf(":%d", cfg.ServicePort), r)
	if err != nil {
		log.Fatalf("failed to serve: %v\n", err)
	}
}