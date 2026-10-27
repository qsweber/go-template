package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/qsweber/go-template/internal/auth"
	"github.com/qsweber/go-template/internal/repositories"
)

// TokenVerifier verifies a bearer token and returns its claims. *auth.CognitoVerifier
// satisfies it; tests substitute a fake.
type TokenVerifier interface {
	VerifyToken(ctx context.Context, token string) (*auth.Claims, error)
}

type ServiceContext struct {
	Repositories repositories.Repositories
	// CognitoVerifier is nil when Cognito isn't configured, which disables authentication.
	CognitoVerifier TokenVerifier
}

func NewServiceContext(ctx context.Context) (ServiceContext, error) {
	var sc ServiceContext

	// Initialize Cognito verifier (optional — auth disabled if config is absent)
	cognitoConfig, err := auth.GetCognitoConfig()
	if err != nil {
		log.Printf("Warning: Failed to load Cognito config: %v", err)
		log.Printf("Authentication will be disabled. Set COGNITO_REGION, COGNITO_USER_POOL_ID, and COGNITO_CLIENT_ID to enable.")
	} else {
		sc.CognitoVerifier = auth.NewCognitoVerifier(cognitoConfig)
		log.Printf("Cognito authentication enabled for region: %s", cognitoConfig.Region)
	}

	// Initialize DynamoDB client
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return ServiceContext{}, fmt.Errorf("failed to load AWS config: %w", err)
	}
	dynamoDBClient := dynamodb.NewFromConfig(cfg)

	// Create repositories backed by the single records table
	recordsTableName := os.Getenv("RECORDS_TABLE")
	if recordsTableName == "" {
		return ServiceContext{}, errors.New("RECORDS_TABLE environment variable is not set")
	}
	sc.Repositories.Users = repositories.NewUsersRepository(recordsTableName, dynamoDBClient)
	log.Printf("Initialized DynamoDB records table: %s", recordsTableName)

	return sc, nil
}
