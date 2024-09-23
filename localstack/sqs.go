package localstack

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	smithyendpoints "github.com/aws/smithy-go/endpoints"
)

func SQSEndpointResolver() *sqsEndpointResolver {
	return &sqsEndpointResolver{
		baseResolver: sqs.NewDefaultEndpointResolverV2(),
	}
}

type sqsEndpointResolver struct {
	baseResolver sqs.EndpointResolverV2
}

func (r *sqsEndpointResolver) ResolveEndpoint(ctx context.Context, params sqs.EndpointParameters) (smithyendpoints.Endpoint, error) {
	params.Endpoint = aws.String("http://localhost:4566")
	return r.baseResolver.ResolveEndpoint(ctx, params)
}
