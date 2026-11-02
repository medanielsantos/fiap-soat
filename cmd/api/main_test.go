package main

import (
	"os"
	"syscall"
	"testing"
)

func TestMainShutsDownGracefullyOnSIGTERM(t *testing.T) {
	baseURL := setAPIEnvironment(t)
	done := make(chan error, 1)
	go func() {
		main()
		done <- nil
	}()

	// The server only answers after main registered its signal handler, so SIGTERM is safe to send.
	waitHealthy(t, baseURL)
	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	if err := waitDone(t, done); err != nil {
		t.Fatal(err)
	}
}
