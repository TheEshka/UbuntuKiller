package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"gateway/internal/common"
	"gateway/internal/handler/gateway"
	"github.com/Shopify/sarama"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	ServicePort      int    `default:"9117" envconfig:"port"`

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

func initKafka(cfg Config) sarama.Client {
	addresses := strings.Split(cfg.CloudkarafkaBrokers, ",")

	//caCert, err := os.ReadFile("../gateway/deployments/kafkaCA.pem")
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

	var hosts gateway.Services
	err = envconfig.Process("", &hosts)
	if err != nil {
		log.Fatalf("failed to initialize env config: %v\n", err)
	}

	kafka := initKafka(cfg)
	producer, err := sarama.NewAsyncProducerFromClient(kafka)
	if err != nil {
		log.Fatalf("failed to initialize kafka producer: %v\n", err)
	}

	gatewayHandler := gateway.New(hosts, &producer)

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// to session service
	r.Group(func(r chi.Router) {
		r.Get("/users", gatewayHandler.ProxyHandler(hosts.SessionService))
		r.Post("/users", gatewayHandler.ProxyHandler(hosts.SessionService))
		r.Delete("/users", gatewayHandler.ProxyHandler(hosts.SessionService))
	})

	r.Route("/library", func(r chi.Router) {
		r.Route("/{libraryUid}", func(r chi.Router) {
			r.Get("/books", gatewayHandler.LibBooks)        // 1 - ???????????? ???????? ?? ????????????????????

			r.Group(func(r chi.Router) {
				r.Use(gatewayHandler.AdminChecker)
				r.Post("/book/{bookUid}", gatewayHandler.ProxyHandler(hosts.LibraryService))   // 11 - ???????????????? ?????????? ?? ????????????????????
				r.Delete("/book/{bookUid}", gatewayHandler.ProxyHandler(hosts.LibraryService)) // 12 - ???????????? ?????????? ???? ????????????????????
			})

			r.Group(func(r chi.Router) {
				r.Use(gatewayHandler.AuthChecker)
				r.Post("/book/{bookUid}/take", gatewayHandler.ProxyHandler(hosts.LibraryService))         // 7 - ?????????? ?????????? ?? ????????????????????
				r.Post("/book/{bookUid}/books_return", gatewayHandler.ProxyHandler(hosts.LibraryService)) // 8 - ?????????????? ??????????
			})
		})
		r.Group(func(r chi.Router) {
			r.Use(gatewayHandler.AuthChecker)
			r.Get("/book/{bookUid}", gatewayHandler.ProxyHandler(hosts.LibraryService))       // 6 - ?????????? ?????????? ?? ????????????????????
			r.Get("/user/{userUid}/books", gatewayHandler.TakenBooks) // 13 - ???????????????????? ???????????? ???????????? ????????
		})
	})

	r.Route("/books", func(r chi.Router) {
		r.Get("/{bookUid}", gatewayHandler.ProxyHandler(hosts.BookService)) // 2 - ???????? ?? ??????????
		r.Get("/", gatewayHandler.ProxyHandler(hosts.BookService)) // 3 - ?????????? ???? ???????????????? ??????????
		r.Group(func(r chi.Router) {
			r.Use(gatewayHandler.AdminChecker)
			r.Post("/", gatewayHandler.ProxyHandler(hosts.BookService)) // 9 ???????????????? ??????????
			r.Delete("/{bookUid}", gatewayHandler.ProxyHandler(hosts.BookService)) // 10 ?????????????? ??????????
		})
	})

	r.Route("/author", func(r chi.Router) {
		r.Get("/{authorUid}", gatewayHandler.ProxyHandler(hosts.BookService)) // 4 - ???????? ???? ????????????
		r.Get("/{authorUid}/books", gatewayHandler.ProxyHandler(hosts.BookService)) // 5 ???????? ???? ????????????
	})

	r.Group(func(r chi.Router) {
		r.Use(gatewayHandler.AdminChecker)
		r.Get("/reports/books-return", gatewayHandler.ProxyHandler(hosts.ReportService)) // 15 ???????????????????? ????????????????
		r.Get("/reports/books-genre", gatewayHandler.ProxyHandler(hosts.ReportService)) // 16 ???????????????????? ????????????
	})


	err = http.ListenAndServe(fmt.Sprintf(":%d", cfg.ServicePort), r)
	if err != nil {
		log.Fatalf("failed to serve: %v\n", err)
	}
}