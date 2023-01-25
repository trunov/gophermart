package main

import (
	"net/http"

	"github.com/trunov/gophermart/internal/app/config"
	"github.com/trunov/gophermart/internal/app/handler"
	"github.com/trunov/gophermart/internal/app/postgres"
	"github.com/trunov/gophermart/logger"
)

func StartServer(cfg config.Config, dbStorage postgres.DBStorager) {
	l := logger.Get()

	h := handler.NewHandler(dbStorage, l)
	r := handler.NewRouter(h)

	l.Info().
		Msgf("Starting the Gophermart app server on address '%s'", cfg.RunAddress)

	l.Fatal().
		Err(http.ListenAndServe(cfg.RunAddress, r)).
		Msg("Gophermart App Server Closed")
}
