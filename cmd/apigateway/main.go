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

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	srv := server.New(serviceCtx)
	req := server.Request{
		Method:  request.HTTPMethod,
		Path:    request.Path,
		Headers: request.Headers,
	}

	resp := srv.Handle(ctx, req)

	return events.APIGatewayProxyResponse{
		StatusCode: resp.StatusCode,
		Body:       resp.Body,
		Headers:    resp.Headers,
	}, nil

}

func main() {
	lambda.Start(handler)
}
