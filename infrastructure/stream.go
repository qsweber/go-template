package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/dynamodb"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/lambda"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const DynamoDBStreamPolicy = `{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Effect": "Allow",
			"Action": [
				"dynamodb:DescribeStream",
				"dynamodb:GetRecords",
				"dynamodb:GetShardIterator",
				"dynamodb:ListShards"
			],
			"Resource": "%s"
		},
		{
			"Effect": "Allow",
			"Action": [
				"dynamodb:ListStreams"
			],
			"Resource": "*"
		}
	]
}`

func createDynamoDBStreamResources(ctx *pulumi.Context, projectStackName string, clicksTable *dynamodb.Table) error {
	streamRole, err := iam.NewRole(ctx, projectStackName+"-dynamodb-stream-role", &iam.RoleArgs{
		AssumeRolePolicy: pulumi.String(RolePolicy),
	})
	if err != nil {
		return err
	}

	streamLogPolicy, err := iam.NewRolePolicy(ctx, projectStackName+"-dynamodb-stream-log-policy", &iam.RolePolicyArgs{
		Role:   streamRole.Name,
		Policy: pulumi.String(LogPolicy),
	})
	if err != nil {
		return err
	}

	streamPolicy, err := iam.NewRolePolicy(ctx, projectStackName+"-dynamodb-stream-policy", &iam.RolePolicyArgs{
		Role:   streamRole.Name,
		Policy: pulumi.Sprintf(DynamoDBStreamPolicy, clicksTable.StreamArn),
	})
	if err != nil {
		return err
	}

	streamFunction, err := lambda.NewFunction(ctx, projectStackName+"-dynamodb-stream-function", &lambda.FunctionArgs{
		Name:    pulumi.String(projectStackName + "-dynamodb-stream-function"),
		Handler: pulumi.String("bootstrap"),
		Role:    streamRole.Arn,
		Runtime: pulumi.String("provided.al2"),
		Code:    pulumi.NewFileArchive("../stream.zip"),
	}, pulumi.DependsOn([]pulumi.Resource{streamLogPolicy, streamPolicy}))
	if err != nil {
		return err
	}

	_, err = lambda.NewEventSourceMapping(ctx, projectStackName+"-dynamodb-stream-clicks-mapping", &lambda.EventSourceMappingArgs{
		EventSourceArn:   clicksTable.StreamArn,
		FunctionName:     streamFunction.Name,
		StartingPosition: pulumi.String("LATEST"),
		BatchSize:        pulumi.Int(1),
		Enabled:          pulumi.Bool(true),
	}, pulumi.DependsOn([]pulumi.Resource{streamFunction}))
	if err != nil {
		return err
	}

	return nil
}
