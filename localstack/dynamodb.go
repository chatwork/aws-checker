package localstack

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	smithyendpoints "github.com/aws/smithy-go/endpoints"
)

func DynamoDBEndpointResolver() *dynamoDBEndpointResolver {
	return &dynamoDBEndpointResolver{
		baseResolver: dynamodb.NewDefaultEndpointResolverV2(),
	}
}

type dynamoDBEndpointResolver struct {
	baseResolver dynamodb.EndpointResolverV2
}

func (r *dynamoDBEndpointResolver) ResolveEndpoint(ctx context.Context, params dynamodb.EndpointParameters) (smithyendpoints.Endpoint, error) {
	params.Endpoint = aws.String("http://localhost:4566")

	return r.baseResolver.ResolveEndpoint(ctx, params)
}
