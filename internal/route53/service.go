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
	dataStore *dataStore
}

func NewService(dataPath string) *Service {

	databasePath := dataPath + "/route53"
	dataStore := newDataStore(databasePath)

	result := Service{
		dataStore: dataStore,
	}

	return &result
}

func (s *Service) Close() {

	if s.dataStore != nil {
		s.dataStore.Close()
	}
}

func (s *Service) CreateHostedZone(zone *aws53.CreateHostedZoneInput) (*aws53.CreateHostedZoneOutput, core.ErrorCode) {

	submittedAt := time.Now().UTC()

	cd := ChangeInfoData{
		Id:          core.GenerateRandomString(14),
		Status:      awstypes.ChangeStatusInsync,
		SubmittedAt: submittedAt.Format(time.RFC3339),
	}

	hz := HostedZoneData{
		CallerReference: aws.ToString(zone.CallerReference),
		Id:              core.GenerateRandomString(14),
		Name:            aws.ToString(zone.Name),
		ChangeId:        cd.Id,
		Config: struct {
			Comment     string
			PrivateZone bool
		}{
			Comment:     "",
			PrivateZone: false,
		},
	}

	if zone.HostedZoneConfig != nil {
		hz.Config.Comment = aws.ToString(zone.HostedZoneConfig.Comment)
	}

	err := s.dataStore.putHostedZone(&hz, &cd)
	if err != core.ErrNone {

		return nil, err
	}

	result := aws53.CreateHostedZoneOutput{
		HostedZone: &awstypes.HostedZone{
			CallerReference:        zone.CallerReference,
			Id:                     aws.String(hz.Id),
			Name:                   zone.Name,
			Config:                 zone.HostedZoneConfig,
			LinkedService:          nil,
			ResourceRecordSetCount: aws.Int64(2),
		},
		ChangeInfo: &awstypes.ChangeInfo{
			Id:          aws.String(cd.Id),
			Status:      cd.Status,
			SubmittedAt: &submittedAt,
		},
		DelegationSet: &awstypes.DelegationSet{
			NameServers: []string{"ns-1.example.com", "ns-2.example.com", "ns-3.example.com", "ns-4.example.com"},
		},
	}

	return &result, core.ErrNone
}

func (s *Service) ListHostedZones(_ *aws53.ListHostedZonesInput) (*aws53.ListHostedZonesOutput, core.ErrorCode) {

	zones, err := s.dataStore.findHostedZones()
	if err != core.ErrNone {

		return nil, err
	}

	var result aws53.ListHostedZonesOutput

	for _, hz := range zones {
		result.HostedZones = append(result.HostedZones, awstypes.HostedZone{
			CallerReference: aws.String(hz.CallerReference),
			Id:              aws.String(hz.Id),
			Name:            aws.String(hz.Name),
			Config: &awstypes.HostedZoneConfig{
				Comment:     aws.String(hz.Config.Comment),
				PrivateZone: hz.Config.PrivateZone,
			},
			LinkedService: nil,
		})
	}

	result.IsTruncated = false

	return &result, core.ErrNone
}
