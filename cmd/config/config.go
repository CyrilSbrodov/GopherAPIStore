package config

import (
	"flag"
	"fmt"
	"os"

	"github.com/caarlos0/env/v6"
)

type ServerConfig struct {
	Addr        string `env:"RUN_ADDRESS"`
	Accrual     string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	DatabaseURI string `env:"DATABASE_URI"`
	SessionKey  string `env:"SESSION_KEY"`
}

var cfgSrv ServerConfig

func ServerConfigInit() ServerConfig {
	flag.StringVar(&cfgSrv.Addr, "a", "localhost:8282", "ADDRESS")
	flag.StringVar(&cfgSrv.Accrual, "r", "localhost:8080", "ACCRUAL_SYSTEM_ADDRESS")
	flag.StringVar(&cfgSrv.SessionKey, "k", "secret", "session key")
	flag.StringVar(&cfgSrv.DatabaseURI, "d", "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable", "DATABASE_URI")
	flag.Parse()
	if err := env.Parse(&cfgSrv); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return cfgSrv
}
