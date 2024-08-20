package localstack

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	smithyendpoints "github.com/aws/smithy-go/endpoints"
)

func S3EndpointResolver() s3.EndpointResolverV2 {
	return &s3EndpointResolver{
		baseResolver: s3.NewDefaultEndpointResolverV2(),
	}
}

type s3EndpointResolver struct {
	baseResolver s3.EndpointResolverV2
}

func (r *s3EndpointResolver) ResolveEndpoint(ctx context.Context, params s3.EndpointParameters) (smithyendpoints.Endpoint, error) {
	params.ForcePathStyle = aws.Bool(true)
	params.Endpoint = aws.String("http://localhost:4566")

	return r.baseResolver.ResolveEndpoint(ctx, params)
}
