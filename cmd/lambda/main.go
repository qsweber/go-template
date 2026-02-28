package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/qsweber/go-template/internal/auth"
	"github.com/qsweber/go-template/internal/server"
)

var cognitoVerifier *auth.CognitoVerifier

func init() {
	// Initialize Cognito verifier at startup
	config, err := auth.GetCognitoConfig()
	if err != nil {
		log.Printf("Warning: Failed to load Cognito config: %v", err)
		log.Printf("Authentication will be disabled. Set COGNITO_REGION, COGNITO_USER_POOL_ID, and COGNITO_CLIENT_ID to enable.")
		cognitoVerifier = nil
	} else {
		cognitoVerifier = auth.NewCognitoVerifier(config)
		log.Printf("Cognito authentication enabled for region: %s", config.Region)
	}
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	srv := server.New()

	// Handle CORS preflight requests
	if request.HTTPMethod == "OPTIONS" {
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"Access-Control-Allow-Origin":  "*",
				"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
				"Access-Control-Allow-Headers": "Content-Type, Authorization",
			},
		}, nil
	}

	switch request.Path {
	case "/ping":
		result := srv.Ping()
		responseBytes, err := json.Marshal(map[string]bool{"ok": result})
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500, Body: "Error encoding response"}, err
		}
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Body:       string(responseBytes),
			Headers: map[string]string{
				"Content-Type":                "application/json",
				"Access-Control-Allow-Origin": "*",
			},
		}, nil
	case "/foo":
		// Verify authentication for /foo endpoint
		if cognitoVerifier != nil {
			authHeader := request.Headers["Authorization"]
			if authHeader == "" {
				authHeader = request.Headers["authorization"] // try lowercase
			}

			token, err := auth.ExtractBearerToken(authHeader)
			if err != nil {
				return events.APIGatewayProxyResponse{
					StatusCode: 401,
					Body:       `{"error": "Unauthorized: ` + err.Error() + `"}`,
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
				}, nil
			}

			claims, err := cognitoVerifier.VerifyToken(context.Background(), token)
			if err != nil {
				return events.APIGatewayProxyResponse{
					StatusCode: 401,
					Body:       `{"error": "Unauthorized: Invalid or expired token"}`,
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
				}, nil
			}

			log.Printf("Authenticated user: %s (email: %s)", claims.Subject, claims.Email)
		}

		bar := request.QueryStringParameters["bar"] // /foo?bar=hello
		input := server.FooInput{Bar: bar}
		output, err := srv.Foo(input)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500, Body: "Error calling Foo"}, err
		}
		responseBytes, err := json.Marshal(output)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500, Body: "Error encoding response"}, err
		}
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Body:       string(responseBytes),
			Headers: map[string]string{
				"Content-Type":                "application/json",
				"Access-Control-Allow-Origin": "*",
			},
		}, nil
	default:
		return events.APIGatewayProxyResponse{StatusCode: 404, Body: "Not Found"}, nil
	}
}

func main() {
	lambda.Start(handler)
}
