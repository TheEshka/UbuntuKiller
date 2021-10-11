package main

import (
	"fmt"
	"gateway/internal/handler/gateway"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	ServicePort      int    `default:"3000" envconfig:"port"`
	ServiceHostsMap	 map[string]string `required:"true" split_words:"true"`
}

func main() {
	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		log.Fatalf("failed to initialize env config: %v\n", err)
	}

	gatewayHandler := gateway.New()

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Group(func(r chi.Router) {
		r.Route("/rating", func(r chi.Router) {
			r.Route("/{userUid}", func(r chi.Router) {
				r.Get("/", gatewayHandler.GetUserRating)
				r.Post("/up", gatewayHandler.GetUserRating)
				r.Post("/down", gatewayHandler.GetUserRating)
			})
		})
	})

	err = http.ListenAndServe(fmt.Sprintf(":%d", cfg.ServicePort), r)
	if err != nil {
		log.Fatalf("failed to serve: %v\n", err)
	}
}