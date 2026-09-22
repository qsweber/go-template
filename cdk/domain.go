package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2/awsapigateway"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsroute53"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

// sharedWildcardCertArn is the existing *.quinnweber.com ACM certificate in us-east-1,
// already issued and reused by both environments' edge-optimized API Gateway custom domains.
const sharedWildcardCertArn = "arn:aws:acm:us-east-1:120356305272:certificate/10f59a3f-a08e-4b8d-8a4c-f0a5fcb61e83"

func createCustomDomainResources(scope constructs.Construct, cfg EnvConfig, gateway awsapigateway.CfnRestApi, stageName string) {
	domainName := awsapigateway.NewCfnDomainName(scope, jsii.String("ApiDomainName"), &awsapigateway.CfnDomainNameProps{
		DomainName:     jsii.String(cfg.ApiDomainName),
		CertificateArn: jsii.String(sharedWildcardCertArn),
		EndpointConfiguration: &awsapigateway.CfnDomainName_EndpointConfigurationProperty{
			Types: jsii.Strings("EDGE"),
		},
	})

	basePathMapping := awsapigateway.NewCfnBasePathMapping(scope, jsii.String("ApiBasePathMapping"), &awsapigateway.CfnBasePathMappingProps{
		DomainName: domainName.Ref(),
		RestApiId:  gateway.AttrRestApiId(),
		Stage:      jsii.String(stageName),
	})
	basePathMapping.AddDependency(domainName)

	awsroute53.NewCfnRecordSet(scope, jsii.String("ApiDomainAliasRecord"), &awsroute53.CfnRecordSetProps{
		Name:         jsii.String(cfg.ApiDomainName),
		Type:         jsii.String("A"),
		HostedZoneId: jsii.String(cfg.ZoneID),
		AliasTarget: &awsroute53.CfnRecordSet_AliasTargetProperty{
			DnsName:      domainName.AttrDistributionDomainName(),
			HostedZoneId: domainName.AttrDistributionHostedZoneId(),
		},
	})
}
