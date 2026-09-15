package config

import "os"

const connStr = "postgres://postgres:postgres@localhost:5432/message_api"

type Env struct {
	PORT         string
	POSTGRES_URL string
}

func Load() *Env {
	return &Env{
		PORT:         getEnv("PORT", "8081"),
		POSTGRES_URL: getEnv("POSTGRES_URL", connStr),
	}
}

func getEnv(key, def string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return def
}
