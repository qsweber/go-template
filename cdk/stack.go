package main

import (
	"fmt"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

func NewGoTemplateStack(scope constructs.Construct, id string, cfg EnvConfig, props *awscdk.StackProps) awscdk.Stack {
	stack := awscdk.NewStack(scope, jsii.String(id), props)

	projectStackName := "go-template-" + cfg.Env

	role := createBaseRole(stack, projectStackName, cfg)

	dynamoResources := createDynamoDBResources(stack, projectStackName)

	createDynamoDBStreamResources(stack, projectStackName, cfg, dynamoResources.ClicksTable)

	sesResources := createSESResources(stack, projectStackName, cfg)

	cognitoResources := createCognitoResources(stack, projectStackName, cfg, sesResources.EmailIdentity)

	apigatewayResources := createAPIGatewayResources(stack, projectStackName, cfg, role, dynamoResources, cognitoResources)

	createCustomDomainResources(stack, cfg, apigatewayResources.Gateway, cfg.Env)

	awscdk.NewCfnOutput(stack, jsii.String("LambdaName"), &awscdk.CfnOutputProps{
		Value: apigatewayResources.Function.FunctionName(),
	})
	awscdk.NewCfnOutput(stack, jsii.String("CognitoUserPoolId"), &awscdk.CfnOutputProps{
		Value: cognitoResources.UserPool.AttrUserPoolId(),
	})
	awscdk.NewCfnOutput(stack, jsii.String("CognitoUserPoolClientId"), &awscdk.CfnOutputProps{
		Value: cognitoResources.UserPoolClient.AttrClientId(),
	})
	awscdk.NewCfnOutput(stack, jsii.String("InvocationURL"), &awscdk.CfnOutputProps{
		Value: jsii.String(fmt.Sprintf(
			"https://%s.execute-api.%s.amazonaws.com/%s/{message}",
			*apigatewayResources.Gateway.AttrRestApiId(),
			cfg.Region,
			cfg.Env,
		)),
	})
	awscdk.NewCfnOutput(stack, jsii.String("CustomDomainURL"), &awscdk.CfnOutputProps{
		Value: jsii.String(fmt.Sprintf("https://%s/{message}", cfg.ApiDomainName)),
	})

	return stack
}
