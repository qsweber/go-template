package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type DynamoDBResources struct {
	RecordsTable awsdynamodb.CfnTable
}

// createDynamoDBResources creates the app's single table. Every entity type lives in it,
// keyed by a generic partition key (pk, e.g. "USER#<id>") and sort key (sk, e.g. "USER").
func createDynamoDBResources(scope constructs.Construct, projectStackName string) *DynamoDBResources {
	recordsTable := awsdynamodb.NewCfnTable(scope, jsii.String("RecordsTable"), &awsdynamodb.CfnTableProps{
		TableName:   jsii.String(projectStackName + "-records"),
		BillingMode: jsii.String("PAY_PER_REQUEST"),
		KeySchema: []interface{}{
			&awsdynamodb.CfnTable_KeySchemaProperty{
				AttributeName: jsii.String("pk"),
				KeyType:       jsii.String("HASH"),
			},
			&awsdynamodb.CfnTable_KeySchemaProperty{
				AttributeName: jsii.String("sk"),
				KeyType:       jsii.String("RANGE"),
			},
		},
		AttributeDefinitions: []interface{}{
			&awsdynamodb.CfnTable_AttributeDefinitionProperty{
				AttributeName: jsii.String("pk"),
				AttributeType: jsii.String("S"),
			},
			&awsdynamodb.CfnTable_AttributeDefinitionProperty{
				AttributeName: jsii.String("sk"),
				AttributeType: jsii.String("S"),
			},
		},
		StreamSpecification: &awsdynamodb.CfnTable_StreamSpecificationProperty{
			StreamViewType: jsii.String("NEW_IMAGE"),
		},
	})

	return &DynamoDBResources{
		RecordsTable: recordsTable,
	}
}
