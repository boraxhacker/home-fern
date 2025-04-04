package route53

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	aws53 "github.com/aws/aws-sdk-go-v2/service/route53"
	awstypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"home-fern/internal/core"
	"time"
)

/*
/{id}/hostedzone
/{id}/change/{id}
/{id}/record/{name}
*/

type Service struct {
}

func (s *Service) CreateHostedZone(zone *aws53.CreateHostedZoneInput) (*aws53.CreateHostedZoneOutput, error) {

	id := core.GenerateRandomString(14)

	data := HostedZoneData{
		CallerReference: aws.ToString(zone.CallerReference),
		Id:              id,
		Name:            aws.ToString(zone.Name),
	}

	if zone.HostedZoneConfig != nil {
		data.Config = struct {
			Comment     string
			PrivateZone bool
		}{
			Comment:     aws.ToString(zone.HostedZoneConfig.Comment),
			PrivateZone: zone.HostedZoneConfig.PrivateZone,
		}
	}

	submittedAt := time.Now().UTC()

	change := ChangeInfoData{
		Id:          core.GenerateRandomString(14),
		Status:      awstypes.ChangeStatusInsync,
		SubmittedAt: submittedAt.Format(time.RFC3339),
	}

	// save /{id}/hostedzone

	result := aws53.CreateHostedZoneOutput{
		HostedZone: &awstypes.HostedZone{
			CallerReference:        zone.CallerReference,
			Id:                     aws.String(id),
			Name:                   zone.Name,
			Config:                 zone.HostedZoneConfig,
			LinkedService:          nil,
			ResourceRecordSetCount: aws.Int64(2),
		},
		ChangeInfo: &awstypes.ChangeInfo{
			Id:          aws.String(change.Id),
			Status:      change.Status,
			SubmittedAt: &submittedAt,
		},
		DelegationSet: &awstypes.DelegationSet{
			NameServers: []string{"ns-1.example.com", "ns-2.example.com", "ns-3.example.com", "ns-4.example.com"},
		},
	}

	return &result, nil
}

func (s *Service) ListHostedZones(zone *aws53.ListHostedZonesInput) (*aws53.ListHostedZonesOutput, error) {

	return nil, nil
}
