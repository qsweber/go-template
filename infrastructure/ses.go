package main

import (
	"strconv"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/route53"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ses"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type SESResources struct {
	DomainIdentity *ses.DomainIdentity
	DomainDkim     *ses.DomainDkim
	Records        []*route53.Record
}

func createSESResources(ctx *pulumi.Context, projectStackName string, domainName string, zoneID string) (*SESResources, error) {
	domainIdentity, err := ses.NewDomainIdentity(ctx, projectStackName+"-ses-domain-identity", &ses.DomainIdentityArgs{
		Domain: pulumi.String(domainName),
	}, pulumi.Protect(true))
	if err != nil {
		return nil, err
	}

	verificationRecord, err := route53.NewRecord(ctx, projectStackName+"-ses-domain-verification", &route53.RecordArgs{
		Name:    pulumi.Sprintf("_amazonses.%s", domainName),
		Type:    pulumi.String("TXT"),
		ZoneId:  pulumi.String(zoneID),
		Ttl:     pulumi.Int(300),
		Records: pulumi.StringArray{domainIdentity.VerificationToken},
	}, pulumi.Protect(true))
	if err != nil {
		return nil, err
	}

	domainDkim, err := ses.NewDomainDkim(ctx, projectStackName+"-ses-domain-dkim", &ses.DomainDkimArgs{
		Domain: pulumi.String(domainName),
	}, pulumi.Protect(true))
	if err != nil {
		return nil, err
	}

	records := []*route53.Record{verificationRecord}
	for i := 0; i < 3; i++ {
		token := domainDkim.DkimTokens.Index(pulumi.Int(i))
		record, err := route53.NewRecord(ctx, projectStackName+"-ses-dkim-"+strconv.Itoa(i), &route53.RecordArgs{
			Name:    pulumi.Sprintf("%s._domainkey.%s", token, domainName),
			Type:    pulumi.String("CNAME"),
			ZoneId:  pulumi.String(zoneID),
			Ttl:     pulumi.Int(300),
			Records: pulumi.StringArray{pulumi.Sprintf("%s.dkim.amazonses.com", token)},
		}, pulumi.Protect(true))
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return &SESResources{
		DomainIdentity: domainIdentity,
		DomainDkim:     domainDkim,
		Records:        records,
	}, nil
}
