package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3assets"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

func createDynamoDBStreamResources(scope constructs.Construct, projectStackName string, cfg EnvConfig, clicksTable awsdynamodb.CfnTable) {
	streamPolicyDocument := map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []interface{}{
			map[string]interface{}{
				"Effect": "Allow",
				"Action": []interface{}{
					"dynamodb:DescribeStream",
					"dynamodb:GetRecords",
					"dynamodb:GetShardIterator",
					"dynamodb:ListShards",
				},
				"Resource": *clicksTable.AttrStreamArn(),
			},
			map[string]interface{}{
				"Effect": "Allow",
				"Action": []interface{}{
					"dynamodb:ListStreams",
				},
				"Resource": "*",
			},
		},
	}

	streamRole := awsiam.NewCfnRole(scope, jsii.String("DynamoDBStreamRole"), &awsiam.CfnRoleProps{
		RoleName:                 jsii.String(cfg.StreamRoleName),
		AssumeRolePolicyDocument: assumeRolePolicyDocument,
		Policies: []interface{}{
			&awsiam.CfnRole_PolicyProperty{
				PolicyName:     jsii.String(projectStackName + "-dynamodb-stream-log-policy"),
				PolicyDocument: logPolicyDocument,
			},
			&awsiam.CfnRole_PolicyProperty{
				PolicyName:     jsii.String(projectStackName + "-dynamodb-stream-policy"),
				PolicyDocument: streamPolicyDocument,
			},
		},
	})

	asset := awss3assets.NewAsset(scope, jsii.String("DynamoDBStreamFunctionAsset"), &awss3assets.AssetProps{
		Path: jsii.String("../stream.zip"),
	})

	streamFunction := awslambda.NewCfnFunction(scope, jsii.String("DynamoDBStreamFunction"), &awslambda.CfnFunctionProps{
		FunctionName: jsii.String(projectStackName + "-dynamodb-stream-function"),
		Handler:      jsii.String("bootstrap"),
		Role:         streamRole.AttrArn(),
		Runtime:      jsii.String("provided.al2"),
		Code: &awslambda.CfnFunction_CodeProperty{
			S3Bucket: asset.S3BucketName(),
			S3Key:    asset.S3ObjectKey(),
		},
	})
	streamFunction.AddDependency(streamRole)

	mapping := awslambda.NewCfnEventSourceMapping(scope, jsii.String("DynamoDBStreamClicksMapping"), &awslambda.CfnEventSourceMappingProps{
		EventSourceArn:   clicksTable.AttrStreamArn(),
		FunctionName:     streamFunction.Ref(),
		StartingPosition: jsii.String("LATEST"),
		BatchSize:        jsii.Number(1),
		Enabled:          jsii.Bool(true),
	})
	mapping.AddDependency(streamFunction)
}
