package main

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/trunov/gophermart/internal/app/config"
	"github.com/trunov/gophermart/internal/app/handler"
	"github.com/trunov/gophermart/internal/app/postgres"
	"github.com/trunov/gophermart/logger"
	"github.com/trunov/gophermart/migrate"
)

func StartServer(cfg config.Config) {
	l := logger.Get()
	ctx := context.Background()

	var dbStorage postgres.DBStorager

	if cfg.DatabaseURI != "" {
		var err error
		dbpool, err := pgxpool.Connect(ctx, cfg.DatabaseURI)
		if err != nil {
			l.Fatal().Err(err)
		}
		defer dbpool.Close()

		dbStorage = postgres.NewDBStorage(dbpool)

		err = migrate.Migrate(cfg.DatabaseURI, migrate.Migrations)
		if err != nil {
			l.Fatal().
				Err(err).
				Msgf("Failed to run migrations.")
		}
	} else {
		l.Fatal().Msgf("Cannot start database. Please provide DatabaseURI")
	}

	h := handler.NewHandler(dbStorage, l)
	r := handler.NewRouter(h)

	l.Info().
		Msgf("Starting the Gophermart app server on address '%s'", cfg.RunAddress)

	l.Fatal().
		Err(http.ListenAndServe(cfg.RunAddress, r)).
		Msg("Gophermart App Server Closed")
}
