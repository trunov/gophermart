package config

import (
	"flag"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	RunAddress           string `env:"RUN_ADDRESS" envDefault:"localhost:8080"`
	DatabaseURI          string `env:"DATABASE_URI"`
	AccrualSystemAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
}

func ReadConfig() (Config, error) {
	cfgEnv := Config{}

	if err := env.Parse(&cfgEnv); err != nil {
		return cfgEnv, err
	}

	cfgFlag := Config{}

	flag.StringVar(&cfgFlag.RunAddress, "a", cfgEnv.RunAddress, "run address")
	flag.StringVar(&cfgFlag.DatabaseURI, "d", cfgEnv.DatabaseURI, "database URI")
	flag.StringVar(&cfgFlag.AccrualSystemAddress, "r", cfgEnv.AccrualSystemAddress, "accrual system address")

	flag.Parse()

	return cfgFlag, nil
}
