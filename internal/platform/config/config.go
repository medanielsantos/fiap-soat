package config

import (
	"bufio"
	"net/url"
	"os"
	"strings"
)

const defaultHTTPPort = "8080"

type Config struct {
	HTTPAddress string
	DatabaseURL string
}

func Load() Config {
	loadDotEnv(".env")
	port := os.Getenv("APP_HTTP_PORT")
	if port == "" {
		port = defaultHTTPPort
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = postgresURLFromEnvironment()
	}
	return Config{
		HTTPAddress: ":" + port,
		DatabaseURL: databaseURL,
	}
}

func postgresURLFromEnvironment() string {
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	name := os.Getenv("DB_NAME")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	if name == "" || user == "" {
		return ""
	}
	databaseURL := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   host + ":" + port,
		Path:   name,
	}
	query := url.Values{}
	sslMode := os.Getenv("DB_SSL_MODE")
	if sslMode == "" {
		sslMode = "disable"
	}
	query.Set("sslmode", sslMode)
	databaseURL.RawQuery = query.Encode()
	return databaseURL.String()
}

func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}
		if key != "" {
			if _, exists := os.LookupEnv(key); !exists {
				_ = os.Setenv(key, value)
			}
		}
	}
}
