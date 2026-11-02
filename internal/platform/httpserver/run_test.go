package httpserver

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

func freeAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	_ = listener.Close()
	return address
}

func TestRunServesUntilContextIsCanceled(t *testing.T) {
	address := freeAddress(t)
	server := newTestServer(t, address)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- server.Run(ctx) }()

	waitHealthy(t, "http://"+address+"/health")
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run after cancel = %v, want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not stop after context cancellation")
	}
}

func TestRunReturnsListenError(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	server := newTestServer(t, listener.Addr().String())
	if err := server.Run(context.Background()); err == nil {
		t.Fatal("Run on an occupied address must fail")
	}
}

func waitHealthy(t *testing.T, url string) {
	t.Helper()
	client := http.Client{Timeout: time.Second}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if response, err := client.Get(url); err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("server at %s never became healthy", url)
}
