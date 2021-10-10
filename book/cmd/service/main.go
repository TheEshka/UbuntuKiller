package main

import (
	"book/internal/handler/author"
	"book/internal/handler/book"
	"fmt"
	"github.com/jmoiron/sqlx"
	"log"
	"net/http"
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
	DatabaseHost     string `default:"postgres" split_words:"true"`
	DatabasePort     int    `default:"5432" split_words:"true"`
	DatabaseUser     string `default:"postgres" split_words:"true"`
	DatabasePassword string `required:"true" split_words:"true"`
	DatabaseName     string `default:"books" split_words:"true"`
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

func main() {
	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		log.Fatalf("failed to initialize env config: %v\n", err)
	}

	db := initDatabase(cfg)

	bookHandler := book.New(db)
	authorHandler := author.New(db)

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Route("/books", func(r chi.Router) {
		r.Get("/{bookUid}", bookHandler.GetBooksByUUID)
		r.Get("/", bookHandler.GetBooks)
		r.Post("/", bookHandler.CreateBook)
		r.Delete("/{bookUid}", bookHandler.DeleteBook)
	})

	r.Route("/author", func(r chi.Router) {
		r.Get("/{authorUid}", authorHandler.GetAuthor)
		r.Get("/{authorUid}/books", authorHandler.GetAuthorBooks)
	})

	err = http.ListenAndServe(fmt.Sprintf(":%d", cfg.ServicePort), r)
	if err != nil {
		log.Fatalf("failed to serve: %v\n", err)
	}
}