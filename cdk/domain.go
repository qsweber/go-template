package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2/awscertificatemanager"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type CustomDomainResources struct {
	Certificate awscertificatemanager.CfnCertificate
}

// createCustomDomainResources issues the regional (us-west-2) ACM certificate for the API's
// custom domain. CloudFormation creates the DNS validation record in the hosted zone and waits
// for the certificate to be issued.
//
// The regional custom domain itself is added in a follow-up deploy: API Gateway domain names are
// unique, and CloudFormation creates new resources before deleting removed ones, so the old
// edge-optimized domain has to be gone before the regional one can be created.
func createCustomDomainResources(scope constructs.Construct, cfg EnvConfig) *CustomDomainResources {
	certificate := awscertificatemanager.NewCfnCertificate(scope, jsii.String("ApiCertificate"), &awscertificatemanager.CfnCertificateProps{
		DomainName:       jsii.String(cfg.ApiDomainName),
		ValidationMethod: jsii.String("DNS"),
		DomainValidationOptions: []interface{}{
			&awscertificatemanager.CfnCertificate_DomainValidationOptionProperty{
				DomainName:   jsii.String(cfg.ApiDomainName),
				HostedZoneId: jsii.String(cfg.ZoneID),
			},
		},
	})

	return &CustomDomainResources{Certificate: certificate}
}
