package config

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

const defaultHTTPPort = "8080"

const (
	EnvLocal      = "local"
	EnvStaging    = "staging"
	EnvProduction = "production"
)

type Config struct {
	HTTPAddress              string
	DatabaseURL              string
	RedisURL                 string
	JWTSecret                string
	JWTIssuer                string
	JWTExpiry                time.Duration
	SMTPHost                 string
	SMTPPort                 string
	SMTPUser                 string
	SMTPPassword             string
	SMTPFrom                 string
	PasswordResetURL         string
	CustomerPasswordResetURL string
	AppEnv                   string
	SeedPassword             string
}

func Load() Config {
	LoadDotEnvFile(".env")
	port := os.Getenv("APP_HTTP_PORT")
	if port == "" {
		port = defaultHTTPPort
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = postgresURLFromEnvironment()
	}
	return Config{
		HTTPAddress:              ":" + port,
		DatabaseURL:              databaseURL,
		RedisURL:                 os.Getenv("REDIS_URL"),
		JWTSecret:                os.Getenv("JWT_SECRET"),
		JWTIssuer:                envOr("JWT_ISSUER", "fiap-soat"),
		JWTExpiry:                24 * time.Hour,
		SMTPHost:                 os.Getenv("SMTP_HOST"),
		SMTPPort:                 envOr("SMTP_PORT", "587"),
		SMTPUser:                 os.Getenv("SMTP_USERNAME"),
		SMTPPassword:             os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:                 os.Getenv("SMTP_FROM"),
		PasswordResetURL:         os.Getenv("PASSWORD_RESET_URL"),
		CustomerPasswordResetURL: os.Getenv("CUSTOMER_PASSWORD_RESET_URL"),
		AppEnv:                   envOr("APP_ENV", EnvProduction),
		SeedPassword:             os.Getenv("DEFAULT_PASSWORD_SEED"),
	}
}

func LoadValidated() (Config, error) {
	cfg := Load()
	return cfg, cfg.Validate()
}

func (c Config) Validate() error {
	switch c.AppEnv {
	case EnvLocal, EnvStaging, EnvProduction:
		return nil
	}
	return fmt.Errorf("invalid APP_ENV %q: use %s, %s or %s", c.AppEnv, EnvLocal, EnvStaging, EnvProduction)
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// LoadDotEnvFile loads missing environment variables from the supplied file.
// Variables already present in the process environment take precedence.
func LoadDotEnvFile(path string) {
	loadDotEnv(path)
}

// PostgresURLFromEnvironment builds a PostgreSQL URL from DB_* variables,
// without consulting DATABASE_URL. This is useful for isolated test configs.
func PostgresURLFromEnvironment() string {
	return postgresURLFromEnvironment()
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
	defer func() {
		_ = file.Close()
	}()

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
