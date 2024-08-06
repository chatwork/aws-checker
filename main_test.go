package main

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSigint(t *testing.T) {
	var (
		runErr = make(chan error, 1)

		sigs = make(chan os.Signal, 1)
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		runErr <- Run(sigs)

		cancel()
	}()

	sigs <- syscall.SIGINT

	select {
	case <-ctx.Done():
		require.NoError(t, <-runErr)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout")
	}
}
