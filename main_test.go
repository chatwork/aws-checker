package main

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestSigint(t *testing.T) {
	sigs := make(chan os.Signal, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		Run(sigs)

		cancel()
	}()

	sigs <- syscall.SIGINT

	select {
	case <-ctx.Done():
	case <-time.After(1 * time.Second):
		t.Fatal("timeout")
	}
}
