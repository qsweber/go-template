package main

import (
	"fmt"

	"github.com/aws/aws-cdk-go/awscdk/v2/awsapigateway"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3assets"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

var gatewayPolicyDocument = map[string]interface{}{
	"Version": "2012-10-17",
	"Statement": []interface{}{
		map[string]interface{}{
			"Action": "sts:AssumeRole",
			"Principal": map[string]interface{}{
				"Service": "lambda.amazonaws.com",
			},
			"Effect": "Allow",
			"Sid":    "",
		},
		map[string]interface{}{
			"Action":    "execute-api:Invoke",
			"Resource":  "*",
			"Principal": "*",
			"Effect":    "Allow",
			"Sid":       "",
		},
	},
}

type APIGatewayResources struct {
	Function awslambda.CfnFunction
	Gateway  awsapigateway.CfnRestApi
}

func createAPIGatewayResources(
	scope constructs.Construct,
	projectStackName string,
	cfg EnvConfig,
	role awsiam.CfnRole,
	dynamoResources *DynamoDBResources,
	cognitoResources *CognitoResources,
) *APIGatewayResources {
	envVars := map[string]*string{
		"CLICKS_TABLE": dynamoResources.ClicksTable.Ref(),
	}
	if cognitoResources != nil {
		envVars["COGNITO_REGION"] = jsii.String(cfg.Region)
		envVars["COGNITO_USER_POOL_ID"] = cognitoResources.UserPool.AttrUserPoolId()
		envVars["COGNITO_CLIENT_ID"] = cognitoResources.UserPoolClient.AttrClientId()
	}

	asset := awss3assets.NewAsset(scope, jsii.String("APIGatewayFunctionAsset"), &awss3assets.AssetProps{
		Path: jsii.String("../handler.zip"),
	})

	function := awslambda.NewCfnFunction(scope, jsii.String("APIGatewayFunction"), &awslambda.CfnFunctionProps{
		FunctionName: jsii.String(projectStackName + "-apigateway-function"),
		Handler:      jsii.String("bootstrap"),
		Role:         role.AttrArn(),
		Runtime:      jsii.String("provided.al2"),
		Code: &awslambda.CfnFunction_CodeProperty{
			S3Bucket: asset.S3BucketName(),
			S3Key:    asset.S3ObjectKey(),
		},
		Environment: &awslambda.CfnFunction_EnvironmentProperty{
			Variables: envVars,
		},
	})
	function.AddDependency(role)

	gateway := awsapigateway.NewCfnRestApi(scope, jsii.String("RestApi"), &awsapigateway.CfnRestApiProps{
		Name:        jsii.String(projectStackName + "-api"),
		Description: jsii.String("An API Gateway for the " + projectStackName + " function"),
		Policy:      gatewayPolicyDocument,
	})

	apiResource := awsapigateway.NewCfnResource(scope, jsii.String("GatewayResource"), &awsapigateway.CfnResourceProps{
		RestApiId: gateway.AttrRestApiId(),
		PathPart:  jsii.String("{proxy+}"),
		ParentId:  gateway.AttrRootResourceId(),
	})

	anyMethod := awsapigateway.NewCfnMethod(scope, jsii.String("AnyMethod"), &awsapigateway.CfnMethodProps{
		HttpMethod:        jsii.String("ANY"),
		AuthorizationType: jsii.String("NONE"),
		RestApiId:         gateway.AttrRestApiId(),
		ResourceId:        apiResource.AttrResourceId(),
		Integration: &awsapigateway.CfnMethod_IntegrationProperty{
			Type:                  jsii.String("AWS_PROXY"),
			IntegrationHttpMethod: jsii.String("POST"),
			Uri: jsii.String(fmt.Sprintf(
				"arn:aws:apigateway:%s:lambda:path/2015-03-31/functions/%s/invocations",
				cfg.Region,
				*function.AttrArn(),
			)),
		},
	})
	anyMethod.AddDependency(apiResource)

	permission := awslambda.NewCfnPermission(scope, jsii.String("APIPermission"), &awslambda.CfnPermissionProps{
		Action:       jsii.String("lambda:InvokeFunction"),
		FunctionName: function.Ref(),
		Principal:    jsii.String("apigateway.amazonaws.com"),
		SourceArn: jsii.String(fmt.Sprintf(
			"arn:aws:execute-api:%s:%s:%s/*/*/*",
			cfg.Region,
			cfg.Account,
			*gateway.AttrRestApiId(),
		)),
	})
	permission.AddDependency(apiResource)

	deployment := awsapigateway.NewCfnDeployment(scope, jsii.String("Deployment"), &awsapigateway.CfnDeploymentProps{
		RestApiId:   gateway.AttrRestApiId(),
		Description: jsii.String("API deployment"),
	})
	deployment.AddDependency(apiResource)
	deployment.AddDependency(anyMethod)
	deployment.AddDependency(function)
	deployment.AddDependency(permission)

	stage := awsapigateway.NewCfnStage(scope, jsii.String("Stage"), &awsapigateway.CfnStageProps{
		RestApiId:    gateway.AttrRestApiId(),
		StageName:    jsii.String(cfg.Env),
		DeploymentId: deployment.AttrDeploymentId(),
	})
	stage.AddDependency(deployment)

	return &APIGatewayResources{
		Function: function,
		Gateway:  gateway,
	}
}
