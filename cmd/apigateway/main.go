package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/qsweber/go-template/internal/auth"
	"github.com/qsweber/go-template/internal/clicks"
	"github.com/qsweber/go-template/internal/rpc"
	"github.com/qsweber/go-template/internal/server"
)

var cognitoVerifier *auth.CognitoVerifier
var clicksRepository clicks.Repository

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

	// Initialize DynamoDB client
	ctx := context.Background()
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}
	dynamoDBClient := dynamodb.NewFromConfig(cfg)

	// Create clicks repository
	clicksTableName := os.Getenv("CLICKS_TABLE")
	if clicksTableName == "" {
		log.Fatalf("CLICKS_TABLE environment variable is not set")
	}
	clicksRepository = clicks.New(clicksTableName, dynamoDBClient)
	log.Printf("Initialized DynamoDB clicks table: %s", clicksTableName)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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

	req := rpc.Request{
		Path:    request.Path,
		Headers: request.Headers,
	}

	srv := server.New(clicksRepository)

	resp := rpc.Handler(ctx, req, srv, cognitoVerifier)

	return events.APIGatewayProxyResponse{
		StatusCode: resp.StatusCode,
		Body:       resp.Body,
		Headers:    resp.Headers,
	}, nil

}

func main() {
	lambda.Start(handler)
}