package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

const RolePolicy = `{
	"Version": "2012-10-17",
	"Statement": [{
		"Sid": "",
		"Effect": "Allow",
		"Principal": {
			"Service": "lambda.amazonaws.com"
		},
		"Action": "sts:AssumeRole"
	}]
}`

const LogPolicy = `{
	"Version": "2012-10-17",
	"Statement": [{
		"Effect": "Allow",
		"Action": [
			"logs:CreateLogGroup",
			"logs:CreateLogStream",
			"logs:PutLogEvents"
		],
		"Resource": "arn:aws:logs:*:*:*"
	}]
}`

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Load Cognito configuration (optional)
		cfg := config.New(ctx, ctx.Project())
		cognitoRegion := cfg.Get("cognitoRegion")
		cognitoUserPoolId := cfg.Get("cognitoUserPoolId")
		cognitoClientId := cfg.Get("cognitoClientId")

		region, err := aws.GetRegion(ctx, &aws.GetRegionArgs{})
		if err != nil {
			return err
		}

		projectStackName := ctx.Project() + "-" + ctx.Stack()

		// Create an IAM role.
		role, err := iam.NewRole(ctx, projectStackName+"-task-exec-role", &iam.RoleArgs{
			AssumeRolePolicy: pulumi.String(RolePolicy),
		})
		if err != nil {
			return err
		}

		// Attach a policy to allow writing logs to CloudWatch
		logPolicy, err := iam.NewRolePolicy(ctx, projectStackName+"-lambda-log-policy", &iam.RolePolicyArgs{
			Role:   role.Name,
			Policy: pulumi.String(LogPolicy),
		})
		if err != nil {
			return err
		}

		// Create DynamoDB resources
		dynamoResources, err := createDynamoDBResources(ctx, projectStackName, role)
		if err != nil {
			return err
		}

		// Create the DynamoDB stream consumer Lambda
		if err := createDynamoDBStreamResources(ctx, projectStackName, dynamoResources.ClicksTable); err != nil {
			return err
		}

		apigatewayResources, err := createAPIGatewayResources(ctx, projectStackName, role, logPolicy, dynamoResources, cognitoRegion, cognitoUserPoolId, cognitoClientId)
		if err != nil {
			return err
		}

		ctx.Export("Lambda Name", apigatewayResources.Function.Name)
		ctx.Export("invocation URL", pulumi.Sprintf("https://%s.execute-api.%s.amazonaws.com/%s/{message}", apigatewayResources.Gateway.ID(), region.Name, ctx.Stack()))

		return nil
	})
}
