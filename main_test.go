package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/cw-sakamoto/sample/localstack"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/require"
)

type testcase struct {
	setupS3  bool
	setupDDB bool
	setupSQS bool

	okLabels []labels
	ngLabels []labels
}

func TestRun(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		checkRun(t, testcase{
			setupS3:  true,
			setupDDB: true,
			setupSQS: true,
			okLabels: []labels{
				{"S3", "GetObject", "Success"},
				{"SQS", "ReceiveMessage", "Success"},
				{"DynamoDB", "DeleteItem", "Success"},
				{"DynamoDB", "GetItem", "Success"},
				{"DynamoDB", "GetItemConsistent", "Success"},
				{"DynamoDB", "PutGetItemConsistent", "Success"},
				{"DynamoDB", "PutItem", "Success"},
				{"DynamoDB", "Query", "Success"},
				{"DynamoDB", "QueryConsistent", "Success"},
				{"DynamoDB", "Scan", "Success"},
				{"DynamoDB", "UpdateItem", "Success"},
			},
			ngLabels: []labels{
				{"S3", "GetObject", "Failure"},
				{"SQS", "ReceiveMessage", "Failure"},
				{"DynamoDB", "DeleteItem", "Failure"},
				{"DynamoDB", "GetItem", "Failure"},
				{"DynamoDB", "GetItemConsistent", "Failure"},
				{"DynamoDB", "PutGetItemConsistent", "Failure"},
				{"DynamoDB", "PutItem", "Failure"},
				{"DynamoDB", "Query", "Failure"},
				{"DynamoDB", "QueryConsistent", "Failure"},
				{"DynamoDB", "Scan", "Failure"},
				{"DynamoDB", "UpdateItem", "Failure"},
			},
		})
	})

	t.Run("s3 failing", func(t *testing.T) {
		checkRun(t, testcase{
			setupS3:  false,
			setupDDB: true,
			setupSQS: true,
			okLabels: []labels{
				{"S3", "GetObject", "Failure"},
				{"SQS", "ReceiveMessage", "Success"},
				{"DynamoDB", "DeleteItem", "Success"},
				{"DynamoDB", "GetItem", "Success"},
				{"DynamoDB", "GetItemConsistent", "Success"},
				{"DynamoDB", "PutGetItemConsistent", "Success"},
				{"DynamoDB", "PutItem", "Success"},
				{"DynamoDB", "Query", "Success"},
				{"DynamoDB", "QueryConsistent", "Success"},
				{"DynamoDB", "Scan", "Success"},
				{"DynamoDB", "UpdateItem", "Success"},
			},
			ngLabels: []labels{
				{"S3", "GetObject", "Success"},
				{"SQS", "ReceiveMessage", "Failure"},
				{"DynamoDB", "DeleteItem", "Failure"},
				{"DynamoDB", "GetItem", "Failure"},
				{"DynamoDB", "GetItemConsistent", "Failure"},
				{"DynamoDB", "PutGetItemConsistent", "Failure"},
				{"DynamoDB", "PutItem", "Failure"},
				{"DynamoDB", "Query", "Failure"},
				{"DynamoDB", "QueryConsistent", "Failure"},
				{"DynamoDB", "Scan", "Failure"},
				{"DynamoDB", "UpdateItem", "Failure"},
			},
		})
	})

	t.Run("sqs failing", func(t *testing.T) {
		checkRun(t, testcase{
			setupS3:  true,
			setupDDB: true,
			setupSQS: false,
			okLabels: []labels{
				{"S3", "GetObject", "Success"},
				{"SQS", "ReceiveMessage", "Failure"},
				{"DynamoDB", "DeleteItem", "Success"},
				{"DynamoDB", "GetItem", "Success"},
				{"DynamoDB", "GetItemConsistent", "Success"},
				{"DynamoDB", "PutGetItemConsistent", "Success"},
				{"DynamoDB", "PutItem", "Success"},
				{"DynamoDB", "Query", "Success"},
				{"DynamoDB", "QueryConsistent", "Success"},
				{"DynamoDB", "Scan", "Success"},
				{"DynamoDB", "UpdateItem", "Success"},
			},
			ngLabels: []labels{
				{"S3", "GetObject", "Failure"},
				{"SQS", "ReceiveMessage", "Success"},
				{"DynamoDB", "DeleteItem", "Failure"},
				{"DynamoDB", "GetItem", "Failure"},
				{"DynamoDB", "GetItemConsistent", "Failure"},
				{"DynamoDB", "PutGetItemConsistent", "Failure"},
				{"DynamoDB", "PutItem", "Failure"},
				{"DynamoDB", "Query", "Failure"},
				{"DynamoDB", "QueryConsistent", "Failure"},
				{"DynamoDB", "Scan", "Failure"},
				{"DynamoDB", "UpdateItem", "Failure"},
			},
		})
	})

	t.Run("DynamoDB failing", func(t *testing.T) {
		checkRun(t, testcase{
			setupS3:  true,
			setupDDB: false,
			setupSQS: true,
			okLabels: []labels{
				{"S3", "GetObject", "Success"},
				{"SQS", "ReceiveMessage", "Success"},
				{"DynamoDB", "DeleteItem", "Failure"},
				{"DynamoDB", "GetItem", "Failure"},
				{"DynamoDB", "GetItemConsistent", "Failure"},
				{"DynamoDB", "PutGetItemConsistent", "Failure"},
				{"DynamoDB", "PutItem", "Failure"},
				{"DynamoDB", "Query", "Failure"},
				{"DynamoDB", "QueryConsistent", "Failure"},
				{"DynamoDB", "Scan", "Failure"},
				{"DynamoDB", "UpdateItem", "Failure"},
			},
			ngLabels: []labels{
				{"S3", "GetObject", "Failure"},
				{"SQS", "ReceiveMessage", "Failure"},
				{"DynamoDB", "DeleteItem", "Success"},
				{"DynamoDB", "GetItem", "Success"},
				{"DynamoDB", "GetItemConsistent", "Success"},
				{"DynamoDB", "PutGetItemConsistent", "Success"},
				{"DynamoDB", "PutItem", "Success"},
				{"DynamoDB", "Query", "Success"},
				{"DynamoDB", "QueryConsistent", "Success"},
				{"DynamoDB", "Scan", "Success"},
				{"DynamoDB", "UpdateItem", "Success"},
			},
		})
	})
}

func checkRun(t *testing.T, tc testcase) {
	t.Helper()

	var (
		runErr = make(chan error, 1)

		sigs = make(chan os.Signal, 1)
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	awsConfig, err := config.LoadDefaultConfig(ctx)
	require.NoError(t, err)

	s3EndpointResolver := localstack.S3EndpointResolver()
	sqsEndpointResolver := localstack.SQSEndpointResolver()
	dynamodbEndpointResolver := localstack.DynamoDBEndpointResolver()

	if tc.setupS3 {
		setupS3BucketAndObject(t, ctx, awsConfig, s3EndpointResolver)
	}

	if tc.setupDDB {
		setupDynamoDBTable(t, ctx, awsConfig, dynamodbEndpointResolver)
	}

	if tc.setupSQS {
		preservedQueueURL := os.Getenv("SQS_QUEUE_URL")
		queueURL := setupSQSQueue(t, ctx, awsConfig, sqsEndpointResolver)
		os.Setenv("SQS_QUEUE_URL", queueURL)
		defer func() {
			os.Setenv("SQS_QUEUE_URL", preservedQueueURL)
		}()
	}

	go func() {
		runErr <- Run(ContextWithSignal(ctx, sigs), func(c *checker) {
			// Use localstack for S3 and DynamoDB
			c.s3Opts = append(c.s3Opts, s3.WithEndpointResolverV2(s3EndpointResolver))
			c.sqsOpts = append(c.sqsOpts, sqs.WithEndpointResolverV2(sqsEndpointResolver))
			c.dynamodbOpts = append(c.dynamodbOpts, dynamodb.WithEndpointResolverV2(dynamodbEndpointResolver))
			c.awsAPICallInterval = 1 * time.Millisecond
		})

		cancel()
	}()

	// Note that we don't propagate `ctx` to `waitCtx`
	// because we want to wait for the server to start
	// exposing metrics even if the context is canceled.
	//
	// In other words, propagating `ctx` to `waitCtx` results in
	// httpGetStr always reporting `connection refused` in case of timeout,
	// which is not what we want.
	waitCtx, waitCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer waitCancel()

	//
	// Wait for the server to start exposing metrics
	//

	type errorInfo struct {
		err     error
		metrics map[labels]struct{}
	}

	var (
		errorInfoCh = make(chan errorInfo, 1)
	)

	metrics := make(chan map[labels]struct{}, 1)
	go func() {
		var lastErr error
		var lastMetrics map[labels]struct{}
		for waitCtx.Err() == nil {
			time.Sleep(100 * time.Millisecond)

			m, err := httpGetStr(t, "http://localhost:8080/metrics")
			if err != nil {
				lastErr = err
				lastMetrics = nil
				continue
			}

			mm, err := parseMetrics(t, m)
			if err != nil {
				lastErr = err
				lastMetrics = nil
				continue
			}

			lastErr = nil
			lastMetrics = mm

			if len(mm) == len(tc.okLabels) {
				metrics <- mm
				break
			}
		}

		var errInfo errorInfo
		if lastErr != nil {
			errInfo.err = lastErr
		} else if lastMetrics != nil {
			errInfo.metrics = lastMetrics
		}
		errorInfoCh <- errInfo
	}()

	select {
	case mm := <-metrics:
		for _, l := range tc.okLabels {
			require.Contains(t, mm, l)
		}

		for _, l := range tc.ngLabels {
			require.NotContains(t, mm, l)
		}
	case <-waitCtx.Done():
		errInfo := <-errorInfoCh

		if errInfo.err != nil {
			t.Logf("Last error seen before timeout: %v", errInfo.err)
		}

		if errInfo.metrics != nil {
			t.Logf("Last metrics seen before timeout: %v", errInfo.metrics)
		}

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

type labels struct {
	service, method, status string
}

func parseMetrics(t *testing.T, m string) (map[labels]struct{}, error) {
	p := expfmt.NewTextParser(model.UTF8Validation)
	mf, err := p.TextToMetricFamilies(strings.NewReader(m))
	require.NoError(t, err)

	mt, ok := mf["aws_request_duration_seconds"]
	if !ok {
		return nil, fmt.Errorf("metric family aws_request_duration_seconds not found")
	}

	mm := make(map[labels]struct{})

	for _, m := range mt.Metric {
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

	return mm, nil
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

func setupSQSQueue(t *testing.T, ctx context.Context, awsConfig aws.Config, sqsEndpointResolver sqs.EndpointResolverV2) string {
	sqsClient := sqs.NewFromConfig(awsConfig, sqs.WithEndpointResolverV2(sqsEndpointResolver))

	// We assume that SQS_QUEUE_URL is set in the environment variables
	// before running the tests.
	sqsQueueURL := os.Getenv("SQS_QUEUE_URL")
	queueName := sqsQueueURL[strings.LastIndex(sqsQueueURL, "/")+1:]

	res, err := sqsClient.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: aws.String(queueName),
	})
	require.NoError(t, err)
	sqsQueueURL = *res.QueueUrl
	t.Cleanup(func() {
		_, err = sqsClient.DeleteQueue(context.Background(), &sqs.DeleteQueueInput{
			QueueUrl: &sqsQueueURL,
		})
		if err != nil {
			t.Logf("Error: %v", err)
		}
	})

	return sqsQueueURL
}

func setupDynamoDBTable(t *testing.T, ctx context.Context, awsConfig aws.Config, dynamodbEndpointResolver dynamodb.EndpointResolverV2) {
	dynamodbClient := dynamodb.NewFromConfig(awsConfig, dynamodb.WithEndpointResolverV2(dynamodbEndpointResolver))

	// We assume that DYNAMODB_TABLE is set in the environment variables
	// before running the tests.
	dynamodbTable := os.Getenv("DYNAMODB_TABLE")

	_, err := dynamodbClient.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: &dynamodbTable,
		KeySchema: []dynamodbtypes.KeySchemaElement{
			{
				AttributeName: aws.String("id"),
				KeyType:       dynamodbtypes.KeyTypeHash,
			},
		},
		AttributeDefinitions: []dynamodbtypes.AttributeDefinition{
			{
				AttributeName: aws.String("id"),
				AttributeType: dynamodbtypes.ScalarAttributeTypeS,
			},
		},
		ProvisionedThroughput: &dynamodbtypes.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(1),
			WriteCapacityUnits: aws.Int64(1),
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err = dynamodbClient.DeleteTable(context.Background(), &dynamodb.DeleteTableInput{
			TableName: &dynamodbTable,
		})
		if err != nil {
			t.Logf("Error: %v", err)
		}
	})
}

func httpGetStr(t *testing.T, url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
