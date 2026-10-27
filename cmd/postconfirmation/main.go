package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/qsweber/go-template/internal/repositories"
)

type postConfirmationHandler func(ctx context.Context, event events.CognitoEventUserPoolsPostConfirmation) (events.CognitoEventUserPoolsPostConfirmation, error)

// newHandler returns the Cognito PostConfirmation trigger. It fires once a user has
// confirmed their sign-up, and creates the corresponding USER record in the records table.
func newHandler(usersRepository repositories.UsersRepository) postConfirmationHandler {
	return func(ctx context.Context, event events.CognitoEventUserPoolsPostConfirmation) (events.CognitoEventUserPoolsPostConfirmation, error) {
		// PostConfirmation also fires after a forgot-password confirmation; only
		// sign-up confirmations represent a newly created user.
		if event.TriggerSource != "PostConfirmation_ConfirmSignUp" {
			return event, nil
		}

		userID := event.Request.UserAttributes["sub"]
		if userID == "" {
			return event, fmt.Errorf("missing sub in post confirmation event")
		}
		if err := usersRepository.CreateUser(ctx, userID); err != nil {
			return event, fmt.Errorf("failed to create user record: %w", err)
		}

		return event, nil
	}
}

func main() {
	tableName := os.Getenv("RECORDS_TABLE")
	if tableName == "" {
		log.Fatalf("RECORDS_TABLE environment variable is not set")
	}

	cfg, err := awsconfig.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	lambda.Start(newHandler(repositories.NewUsersRepository(tableName, dynamodb.NewFromConfig(cfg))))
}
