package main

import (
	"context"
	"encoding/base64"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/qsweber/go-template/internal/server"
)

type apiGatewayHandler func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

// newHandler adapts API Gateway proxy events to the server's Request/Response types.
func newHandler(srv server.Server) apiGatewayHandler {
	return func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		body := request.Body
		if request.IsBase64Encoded {
			decoded, err := base64.StdEncoding.DecodeString(body)
			if err != nil {
				return events.APIGatewayProxyResponse{StatusCode: 400, Body: `{"error":"invalid base64 body"}`}, nil
			}
			body = string(decoded)
		}

		req := server.Request{
			Method:      request.HTTPMethod,
			Path:        request.Path,
			Headers:     request.Headers,
			QueryParams: request.QueryStringParameters,
			Body:        body,
		}

		resp := srv.Handle(ctx, req)

		return events.APIGatewayProxyResponse{
			StatusCode: resp.StatusCode,
			Body:       resp.Body,
			Headers:    resp.Headers,
		}, nil
	}
}

func main() {
	serviceCtx, err := server.NewServiceContext(context.Background())
	if err != nil {
		log.Fatalf("Failed to initialize service context: %v", err)
	}

	lambda.Start(newHandler(server.New(serviceCtx)))
}
