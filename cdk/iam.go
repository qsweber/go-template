package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

var assumeRolePolicyDocument = map[string]interface{}{
	"Version": "2012-10-17",
	"Statement": []interface{}{
		map[string]interface{}{
			"Sid":    "",
			"Effect": "Allow",
			"Principal": map[string]interface{}{
				"Service": "lambda.amazonaws.com",
			},
			"Action": "sts:AssumeRole",
		},
	},
}

var logPolicyDocument = map[string]interface{}{
	"Version": "2012-10-17",
	"Statement": []interface{}{
		map[string]interface{}{
			"Effect": "Allow",
			"Action": []interface{}{
				"logs:CreateLogGroup",
				"logs:CreateLogStream",
				"logs:PutLogEvents",
			},
			"Resource": "arn:aws:logs:*:*:*",
		},
	},
}

var dynamoDBPolicyDocument = map[string]interface{}{
	"Version": "2012-10-17",
	"Statement": []interface{}{
		map[string]interface{}{
			"Effect": "Allow",
			"Action": []interface{}{
				"dynamodb:PutItem",
				"dynamodb:DescribeTable",
			},
			"Resource": "arn:aws:dynamodb:*:*:table/*",
		},
	},
}

// createBaseRole recreates the Lambda execution role Pulumi built from two resources
// (an iam.Role plus two attached iam.RolePolicy resources for logging and DynamoDB access).
// CloudFormation has no standalone "attached role policy" resource for inline policies -
// they are just entries in the Role's own Policies property - so both inline policies are
// embedded directly on the CfnRole here.
func createBaseRole(scope constructs.Construct, projectStackName string, cfg EnvConfig) awsiam.CfnRole {
	role := awsiam.NewCfnRole(scope, jsii.String("TaskExecRole"), &awsiam.CfnRoleProps{
		RoleName:                 jsii.String(cfg.TaskExecRoleName),
		AssumeRolePolicyDocument: assumeRolePolicyDocument,
		Policies: []interface{}{
			&awsiam.CfnRole_PolicyProperty{
				PolicyName:     jsii.String(projectStackName + "-lambda-log-policy"),
				PolicyDocument: logPolicyDocument,
			},
			&awsiam.CfnRole_PolicyProperty{
				PolicyName:     jsii.String(projectStackName + "-dynamodb-policy"),
				PolicyDocument: dynamoDBPolicyDocument,
			},
		},
	})

	return role
}
