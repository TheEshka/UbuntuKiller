module session

// +heroku install ./cmd/service

// +heroku goVersion go1.16
go 1.16

require (
	github.com/Masterminds/squirrel v1.5.0 // indirect
	github.com/antelman107/net-wait-go v0.0.0-20210623112055-cf684aebda7b
	github.com/go-chi/chi/v5 v5.0.4
	github.com/go-chi/jwtauth/v5 v5.0.2
	github.com/jackc/pgx/v4 v4.13.0
	github.com/jmoiron/sqlx v1.3.4
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/lestrrat-go/jwx v1.2.6
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
)
