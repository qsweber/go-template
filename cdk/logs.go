package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslogs"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

// logRetentionDays applies to every log group the stack creates, so no logs are kept forever.
const logRetentionDays = 30

func newLogGroup(scope constructs.Construct, id string, name string) awslogs.CfnLogGroup {
	return awslogs.NewCfnLogGroup(scope, jsii.String(id), &awslogs.CfnLogGroupProps{
		LogGroupName:    jsii.String(name),
		RetentionInDays: jsii.Number(logRetentionDays),
	})
}

// functionLogging creates a log group for a Lambda function and returns the LoggingConfig that
// points the function at it. Lambda's automatic /aws/lambda/<function-name> groups never expire
// and can't be adopted by CloudFormation once they exist, so each function logs to its own
// stack-managed group, /<stack>/<suffix>, instead.
func functionLogging(scope constructs.Construct, id string, projectStackName string, suffix string) *awslambda.CfnFunction_LoggingConfigProperty {
	logGroup := newLogGroup(scope, id, "/"+projectStackName+"/"+suffix)
	return &awslambda.CfnFunction_LoggingConfigProperty{
		LogGroup: logGroup.Ref(),
	}
}
