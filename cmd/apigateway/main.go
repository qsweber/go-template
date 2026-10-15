package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/qsweber/go-template/internal/server"
)

var serviceCtx server.ServiceContext

func init() {
	var err error
	serviceCtx, err = server.NewServiceContext(context.Background())
	if err != nil {
		log.Fatalf("Failed to initialize service context: %v", err)
	}
}

// handler receives HTTP API (payload format 2.0) events. Header names arrive lowercased.
func handler(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	srv := server.New(serviceCtx)
	req := server.Request{
		Method:      request.RequestContext.HTTP.Method,
		Path:        request.RawPath,
		Headers:     request.Headers,
		QueryParams: request.QueryStringParameters,
	}

	resp := srv.Handle(ctx, req)

	return events.APIGatewayV2HTTPResponse{
		StatusCode: resp.StatusCode,
		Body:       resp.Body,
		Headers:    resp.Headers,
	}, nil

}

func main() {
	lambda.Start(handler)
}
