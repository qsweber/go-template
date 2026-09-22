package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type DynamoDBResources struct {
	ClicksTable awsdynamodb.CfnTable
}

func createDynamoDBResources(scope constructs.Construct, projectStackName string) *DynamoDBResources {
	clicksTable := awsdynamodb.NewCfnTable(scope, jsii.String("ClicksTable"), &awsdynamodb.CfnTableProps{
		TableName:   jsii.String(projectStackName + "-clicks"),
		BillingMode: jsii.String("PAY_PER_REQUEST"),
		KeySchema: []interface{}{
			&awsdynamodb.CfnTable_KeySchemaProperty{
				AttributeName: jsii.String("cognito_user_id"),
				KeyType:       jsii.String("HASH"),
			},
			&awsdynamodb.CfnTable_KeySchemaProperty{
				AttributeName: jsii.String("occurred_at"),
				KeyType:       jsii.String("RANGE"),
			},
		},
		AttributeDefinitions: []interface{}{
			&awsdynamodb.CfnTable_AttributeDefinitionProperty{
				AttributeName: jsii.String("cognito_user_id"),
				AttributeType: jsii.String("S"),
			},
			&awsdynamodb.CfnTable_AttributeDefinitionProperty{
				AttributeName: jsii.String("occurred_at"),
				AttributeType: jsii.String("N"),
			},
		},
		StreamSpecification: &awsdynamodb.CfnTable_StreamSpecificationProperty{
			StreamViewType: jsii.String("NEW_IMAGE"),
		},
	})

	return &DynamoDBResources{
		ClicksTable: clicksTable,
	}
}
