package repositories

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type Click struct {
	CognitoUserID string `dynamodbav:"cognito_user_id"`
	OccurredAt    int64  `dynamodbav:"occurred_at"`
}

type ClicksRepository interface {
	GetTableName() string
	RecordClick(ctx context.Context, cognitoUserID string) error
	GetClickCount(ctx context.Context) (int, error)
}

type ClicksRepositoryImpl struct {
	tableName string
	client    *dynamodb.Client
}

func NewClicksRepository(tableName string, client *dynamodb.Client) ClicksRepository {
	return &ClicksRepositoryImpl{
		tableName: tableName,
		client:    client,
	}
}

func (r *ClicksRepositoryImpl) GetTableName() string {
	return r.tableName
}

func (r *ClicksRepositoryImpl) GetClickCount(ctx context.Context) (int, error) {
	out, err := r.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(r.tableName),
	})
	if err != nil {
		return 0, err
	}
	return int(aws.ToInt64(out.Table.ItemCount)), nil
}

func (r *ClicksRepositoryImpl) RecordClick(ctx context.Context, cognitoUserID string) error {
	click := Click{
		CognitoUserID: cognitoUserID,
		OccurredAt:    time.Now().UnixMilli(),
	}

	av, err := attributevalue.MarshalMap(click)
	if err != nil {
		return err
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      av,
	}

	_, err = r.client.PutItem(ctx, input)
	return err
}
