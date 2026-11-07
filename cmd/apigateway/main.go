package main

import (
	"context"
	"encoding/base64"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/qsweber/go-template/internal/server"
)

type apiGatewayHandler func(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error)

// newHandler adapts HTTP API (payload format 2.0) events to the server's
// Request/Response types, and logs one JSON line per request. Header names arrive lowercased.
func newHandler(srv server.Server, logger *slog.Logger) apiGatewayHandler {
	return func(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		start := time.Now()
		resp := handle(ctx, srv, request)

		// Headers, the body and the query string are left out on purpose: they can carry the
		// bearer token, user input and the caller's user id.
		logger.InfoContext(ctx, "request",
			"requestId", request.RequestContext.RequestID,
			"method", request.RequestContext.HTTP.Method,
			"path", request.RawPath,
			"status", resp.StatusCode,
			"durationMs", time.Since(start).Milliseconds(),
			"sourceIp", request.RequestContext.HTTP.SourceIP,
			"userAgent", request.RequestContext.HTTP.UserAgent,
		)

		return resp, nil
	}
}

func handle(ctx context.Context, srv server.Server, request events.APIGatewayV2HTTPRequest) events.APIGatewayV2HTTPResponse {
	body := request.Body
	if request.IsBase64Encoded {
		decoded, err := base64.StdEncoding.DecodeString(body)
		if err != nil {
			return events.APIGatewayV2HTTPResponse{StatusCode: 400, Body: `{"error":"invalid base64 body"}`}
		}
		body = string(decoded)
	}

	req := server.Request{
		Method:      request.RequestContext.HTTP.Method,
		Path:        request.RawPath,
		Headers:     request.Headers,
		QueryParams: request.QueryStringParameters,
		Body:        body,
	}

	resp := srv.Handle(ctx, req)

	return events.APIGatewayV2HTTPResponse{
		StatusCode: resp.StatusCode,
		Body:       resp.Body,
		Headers:    resp.Headers,
	}
}

func main() {
	serviceCtx, err := server.NewServiceContext(context.Background())
	if err != nil {
		log.Fatalf("Failed to initialize service context: %v", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	lambda.Start(newHandler(server.New(serviceCtx), logger))
}
