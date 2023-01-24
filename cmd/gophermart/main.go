package main

import (
	"github.com/trunov/gophermart/logger"

	"github.com/trunov/gophermart/internal/app/config"
)

func main() {
	l := logger.Get()

	// ticker := time.NewTicker(500 * time.Millisecond)
	// done := make(chan bool)

	// go func() {
	// 	for {
	// 		select {
	// 		case <-done:
	// 			return
	// 		case t := <-ticker.C:
	// 			fmt.Println("Tick at", t)
	// 		}
	// 	}
	// }()

	cfg, err := config.ReadConfig()
	if err != nil {
		l.Fatal().
			Err(err).
			Msgf("Failed to read the config.")
	}

	StartServer(cfg)
}
