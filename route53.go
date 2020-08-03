package main

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/libdns/libdns"
)

func getZoneID(svc *route53.Route53, zoneName string) (string, error) {
	listResult, err := svc.ListHostedZonesByName(
		&route53.ListHostedZonesByNameInput{
			DNSName:  aws.String(zoneName),
			MaxItems: aws.String("1"),
		})
	if err != nil {
		return "", err
	}

	if len(listResult.HostedZones) == 0 || *listResult.HostedZones[0].Name != zoneName {
		return "", fmt.Errorf("no zone found for domain %s", zoneName)
	}

	return *listResult.HostedZones[0].Id, nil
}

func GetRecords(svc *route53.Route53, zoneName string) ([]libdns.Record, error) {
	zoneID, err := getZoneID(svc, zoneName)
	if err != nil {
		return nil, err
	}

	listInput := route53.ListResourceRecordSetsInput{
		HostedZoneId: aws.String(zoneID),
		MaxItems:     aws.String("100"),
	}

	var recordSets []*route53.ResourceRecordSet
	for {
		listResult, err := svc.ListResourceRecordSets(&listInput)
		if err != nil {
			return nil, err
		}

		recordSets = append(recordSets, listResult.ResourceRecordSets...)

		if !*listResult.IsTruncated {
			break
		}

		listInput.StartRecordName = listResult.NextRecordName
		listInput.StartRecordType = listResult.NextRecordType
		listInput.StartRecordIdentifier = listResult.NextRecordIdentifier
	}

	var records []libdns.Record
	for _, recordSet := range recordSets {
		for _, record := range recordSet.ResourceRecords {
			records = append(records,
				libdns.Record{
					Name:  *recordSet.Name,
					Value: *record.Value,
					Type:  *recordSet.Type,
					TTL:   time.Duration(*recordSet.TTL) * time.Second,
				})
		}
	}

	return records, nil
}

func SetRecords(svc *route53.Route53, zoneName string, records []libdns.Record) error {
	zoneID, err := getZoneID(svc, zoneName)
	if err != nil {
		return err
	}

	for _, record := range records {
		_, err := svc.ChangeResourceRecordSets(
			&route53.ChangeResourceRecordSetsInput{
				ChangeBatch: &route53.ChangeBatch{
					Changes: []*route53.Change{
						{
							Action: aws.String("UPSERT"),
							ResourceRecordSet: &route53.ResourceRecordSet{
								Name: aws.String(record.Name),
								ResourceRecords: []*route53.ResourceRecord{
									{
										Value: aws.String(record.Value),
									},
								},
								TTL:  aws.Int64(int64(record.TTL / time.Second)),
								Type: aws.String(record.Type),
							},
						},
					},
				},
				HostedZoneId: aws.String(zoneID),
			})

		if err != nil {
			return err
		}
	}

	return nil
}
