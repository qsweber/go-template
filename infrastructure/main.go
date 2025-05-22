package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/apigateway"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/lambda"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
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

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		account, err := aws.GetCallerIdentity(ctx, &aws.GetCallerIdentityArgs{}, nil)
		if err != nil {
			return err
		}

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

		// Create the lambda using the args.
		function, err := lambda.NewFunction(
			ctx,
			projectStackName+"-function",
			&lambda.FunctionArgs{
				Handler: pulumi.String("bootstrap"),
				Role:    role.Arn,
				Runtime: pulumi.String("provided.al2"),
				Code:    pulumi.NewFileArchive("../handler.zip"),
			},
			pulumi.DependsOn([]pulumi.Resource{logPolicy}),
		)
		if err != nil {
			return err
		}

		// Create a new API Gateway.
		gateway, err := apigateway.NewRestApi(ctx, projectStackName+"-api", &apigateway.RestApiArgs{
			Name:        pulumi.String(projectStackName + "-api"),
			Description: pulumi.String("An API Gateway for the " + projectStackName + " function"),
			Policy:      pulumi.String(GatewayPolicy)})
		if err != nil {
			return err
		}

		// Add a resource to the API Gateway.
		// This makes the API Gateway accept requests on "/{message}".
		apiresource, err := apigateway.NewResource(ctx, projectStackName+"-gateway-resource", &apigateway.ResourceArgs{
			RestApi:  gateway.ID(),
			PathPart: pulumi.String("{proxy+}"),
			ParentId: gateway.RootResourceId,
		})
		if err != nil {
			return err
		}

		// Add a method to the API Gateway.
		_, err = apigateway.NewMethod(ctx, projectStackName+"-any-method", &apigateway.MethodArgs{
			HttpMethod:    pulumi.String("ANY"),
			Authorization: pulumi.String("NONE"),
			RestApi:       gateway.ID(),
			ResourceId:    apiresource.ID(),
		})
		if err != nil {
			return err
		}

		// Add an integration to the API Gateway.
		// This makes communication between the API Gateway and the Lambda function work
		_, err = apigateway.NewIntegration(ctx, projectStackName+"-lambda-integration", &apigateway.IntegrationArgs{
			HttpMethod:            pulumi.String("ANY"),
			IntegrationHttpMethod: pulumi.String("POST"),
			ResourceId:            apiresource.ID(),
			RestApi:               gateway.ID(),
			Type:                  pulumi.String("AWS_PROXY"),
			Uri:                   function.InvokeArn,
		})
		if err != nil {
			return err
		}

		// Add a resource based policy to the Lambda function.
		// This is the final step and allows AWS API Gateway to communicate with the AWS Lambda function
		permission, err := lambda.NewPermission(ctx, projectStackName+"-api-permission", &lambda.PermissionArgs{
			Action:    pulumi.String("lambda:InvokeFunction"),
			Function:  function.Name,
			Principal: pulumi.String("apigateway.amazonaws.com"),
			SourceArn: pulumi.Sprintf("arn:aws:execute-api:%s:%s:%s/*/*/*", region.Name, account.AccountId, gateway.ID()),
		}, pulumi.DependsOn([]pulumi.Resource{apiresource}))
		if err != nil {
			return err
		}

		// Create a new deployment
		deployment, err := apigateway.NewDeployment(ctx, projectStackName+"-deployment", &apigateway.DeploymentArgs{
			Description: pulumi.String("UpperCase API deployment"),
			RestApi:     gateway.ID(),
		}, pulumi.DependsOn([]pulumi.Resource{apiresource, function, permission}))
		if err != nil {
			return err
		}

		// Create a new stage
		_, err = apigateway.NewStage(ctx, projectStackName+"-stage", &apigateway.StageArgs{
			RestApi:    gateway.ID(),
			StageName:  pulumi.String(ctx.Stack()),
			Deployment: deployment.ID(),
		})

		ctx.Export("Lambda Name", function.Name)
		ctx.Export("invocation URL", pulumi.Sprintf("https://%s.execute-api.%s.amazonaws.com/%s/{message}", gateway.ID(), region.Name, ctx.Stack()))

		return nil
	})
}
