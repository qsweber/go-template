package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/jsii-runtime-go"
)

// EnvConfig mirrors what used to be per-stack Pulumi config (Pulumi.dev.yaml / Pulumi.prod.yaml),
// plus the pre-existing physical resource names/values that need to be preserved so that
// `cdk import` can adopt the already-deployed AWS resources without replacing them.
type EnvConfig struct {
	Env        string
	Account    string
	Region     string
	DomainName string
	ZoneID     string

	// Physical names of already-deployed resources that Pulumi auto-named (random suffix).
	// These must match exactly for `cdk import` to adopt the existing resources.
	TaskExecRoleName string
	StreamRoleName   string

	// The existing SES domain verification TXT record value (legacy verification path,
	// kept alongside Easy DKIM for continuity). Read from live Route53 state.
	SESVerificationToken string
}

func main() {
	app := awscdk.NewApp(nil)

	account := "120356305272"

	NewGoTemplateStack(app, "go-template-dev", EnvConfig{
		Env:                  "dev",
		Account:              account,
		Region:               "us-west-2",
		DomainName:           "dev-template.quinnweber.com",
		ZoneID:               "Z07290422T6PHN4TKVNW0",
		TaskExecRoleName:     "go-template-dev-task-exec-role-cba5b03",
		StreamRoleName:       "go-template-dev-dynamodb-stream-role-ab9505c",
		SESVerificationToken: "kd3ga0FxChDm+EhBekEaLcVOUT8WGR2Y5aIGEP1cyjc=",
	}, &awscdk.StackProps{
		Env: &awscdk.Environment{
			Account: jsii.String(account),
			Region:  jsii.String("us-west-2"),
		},
	})

	NewGoTemplateStack(app, "go-template-prod", EnvConfig{
		Env:                  "prod",
		Account:              account,
		Region:               "us-west-2",
		DomainName:           "template.quinnweber.com",
		ZoneID:               "Z07290422T6PHN4TKVNW0",
		TaskExecRoleName:     "go-template-prod-task-exec-role-1e22472",
		StreamRoleName:       "go-template-prod-dynamodb-stream-role-cdb3149",
		SESVerificationToken: "6ik66Ah6X7aAWRI4T5ILMtJ0KW4vbpHVpDufXGlpw/Y=",
	}, &awscdk.StackProps{
		Env: &awscdk.Environment{
			Account: jsii.String(account),
			Region:  jsii.String("us-west-2"),
		},
	})

	app.Synth(nil)
}
