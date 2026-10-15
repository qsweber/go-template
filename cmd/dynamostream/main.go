package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/qsweber/go-template/internal/repositories"
)

func handler(ctx context.Context, event events.DynamoDBEvent) error {
	for _, record := range event.Records {
		if record.EventName != "INSERT" {
			continue
		}

		image := record.Change.NewImage
		if stringAttribute(image, repositories.SortKeyAttribute) != repositories.UserSortKey {
			continue
		}

		log.Printf(
			"new user created: id=%s email=%s",
			stringAttribute(image, "id"),
			stringAttribute(image, "email"),
		)
	}

	return nil
}

// stringAttribute returns the named string attribute, or "" if it is absent or not a string
// (DynamoDBAttributeValue.String panics on a type mismatch).
func stringAttribute(image map[string]events.DynamoDBAttributeValue, name string) string {
	value, ok := image[name]
	if !ok || value.DataType() != events.DataTypeString {
		return ""
	}
	return value.String()
}

func main() {
	lambda.Start(handler)
}
