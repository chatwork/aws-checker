package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/cw-sakamoto/sample/localstack"
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
		runErr <- Run(ContextWithSignal(ctx, sigs), func(c *checker) {
			// Use localstack for S3
			c.s3Opts = append(c.s3Opts, s3.WithEndpointResolverV2(localstack.S3EndpointResolver()))
		})

		cancel()
	}()

	//
	// Wait for the server to start exposing metrics
	//

	metrics := make(chan string, 1)
	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			m := httpGetStr(t, "http://localhost:8080/metrics")
			if strings.Contains(m, "aws_request_duration_seconds") {
				metrics <- m
				break
			}
		}
	}()

	select {
	case m := <-metrics:
		// m is the metrics in the Prometheus exposition format,
		// expectedly containing the aws_request_duration_seconds metric.
		// We can check the presence of any metric here, in any detail.
		require.Contains(t, m, "promhttp_metric_handler_requests_total")
	case <-time.After(2 * time.Second):
		// We assume that the server is expected to start and expose metrics within 2 seconds.
		// Otherwise, we consider it as a failure, and you may need to fix the server implementation,
		// or you may need to increase the timeout if the runtime environment is soooo slow.
		t.Fatal("timed out waiting for metrics")
	}

	sigs <- syscall.SIGINT

	select {
	case <-ctx.Done():
		require.NoError(t, <-runErr)
	case <-time.After(1 * time.Second):
		// We assume the server can gracefully shut down within 5 seconds.
		// Otherwise, we consider it as a failure, and you may need to fix the server implementation.
		t.Fatal("timeout")
	}
}

func httpGetStr(t *testing.T, url string) string {
	resp, err := http.Get(url)
	if err != nil {
		t.Logf("Error: %v", err)
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Logf("Error: %v", err)
		return ""
	}

	return string(body)
}
