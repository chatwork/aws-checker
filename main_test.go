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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/cw-sakamoto/sample/localstack"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/require"
)

func TestSigint(t *testing.T) {
	var (
		runErr = make(chan error, 1)

		sigs = make(chan os.Signal, 1)
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	awsConfig, err := config.LoadDefaultConfig(ctx)
	require.NoError(t, err)

	s3EndpointResolver := localstack.S3EndpointResolver()

	setupS3BucketAndObject(t, ctx, awsConfig, s3EndpointResolver)

	go func() {
		runErr <- Run(ContextWithSignal(ctx, sigs), func(c *checker) {
			// Use localstack for S3
			c.s3Opts = append(c.s3Opts, s3.WithEndpointResolverV2(s3EndpointResolver))
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

		p := expfmt.TextParser{}
		mf, err := p.TextToMetricFamilies(strings.NewReader(m))
		require.NoError(t, err)

		mt, ok := mf["aws_request_duration_seconds"]
		require.True(t, ok)

		type labels struct {
			service, method, status string
		}

		mm := make(map[labels]struct{})

		for _, m := range mt.Metric {
			t.Logf("Metric: %v", m)
			var labels labels
			for _, l := range m.Label {
				switch *l.Name {
				case "service":
					labels.service = *l.Value
				case "method":
					labels.method = *l.Value
				case "status":
					labels.status = *l.Value
				}
			}
			mm[labels] = struct{}{}
		}

		require.Len(t, mm, 1)
		require.Contains(t, mm, labels{"S3", "GetObject", "Success"})
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

// Put s3 object for testing
func setupS3BucketAndObject(t *testing.T, ctx context.Context, awsConfig aws.Config, s3EndpointResolver s3.EndpointResolverV2) {
	s3Client := s3.NewFromConfig(awsConfig, s3.WithEndpointResolverV2(s3EndpointResolver))

	// We assume that S3_BUCKET and S3_KEY are set in the environment variables
	// before running the tests.
	s3Bucket := os.Getenv("S3_BUCKET")
	s3Key := os.Getenv("S3_KEY")

	_, err := s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: &s3Bucket,
		// LocationConstraint is required when AWS_DEFAULT_REGION is not us-east-1
		// Otherwise, you will get the following error:
		//   IllegalLocationConstraintException: The unspecified location constraint is incompatible for the region specific endpoint this request was sent to.
		CreateBucketConfiguration: &s3types.CreateBucketConfiguration{
			LocationConstraint: s3types.BucketLocationConstraint(os.Getenv("AWS_DEFAULT_REGION")),
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err = s3Client.DeleteBucket(context.Background(), &s3.DeleteBucketInput{
			Bucket: &s3Bucket,
		})
		if err != nil {
			t.Logf("Error: %v", err)
		}
	})

	_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &s3Bucket,
		Key:    &s3Key,
		Body:   strings.NewReader("hello"),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := s3Client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
			Bucket: &s3Bucket,
			Key:    &s3Key,
		})
		if err != nil {
			t.Logf("Error: %v", err)
		}
	})
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
