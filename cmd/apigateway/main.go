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
	"github.com/qsweber/go-template/internal/repositories"
	"github.com/qsweber/go-template/internal/server"
)

var serviceCtx server.ServiceContext

func init() {
	// Initialize Cognito verifier at startup
	config, err := auth.GetCognitoConfig()
	if err != nil {
		log.Printf("Warning: Failed to load Cognito config: %v", err)
		log.Printf("Authentication will be disabled. Set COGNITO_REGION, COGNITO_USER_POOL_ID, and COGNITO_CLIENT_ID to enable.")
		serviceCtx.CognitoVerifier = nil
	} else {
		serviceCtx.CognitoVerifier = auth.NewCognitoVerifier(config)
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
	serviceCtx.Repositories.Clicks = repositories.NewClicksRepository(clicksTableName, dynamoDBClient)
	log.Printf("Initialized DynamoDB clicks table: %s", clicksTableName)
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
