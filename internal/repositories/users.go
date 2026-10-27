package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const (
	userPartitionKeyPrefix = "USER#"
	UserSortKey            = "USER"
)

type User struct {
	PK          string `dynamodbav:"pk"`
	SK          string `dynamodbav:"sk"`
	ID          string `dynamodbav:"id"`
	DisplayName string `dynamodbav:"display_name,omitempty"`
	CreatedAt   int64  `dynamodbav:"created_at"`
}

func UserPartitionKey(id string) string {
	return userPartitionKeyPrefix + id
}

type UsersRepository interface {
	CreateUser(ctx context.Context, id string) error
	// GetUser returns the user with the given id, or nil if no such user exists.
	GetUser(ctx context.Context, id string) (*User, error)
	// UpdateDisplayName sets the user's display name (an empty name clears it) and returns
	// the updated user, or nil if no such user exists.
	UpdateDisplayName(ctx context.Context, id string, displayName string) (*User, error)
}

type UsersRepositoryImpl struct {
	tableName string
	client    DynamoDBAPI
}

func NewUsersRepository(tableName string, client DynamoDBAPI) UsersRepository {
	return &UsersRepositoryImpl{
		tableName: tableName,
		client:    client,
	}
}

// CreateUser writes the user record. It is idempotent: if the record already exists
// (e.g. Cognito retried the trigger) the existing record is left untouched.
func (r *UsersRepositoryImpl) CreateUser(ctx context.Context, id string) error {
	user := User{
		PK:        UserPartitionKey(id),
		SK:        UserSortKey,
		ID:        id,
		CreatedAt: time.Now().UnixMilli(),
	}

	av, err := attributevalue.MarshalMap(user)
	if err != nil {
		return err
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.tableName),
		Item:                av,
		ConditionExpression: aws.String("attribute_not_exists(" + PartitionKeyAttribute + ")"),
	})
	var conditionFailed *types.ConditionalCheckFailedException
	if errors.As(err, &conditionFailed) {
		return nil
	}
	return err
}

func userKey(id string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		PartitionKeyAttribute: &types.AttributeValueMemberS{Value: UserPartitionKey(id)},
		SortKeyAttribute:      &types.AttributeValueMemberS{Value: UserSortKey},
	}
}

func (r *UsersRepositoryImpl) GetUser(ctx context.Context, id string) (*User, error) {
	out, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key:       userKey(id),
	})
	if err != nil {
		return nil, err
	}
	if out.Item == nil {
		return nil, nil
	}

	var user User
	if err := attributevalue.UnmarshalMap(out.Item, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UsersRepositoryImpl) UpdateDisplayName(ctx context.Context, id string, displayName string) (*User, error) {
	input := &dynamodb.UpdateItemInput{
		TableName:                aws.String(r.tableName),
		Key:                      userKey(id),
		ConditionExpression:      aws.String("attribute_exists(" + PartitionKeyAttribute + ")"),
		ExpressionAttributeNames: map[string]string{"#display_name": "display_name"},
		ReturnValues:             types.ReturnValueAllNew,
	}
	if displayName == "" {
		input.UpdateExpression = aws.String("REMOVE #display_name")
	} else {
		input.UpdateExpression = aws.String("SET #display_name = :display_name")
		input.ExpressionAttributeValues = map[string]types.AttributeValue{
			":display_name": &types.AttributeValueMemberS{Value: displayName},
		}
	}

	out, err := r.client.UpdateItem(ctx, input)
	var conditionFailed *types.ConditionalCheckFailedException
	if errors.As(err, &conditionFailed) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var user User
	if err := attributevalue.UnmarshalMap(out.Attributes, &user); err != nil {
		return nil, err
	}
	return &user, nil
}
