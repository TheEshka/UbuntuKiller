package main

import (
	"fmt"
	"github.com/go-chi/jwtauth"
	"github.com/jmoiron/sqlx"
	"io/ioutil"
	"log"
	"net/http"
	"rating/internal/handler/auth"
	"rating/internal/handler/rating"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	ServicePort      int    `default:"3000" envconfig:"port"`
	ServiceAuthMap   map[string]string `required:"true" split_words:"true"`
	ServiceSecret    string `required:"true" split_words:"true"`

	DatabaseHost     string `default:"ec2-52-209-171-51.eu-west-1.compute.amazonaws.com" split_words:"true"`
	DatabasePort     int    `default:"5432" split_words:"true"`
	DatabaseUser     string `default:"gmdbadwdavugqv" split_words:"true"`
	DatabasePassword string `split_words:"true"`
	DatabaseName     string `default:"dbsa8lnk31u8fj" split_words:"true"`

	DatabaseUrl		string  `split_words:"true"`
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
		println(request)
		_, err := db.DB.Exec(request)
		if err != nil {
			log.Fatalf("failed to init database: %v\n", err)
		}

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

	libraryHandler := rating.New(db)

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
		r.Use(jwtauth.Verifier(tokenAuth))
		r.Use(jwtauth.Authenticator)

		r.Route("/rating", func(r chi.Router) {
			r.Route("/{userUid}", func(r chi.Router) {
				r.Get("/", libraryHandler.GetUserRating)
				r.Post("/up", libraryHandler.IncUserRate)
				r.Post("/down", libraryHandler.DecUserRate)
			})
		})
	})

	err = http.ListenAndServe(fmt.Sprintf(":%d", cfg.ServicePort), r)
	if err != nil {
		log.Fatalf("failed to serve: %v\n", err)
	}
}