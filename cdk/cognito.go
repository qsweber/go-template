package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscognito"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsses"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type CognitoResources struct {
	UserPool       awscognito.CfnUserPool
	UserPoolClient awscognito.CfnUserPoolClient
}

func createCognitoResources(scope constructs.Construct, projectStackName string, cfg EnvConfig, sesIdentity awsses.CfnEmailIdentity) *CognitoResources {
	sesSourceArn := awscdk.Stack_Of(scope).FormatArn(&awscdk.ArnComponents{
		Service:      jsii.String("ses"),
		Resource:     jsii.String("identity"),
		ResourceName: jsii.String(cfg.DomainName),
	})

	userPool := awscognito.NewCfnUserPool(scope, jsii.String("UserPool"), &awscognito.CfnUserPoolProps{
		UserPoolName: jsii.String(cfg.DomainName + "-user-pool"),
		AccountRecoverySetting: &awscognito.CfnUserPool_AccountRecoverySettingProperty{
			RecoveryMechanisms: []interface{}{
				&awscognito.CfnUserPool_RecoveryOptionProperty{
					Name:     jsii.String("verified_email"),
					Priority: jsii.Number(1),
				},
			},
		},
		AutoVerifiedAttributes: jsii.Strings("email"),
		DeletionProtection:     jsii.String("INACTIVE"),
		EmailConfiguration: &awscognito.CfnUserPool_EmailConfigurationProperty{
			EmailSendingAccount: jsii.String("DEVELOPER"),
			From:                jsii.String("\"" + cfg.DomainName + "\" <no-reply@" + cfg.DomainName + ">"),
			SourceArn:           sesSourceArn,
		},
		MfaConfiguration: jsii.String("OFF"),
		Policies: &awscognito.CfnUserPool_PoliciesProperty{
			PasswordPolicy: &awscognito.CfnUserPool_PasswordPolicyProperty{
				MinimumLength:                 jsii.Number(8),
				RequireLowercase:              jsii.Bool(true),
				RequireNumbers:                jsii.Bool(true),
				RequireSymbols:                jsii.Bool(true),
				RequireUppercase:              jsii.Bool(true),
				TemporaryPasswordValidityDays: jsii.Number(7),
			},
			SignInPolicy: &awscognito.CfnUserPool_SignInPolicyProperty{
				AllowedFirstAuthFactors: jsii.Strings("PASSWORD", "WEB_AUTHN"),
			},
		},
		UserPoolTier:       jsii.String("ESSENTIALS"),
		UsernameAttributes: jsii.Strings("email"),
		VerificationMessageTemplate: &awscognito.CfnUserPool_VerificationMessageTemplateProperty{
			DefaultEmailOption: jsii.String("CONFIRM_WITH_CODE"),
			EmailMessage:       jsii.String("Your verification code is {####}"),
			EmailSubject:       jsii.String("Verify your email with " + cfg.DomainName),
		},
		// Lets users register and sign in with a passkey as an alternative to a
		// password, on top of the existing PASSWORD first factor above.
		WebAuthnFactorConfiguration: jsii.String("SINGLE_FACTOR"),
		WebAuthnRelyingPartyId:      jsii.String(cfg.DomainName),
		WebAuthnUserVerification:    jsii.String("preferred"),
	})
	userPool.AddDependency(sesIdentity)
	userPool.ApplyRemovalPolicy(awscdk.RemovalPolicy_RETAIN, nil)

	userPoolClient := awscognito.NewCfnUserPoolClient(scope, jsii.String("UserPoolClient"), &awscognito.CfnUserPoolClientProps{
		UserPoolId:            userPool.Ref(),
		AccessTokenValidity:   jsii.Number(60),
		AuthSessionValidity:   jsii.Number(3),
		EnableTokenRevocation: jsii.Bool(true),
		ExplicitAuthFlows: jsii.Strings(
			"ALLOW_REFRESH_TOKEN_AUTH",
			"ALLOW_USER_PASSWORD_AUTH",
			"ALLOW_USER_SRP_AUTH",
			// USER_AUTH is the selection-based flow that Amplify's signIn(...)
			// uses under the hood for passkey/WebAuthn authentication.
			"ALLOW_USER_AUTH",
		),
		IdTokenValidity:            jsii.Number(60),
		ClientName:                 jsii.String(cfg.DomainName + "-user-pool-client"),
		PreventUserExistenceErrors: jsii.String("ENABLED"),
		ReadAttributes: jsii.Strings(
			"email",
			"email_verified",
			"name",
		),
		RefreshTokenValidity: jsii.Number(30),
		TokenValidityUnits: &awscognito.CfnUserPoolClient_TokenValidityUnitsProperty{
			AccessToken:  jsii.String("minutes"),
			IdToken:      jsii.String("minutes"),
			RefreshToken: jsii.String("days"),
		},
		WriteAttributes: jsii.Strings("email", "name"),
	})
	userPoolClient.ApplyRemovalPolicy(awscdk.RemovalPolicy_RETAIN, nil)

	return &CognitoResources{
		UserPool:       userPool,
		UserPoolClient: userPoolClient,
	}
}
