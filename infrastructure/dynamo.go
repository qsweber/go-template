package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/dynamodb"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const DynamoDBPolicy = `{
	"Version": "2012-10-17",
	"Statement": [{
		"Effect": "Allow",
		"Action": [
			"dynamodb:PutItem"
		],
		"Resource": "arn:aws:dynamodb:*:*:table/*"
	}]
}`

type DynamoDBResources struct {
	ClicksTable *dynamodb.Table
	Policy      *iam.RolePolicy
}

func createDynamoDBResources(ctx *pulumi.Context, projectStackName string, role *iam.Role) (*DynamoDBResources, error) {
	// Attach a policy to allow writing to DynamoDB
	dynamoDBPolicy, err := iam.NewRolePolicy(ctx, projectStackName+"-dynamodb-policy", &iam.RolePolicyArgs{
		Role:   role.Name,
		Policy: pulumi.String(DynamoDBPolicy),
	})
	if err != nil {
		return nil, err
	}

	// Create the DynamoDB clicks table
	clicksTable, err := dynamodb.NewTable(ctx, projectStackName+"-dynamodb-clicks-table", &dynamodb.TableArgs{
		Name:           pulumi.String(projectStackName + "-clicks"),
		BillingMode:    pulumi.String("PAY_PER_REQUEST"),
		HashKey:        pulumi.String("cognito_user_id"),
		RangeKey:       pulumi.String("occurred_at"),
		StreamEnabled:  pulumi.Bool(true),
		StreamViewType: pulumi.String("NEW_IMAGE"),
		Attributes: dynamodb.TableAttributeArray{
			&dynamodb.TableAttributeArgs{
				Name: pulumi.String("cognito_user_id"),
				Type: pulumi.String("S"),
			},
			&dynamodb.TableAttributeArgs{
				Name: pulumi.String("occurred_at"),
				Type: pulumi.String("N"),
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return &DynamoDBResources{
		ClicksTable: clicksTable,
		Policy:      dynamoDBPolicy,
	}, nil
}
