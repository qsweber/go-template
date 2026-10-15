package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3assets"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

// createPostConfirmationFunction creates the Cognito PostConfirmation trigger, which writes a
// USER record to the records table whenever a user confirms their sign-up. It is wired into
// the user pool (and granted invoke permission) in createCognitoResources.
func createPostConfirmationFunction(scope constructs.Construct, projectStackName string, role awsiam.CfnRole, recordsTable awsdynamodb.CfnTable) awslambda.CfnFunction {
	asset := awss3assets.NewAsset(scope, jsii.String("PostConfirmationFunctionAsset"), &awss3assets.AssetProps{
		Path: jsii.String("../postconfirmation.zip"),
	})

	function := awslambda.NewCfnFunction(scope, jsii.String("PostConfirmationFunction"), &awslambda.CfnFunctionProps{
		FunctionName: jsii.String(projectStackName + "-cognito-post-confirmation-function"),
		Handler:      jsii.String("bootstrap"),
		Role:         role.AttrArn(),
		Runtime:      jsii.String("provided.al2"),
		Code: &awslambda.CfnFunction_CodeProperty{
			S3Bucket: asset.S3BucketName(),
			S3Key:    asset.S3ObjectKey(),
		},
		Environment: &awslambda.CfnFunction_EnvironmentProperty{
			Variables: map[string]*string{
				"RECORDS_TABLE": recordsTable.Ref(),
			},
		},
	})
	function.AddDependency(role)

	return function
}
