package main

import (
	"github.com/trunov/gophermart/logger"

	"github.com/trunov/gophermart/internal/app/config"
)

func main() {
	l := logger.Get()

	cfg, err := config.ReadConfig()
	if err != nil {
		l.Fatal().
			Err(err).
			Msgf("Failed to read the config.")
	}

	StartServer(cfg)
}
