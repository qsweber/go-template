package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/apigateway"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/lambda"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const GatewayPolicy = `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    },
    {
      "Action": "execute-api:Invoke",
      "Resource": "*",
      "Principal": "*",
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}`

type APIGatewayResources struct {
	Function *lambda.Function
	Gateway  *apigateway.RestApi
}

func createAPIGatewayResources(
	ctx *pulumi.Context,
	projectStackName string,
	role *iam.Role,
	logPolicy *iam.RolePolicy,
	dynamoResources *DynamoDBResources,
	cognitoRegion string,
	cognitoUserPoolId string,
	cognitoClientId string,
) (*APIGatewayResources, error) {
	envVars := pulumi.StringMap{
		"CLICKS_TABLE": dynamoResources.ClicksTable.Name,
	}
	if cognitoRegion != "" && cognitoUserPoolId != "" && cognitoClientId != "" {
		envVars["COGNITO_REGION"] = pulumi.String(cognitoRegion)
		envVars["COGNITO_USER_POOL_ID"] = pulumi.String(cognitoUserPoolId)
		envVars["COGNITO_CLIENT_ID"] = pulumi.String(cognitoClientId)
	}
	environment := &lambda.FunctionEnvironmentArgs{
		Variables: envVars,
	}

	function, err := lambda.NewFunction(
		ctx,
		projectStackName+"-apigateway-function",
		&lambda.FunctionArgs{
			Name:        pulumi.String(projectStackName + "-apigateway-function"),
			Handler:     pulumi.String("bootstrap"),
			Role:        role.Arn,
			Runtime:     pulumi.String("provided.al2"),
			Code:        pulumi.NewFileArchive("../handler.zip"),
			Environment: environment,
		},
		pulumi.DependsOn([]pulumi.Resource{logPolicy, dynamoResources.Policy}),
	)
	if err != nil {
		return nil, err
	}

	account, err := aws.GetCallerIdentity(ctx, &aws.GetCallerIdentityArgs{}, nil)
	if err != nil {
		return nil, err
	}

	region, err := aws.GetRegion(ctx, &aws.GetRegionArgs{})
	if err != nil {
		return nil, err
	}

	gateway, err := apigateway.NewRestApi(ctx, projectStackName+"-api", &apigateway.RestApiArgs{
		Name:        pulumi.String(projectStackName + "-api"),
		Description: pulumi.String("An API Gateway for the " + projectStackName + " function"),
		Policy:      pulumi.String(GatewayPolicy),
	})
	if err != nil {
		return nil, err
	}

	apiresource, err := apigateway.NewResource(ctx, projectStackName+"-gateway-resource", &apigateway.ResourceArgs{
		RestApi:  gateway.ID(),
		PathPart: pulumi.String("{proxy+}"),
		ParentId: gateway.RootResourceId,
	})
	if err != nil {
		return nil, err
	}

	anyMethod, err := apigateway.NewMethod(ctx, projectStackName+"-any-method", &apigateway.MethodArgs{
		HttpMethod:    pulumi.String("ANY"),
		Authorization: pulumi.String("NONE"),
		RestApi:       gateway.ID(),
		ResourceId:    apiresource.ID(),
	})
	if err != nil {
		return nil, err
	}

	lambdaIntegration, err := apigateway.NewIntegration(ctx, projectStackName+"-lambda-integration", &apigateway.IntegrationArgs{
		HttpMethod:            pulumi.String("ANY"),
		IntegrationHttpMethod: pulumi.String("POST"),
		ResourceId:            apiresource.ID(),
		RestApi:               gateway.ID(),
		Type:                  pulumi.String("AWS_PROXY"),
		Uri:                   function.InvokeArn,
	})
	if err != nil {
		return nil, err
	}

	permission, err := lambda.NewPermission(ctx, projectStackName+"-api-permission", &lambda.PermissionArgs{
		Action:    pulumi.String("lambda:InvokeFunction"),
		Function:  function.Name,
		Principal: pulumi.String("apigateway.amazonaws.com"),
		SourceArn: pulumi.Sprintf("arn:aws:execute-api:%s:%s:%s/*/*/*", region.Name, account.AccountId, gateway.ID()),
	}, pulumi.DependsOn([]pulumi.Resource{apiresource}))
	if err != nil {
		return nil, err
	}

	deployment, err := apigateway.NewDeployment(ctx, projectStackName+"-deployment", &apigateway.DeploymentArgs{
		Description: pulumi.String("API deployment"),
		RestApi:     gateway.ID(),
		Triggers: pulumi.StringMap{
			"apigateway-resource":    apiresource.ID().ToStringOutput(),
			"apigateway-method":      anyMethod.ID().ToStringOutput(),
			"apigateway-integration": function.InvokeArn,
		},
	}, pulumi.DependsOn([]pulumi.Resource{apiresource, anyMethod, lambdaIntegration, function, permission}))
	if err != nil {
		return nil, err
	}

	_, err = apigateway.NewStage(ctx, projectStackName+"-stage", &apigateway.StageArgs{
		RestApi:    gateway.ID(),
		StageName:  pulumi.String(ctx.Stack()),
		Deployment: deployment.ID(),
	})
	if err != nil {
		return nil, err
	}

	return &APIGatewayResources{
		Function: function,
		Gateway:  gateway,
	}, nil
}
