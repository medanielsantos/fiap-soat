package config

import "os"

const defaultHTTPAddress = ":8080"

type Config struct {
	HTTPAddress string
}

func Load() Config {
	address := os.Getenv("HTTP_ADDR")
	if address == "" {
		address = defaultHTTPAddress
	}
	return Config{HTTPAddress: address}
}
