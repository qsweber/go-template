package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/cognito"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type CognitoResources struct {
	UserPool       *cognito.UserPool
	UserPoolClient *cognito.UserPoolClient
}

func createCognitoResources(ctx *pulumi.Context, projectStackName string, domainName string, sesSourceArn pulumi.StringInput) (*CognitoResources, error) {
	// Create the Cognito User Pool
	userPool, err := cognito.NewUserPool(ctx, projectStackName+"-user-pool", &cognito.UserPoolArgs{
		AccountRecoverySetting: &cognito.UserPoolAccountRecoverySettingArgs{
			RecoveryMechanisms: cognito.UserPoolAccountRecoverySettingRecoveryMechanismArray{
				&cognito.UserPoolAccountRecoverySettingRecoveryMechanismArgs{
					Name:     pulumi.String("verified_email"),
					Priority: pulumi.Int(1),
				},
			},
		},
		AutoVerifiedAttributes: pulumi.StringArray{
			pulumi.String("email"),
		},
		DeletionProtection: pulumi.String("INACTIVE"),
		EmailConfiguration: &cognito.UserPoolEmailConfigurationArgs{
			EmailSendingAccount: pulumi.String("DEVELOPER"),
			FromEmailAddress:    pulumi.String("\"" + domainName + "\" <no-reply@" + domainName + ">"),
			SourceArn:           sesSourceArn,
		},
		MfaConfiguration: pulumi.String("OFF"),
		Name:             pulumi.String(domainName + "-user-pool"),
		PasswordPolicy: &cognito.UserPoolPasswordPolicyArgs{
			MinimumLength:                 pulumi.Int(8),
			RequireLowercase:              pulumi.Bool(true),
			RequireNumbers:                pulumi.Bool(true),
			RequireSymbols:                pulumi.Bool(true),
			RequireUppercase:              pulumi.Bool(true),
			TemporaryPasswordValidityDays: pulumi.Int(7),
		},
		SignInPolicy: &cognito.UserPoolSignInPolicyArgs{
			AllowedFirstAuthFactors: pulumi.StringArray{
				pulumi.String("PASSWORD"),
			},
		},
		UserPoolTier: pulumi.String("ESSENTIALS"),
		UsernameAttributes: pulumi.StringArray{
			pulumi.String("email"),
		},
		VerificationMessageTemplate: &cognito.UserPoolVerificationMessageTemplateArgs{
			DefaultEmailOption: pulumi.String("CONFIRM_WITH_CODE"),
			EmailMessage:       pulumi.String("Your verification code is {####}"),
			EmailSubject:       pulumi.String("Verify your email with " + domainName),
		},
	}, pulumi.Protect(true))
	if err != nil {
		return nil, err
	}

	userPoolClient, err := cognito.NewUserPoolClient(ctx, projectStackName+"-user-pool-client", &cognito.UserPoolClientArgs{
		AccessTokenValidity:   pulumi.Int(60),
		AuthSessionValidity:   pulumi.Int(3),
		EnableTokenRevocation: pulumi.Bool(true),
		ExplicitAuthFlows: pulumi.StringArray{
			pulumi.String("ALLOW_REFRESH_TOKEN_AUTH"),
			pulumi.String("ALLOW_USER_PASSWORD_AUTH"),
			pulumi.String("ALLOW_USER_SRP_AUTH"),
		},
		IdTokenValidity:            pulumi.Int(60),
		Name:                       pulumi.String(domainName + "-user-pool-client"),
		PreventUserExistenceErrors: pulumi.String("ENABLED"),
		ReadAttributes: pulumi.StringArray{
			pulumi.String("email"),
			pulumi.String("email_verified"),
			pulumi.String("name"),
		},
		RefreshTokenValidity: pulumi.Int(30),
		TokenValidityUnits: &cognito.UserPoolClientTokenValidityUnitsArgs{
			AccessToken:  pulumi.String("minutes"),
			IdToken:      pulumi.String("minutes"),
			RefreshToken: pulumi.String("days"),
		},
		UserPoolId: userPool.ID(),
		WriteAttributes: pulumi.StringArray{
			pulumi.String("email"),
			pulumi.String("name"),
		},
	}, pulumi.Protect(true))
	if err != nil {
		return nil, err
	}

	return &CognitoResources{
		UserPool:       userPool,
		UserPoolClient: userPoolClient,
	}, nil
}
