package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context, event events.DynamoDBEvent) error {
	for _, record := range event.Records {
		if record.EventName != "INSERT" {
			continue
		}

		userIDValue, ok := record.Change.NewImage["cognito_user_id"]
		if !ok || userIDValue.String() == "" {
			return fmt.Errorf("missing cognito_user_id in stream record")
		}

		occurredAtValue, ok := record.Change.NewImage["occurred_at"]
		if !ok || occurredAtValue.Number() == "" {
			return fmt.Errorf("missing occurred_at in stream record")
		}

		log.Printf("click stream event: cognito_user_id=%s occurred_at=%s", userIDValue.String(), occurredAtValue.Number())
	}

	return nil
}

func main() {
	lambda.Start(handler)
}
