package repositories

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// The app uses a single-table DynamoDB design: every entity lives in the one
// "<stack>-records" table, distinguished by its partition key (pk) and sort key (sk).
const (
	PartitionKeyAttribute = "pk"
	SortKeyAttribute      = "sk"
)

// DynamoDBAPI is the subset of the DynamoDB client the repositories use, so tests can
// substitute a fake. *dynamodb.Client satisfies it.
type DynamoDBAPI interface {
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
}

type Repositories struct {
	Users UsersRepository
}
