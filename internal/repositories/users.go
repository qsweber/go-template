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
	PK        string `dynamodbav:"pk"`
	SK        string `dynamodbav:"sk"`
	ID        string `dynamodbav:"id"`
	Email     string `dynamodbav:"email"`
	CreatedAt int64  `dynamodbav:"created_at"`
}

func UserPartitionKey(id string) string {
	return userPartitionKeyPrefix + id
}

type UsersRepository interface {
	CreateUser(ctx context.Context, id string, email string) error
	// GetUser returns the user with the given id, or nil if no such user exists.
	GetUser(ctx context.Context, id string) (*User, error)
}

type UsersRepositoryImpl struct {
	tableName string
	client    *dynamodb.Client
}

func NewUsersRepository(tableName string, client *dynamodb.Client) UsersRepository {
	return &UsersRepositoryImpl{
		tableName: tableName,
		client:    client,
	}
}

// CreateUser writes the user record. It is idempotent: if the record already exists
// (e.g. Cognito retried the trigger) the existing record is left untouched.
func (r *UsersRepositoryImpl) CreateUser(ctx context.Context, id string, email string) error {
	user := User{
		PK:        UserPartitionKey(id),
		SK:        UserSortKey,
		ID:        id,
		Email:     email,
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

func (r *UsersRepositoryImpl) GetUser(ctx context.Context, id string) (*User, error) {
	out, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			PartitionKeyAttribute: &types.AttributeValueMemberS{Value: UserPartitionKey(id)},
			SortKeyAttribute:      &types.AttributeValueMemberS{Value: UserSortKey},
		},
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
