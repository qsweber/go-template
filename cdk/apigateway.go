package main

import (
	"fmt"

	"github.com/aws/aws-cdk-go/awscdk/v2/awsapigatewayv2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3assets"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

// Burst protection for the API. A flood beyond the stage limits gets 429s from API Gateway before
// it reaches (and is billed as) Lambda invocations, and the reserved concurrency caps how many API
// Lambda instances can run at once, so this function can't drain the account-wide concurrency
// pool that every other Lambda in the account and region shares. Raise these if real traffic
// approaches them.
const (
	apiThrottlingRateLimit  = 50  // steady-state requests per second, across all clients
	apiThrottlingBurstLimit = 100 // requests that can arrive at once above the steady rate
	apiReservedConcurrency  = 20  // max concurrent API Lambda instances
)

type APIGatewayResources struct {
	Function awslambda.CfnFunction
	Api      awsapigatewayv2.CfnApi
	Stage    awsapigatewayv2.CfnStage
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
		"RECORDS_TABLE": dynamoResources.RecordsTable.Ref(),
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
		ReservedConcurrentExecutions: jsii.Number(apiReservedConcurrency),
		Environment: &awslambda.CfnFunction_EnvironmentProperty{
			Variables: envVars,
		},
	})
	function.AddDependency(role)

	// An HTTP API (API Gateway v2) with a single catch-all route: all routing, CORS, and auth
	// happen in the Go handler, so none of the REST API's extra features are needed.
	api := awsapigatewayv2.NewCfnApi(scope, jsii.String("HttpApi"), &awsapigatewayv2.CfnApiProps{
		Name:         jsii.String(projectStackName + "-api"),
		Description:  jsii.String("An HTTP API for the " + projectStackName + " function"),
		ProtocolType: jsii.String("HTTP"),
	})

	integration := awsapigatewayv2.NewCfnIntegration(scope, jsii.String("HttpApiIntegration"), &awsapigatewayv2.CfnIntegrationProps{
		ApiId:                api.Ref(),
		IntegrationType:      jsii.String("AWS_PROXY"),
		IntegrationUri:       function.AttrArn(),
		PayloadFormatVersion: jsii.String("2.0"),
	})

	route := awsapigatewayv2.NewCfnRoute(scope, jsii.String("HttpApiDefaultRoute"), &awsapigatewayv2.CfnRouteProps{
		ApiId:    api.Ref(),
		RouteKey: jsii.String("$default"),
		Target:   jsii.String(fmt.Sprintf("integrations/%s", *integration.Ref())),
	})

	// The $default stage is served at the root of the invoke URL (no stage name in the path)
	// and redeploys automatically whenever routes or integrations change.
	stage := awsapigatewayv2.NewCfnStage(scope, jsii.String("HttpApiStage"), &awsapigatewayv2.CfnStageProps{
		ApiId:      api.Ref(),
		StageName:  jsii.String("$default"),
		AutoDeploy: jsii.Bool(true),
		DefaultRouteSettings: &awsapigatewayv2.CfnStage_RouteSettingsProperty{
			ThrottlingRateLimit:  jsii.Number(apiThrottlingRateLimit),
			ThrottlingBurstLimit: jsii.Number(apiThrottlingBurstLimit),
		},
	})
	stage.AddDependency(route)

	awslambda.NewCfnPermission(scope, jsii.String("APIPermission"), &awslambda.CfnPermissionProps{
		Action:       jsii.String("lambda:InvokeFunction"),
		FunctionName: function.Ref(),
		Principal:    jsii.String("apigateway.amazonaws.com"),
		SourceArn: jsii.String(fmt.Sprintf(
			"arn:aws:execute-api:%s:%s:%s/*",
			cfg.Region,
			cfg.Account,
			*api.Ref(),
		)),
	})

	return &APIGatewayResources{
		Function: function,
		Api:      api,
		Stage:    stage,
	}
}
