package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/trunov/gophermart/logger"
	"github.com/trunov/gophermart/migrate"

	"github.com/trunov/gophermart/internal/app/config"
	"github.com/trunov/gophermart/internal/app/postgres"
)

func main() {
	l := logger.Get()
	ctx := context.Background()

	cfg, err := config.ReadConfig()
	if err != nil {
		l.Fatal().
			Err(err).
			Msgf("Failed to read the config.")
	}

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

	workerpool := NewWorkerpool(&dbStorage, cfg.AccrualSystemAddress)

	inputCh := make(chan string)
	ticker := time.NewTicker(20 * time.Second)
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				return
			case t := <-ticker.C:
				// is not it better to pass inputCh into GetOrders ?
				orders, err := dbStorage.GetOrders(ctx)
				if err != nil {
					l.Err(err).
						Msg("Could not get orders for ticker")
				}

				for _, order := range orders {
					inputCh <- order.Number
				}

				fmt.Println("Tick at", t)
			}
		}
	}()

	go workerpool.Start(ctx, inputCh)

	StartServer(cfg, dbStorage)
}
