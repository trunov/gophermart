package main

import (
	"context"
	"fmt"
	"time"

	"github.com/trunov/gophermart/logger"

	"github.com/trunov/gophermart/internal/app/config"
	"github.com/trunov/gophermart/internal/app/repo"
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

	dbStorage, dbpool, err := repo.CreateRepo(ctx, cfg)
	if err != nil {
		l.Fatal().
			Err(err).
			Msgf("Error occurred while repository was initiating.")
	}
	defer dbpool.Close()

	workerpool := NewWorkerpool(&dbStorage, cfg.AccrualSystemAddress)

	inputCh := make(chan string)
	ticker := time.NewTicker(5 * time.Second)
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
