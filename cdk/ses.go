package main

import (
	"fmt"
	"strconv"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsroute53"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsses"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type SESResources struct {
	EmailIdentity awsses.CfnEmailIdentity
}

// createSESResources uses the unified AWS::SES::EmailIdentity resource, CloudFormation's only
// native resource type for SES domain verification. Its Easy DKIM (3 CNAME tokens) supersedes
// the older TXT-based verification method; the legacy "_amazonses.<domain>" TXT record is kept
// in parallel for continuity with what's already published in DNS.
func createSESResources(scope constructs.Construct, projectStackName string, cfg EnvConfig) *SESResources {
	emailIdentity := awsses.NewCfnEmailIdentity(scope, jsii.String("SESEmailIdentity"), &awsses.CfnEmailIdentityProps{
		EmailIdentity: jsii.String(cfg.DomainName),
	})
	emailIdentity.ApplyRemovalPolicy(awscdk.RemovalPolicy_RETAIN, nil)

	verificationRecord := awsroute53.NewCfnRecordSet(scope, jsii.String("SESDomainVerification"), &awsroute53.CfnRecordSetProps{
		Name:            jsii.String(fmt.Sprintf("_amazonses.%s", cfg.DomainName)),
		Type:            jsii.String("TXT"),
		HostedZoneId:    jsii.String(cfg.ZoneID),
		Ttl:             jsii.String("300"),
		ResourceRecords: &[]*string{jsii.String(fmt.Sprintf("\"%s\"", cfg.SESVerificationToken))},
	})
	verificationRecord.ApplyRemovalPolicy(awscdk.RemovalPolicy_RETAIN, nil)

	dkimTokenNames := []*string{
		emailIdentity.AttrDkimDnsTokenName1(),
		emailIdentity.AttrDkimDnsTokenName2(),
		emailIdentity.AttrDkimDnsTokenName3(),
	}
	dkimTokenValues := []*string{
		emailIdentity.AttrDkimDnsTokenValue1(),
		emailIdentity.AttrDkimDnsTokenValue2(),
		emailIdentity.AttrDkimDnsTokenValue3(),
	}

	for i := 0; i < 3; i++ {
		record := awsroute53.NewCfnRecordSet(scope, jsii.String("SESDkim"+strconv.Itoa(i)), &awsroute53.CfnRecordSetProps{
			Name:            dkimTokenNames[i],
			Type:            jsii.String("CNAME"),
			HostedZoneId:    jsii.String(cfg.ZoneID),
			Ttl:             jsii.String("300"),
			ResourceRecords: &[]*string{dkimTokenValues[i]},
		})
		record.ApplyRemovalPolicy(awscdk.RemovalPolicy_RETAIN, nil)
		record.AddDependency(emailIdentity)
	}

	return &SESResources{
		EmailIdentity: emailIdentity,
	}
}
