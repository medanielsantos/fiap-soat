package main

import (
	"net"
	"net/http"
	"os"
	"testing"
	"time"
)

func setAPIEnvironment(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run the API command tests")
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	_, port, _ := net.SplitHostPort(listener.Addr().String())
	_ = listener.Close()
	for key, value := range map[string]string{
		"APP_HTTP_PORT": port, "DATABASE_URL": dsn, "DB_NAME": "", "DB_USER": "", "REDIS_URL": "",
		"JWT_SECRET": "0123456789abcdef0123456789abcdef", "JWT_ISSUER": "",
	} {
		t.Setenv(key, value)
	}
	return "http://127.0.0.1:" + port
}

func waitHealthy(t *testing.T, baseURL string) {
	t.Helper()
	client := http.Client{Timeout: time.Second}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if response, err := client.Get(baseURL + "/health"); err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("API at %s never became healthy", baseURL)
}

func waitDone(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(15 * time.Second):
		t.Fatal("API did not shut down")
		return nil
	}
}
