package route53

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	aws53 "github.com/aws/aws-sdk-go-v2/service/route53"
	awstypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"home-fern/internal/core"
	"io"
	"slices"
	"strings"
	"time"
)

/*
/{id}/hostedzone
/{id}/change/{id}
/{id}/record/{name}
*/

type Service struct {
	dataStore  *dataStore
	soaDefault string
	nsRecords  []awstypes.ResourceRecord
}

func NewService(dns *core.DnsDefaults, dataPath string) *Service {

	databasePath := dataPath + "/route53"
	dataStore := newDataStore(databasePath)

	nsRecords := make([]awstypes.ResourceRecord, 0)
	for _, ns := range dns.NameServers {
		nsRecords = append(nsRecords, awstypes.ResourceRecord{Value: aws.String(ns)})
	}

	result := Service{
		dataStore:  dataStore,
		soaDefault: dns.Soa,
		nsRecords:  nsRecords,
	}

	return &result
}

func (s *Service) Close() {

	if s.dataStore != nil {
		s.dataStore.Close()
	}
}

func (s *Service) ChangeResourceRecordSets(request *ChangeResourceRecordSetsRequest) (*aws53.ChangeResourceRecordSetsOutput, core.ErrorCode) {

	hz, err := s.dataStore.getHostedZone(request.HostedZoneId)
	if err != core.ErrNone {

		return nil, err
	}

	ci := ChangeInfoData{
		Id:          ChangeInfoPrefix + core.GenerateRandomString(14),
		Status:      awstypes.ChangeStatusInsync,
		SubmittedAt: time.Now().UTC().Format(time.RFC3339),
		Comment:     aws.ToString(request.ChangeBatch.Comment),
	}

	err = s.dataStore.putRecordSets(hz, request.ChangeBatch.Changes, &ci)
	if err != core.ErrNone {
		return nil, err
	}

	result := aws53.ChangeResourceRecordSetsOutput{
		ChangeInfo: ci.toChangeInfo(),
	}

	return &result, core.ErrNone
}

func (s *Service) ChangeTagsForResource(request *aws53.ChangeTagsForResourceInput) (*aws53.ChangeTagsForResourceOutput, core.ErrorCode) {

	if request.ResourceType != awstypes.TagResourceTypeHostedzone {

		return nil, ErrInvalidInput
	}

	hz, err := s.dataStore.getHostedZone(aws.ToString(request.ResourceId))
	if err != core.ErrNone {

		return nil, err
	}

	for _, tag := range request.RemoveTagKeys {

		for i, mytag := range hz.Tags {
			if mytag.Key == tag {
				hz.Tags = slices.Delete(hz.Tags, i, i+1)
				break
			}
		}
	}

	for _, tag := range request.AddTags {

		newtag := core.ResourceTag{
			Key:   aws.ToString(tag.Key),
			Value: aws.ToString(tag.Value),
		}

		hz.Tags = append(hz.Tags, newtag)
	}

	err = s.dataStore.updateHostedZone(hz)
	if err != core.ErrNone {
		return nil, err
	}

	return &aws53.ChangeTagsForResourceOutput{}, core.ErrNone
}

func (s *Service) CreateHostedZone(
	zone *aws53.CreateHostedZoneInput) (*aws53.CreateHostedZoneOutput, core.ErrorCode) {

	ci := ChangeInfoData{
		Id:          ChangeInfoPrefix + core.GenerateRandomString(14),
		Status:      awstypes.ChangeStatusInsync,
		SubmittedAt: time.Now().UTC().Format(time.RFC3339),
		Comment:     "Change is complete.",
	}

	hz := HostedZoneData{
		CallerReference: aws.ToString(zone.CallerReference),
		Id:              HostedZonePrefix + core.GenerateRandomString(14),
		Name:            strings.ToLower(aws.ToString(zone.Name)),
		DelegationSet: DelegationSetData{
			Id:          aws.ToString(zone.DelegationSetId),
			NameServers: strings.Split(NameServers, ","),
		},
	}

	if zone.HostedZoneConfig != nil {
		hz.Config.Comment = aws.ToString(zone.HostedZoneConfig.Comment)
		hz.Config.PrivateZone = zone.HostedZoneConfig.PrivateZone
	}

	if !strings.HasSuffix(hz.Name, ".") {
		hz.Name = hz.Name + "."
	}

	changes := []ChangeData{{
		Action: awstypes.ChangeActionCreate,
		ResourceRecordSet: &ResourceRecordSetData{
			Name:            hz.Name,
			Type:            awstypes.RRTypeSoa,
			TTL:             aws.Int64(900),
			ResourceRecords: []awstypes.ResourceRecord{{Value: aws.String(s.soaDefault)}},
		},
	}, {
		Action: awstypes.ChangeActionCreate,
		ResourceRecordSet: &ResourceRecordSetData{
			Name:            hz.Name,
			Type:            awstypes.RRTypeNs,
			ResourceRecords: s.nsRecords,
		},
	}}

	err := s.dataStore.putHostedZone(&hz, changes, &ci)
	if err != core.ErrNone {

		return nil, err
	}

	result := aws53.CreateHostedZoneOutput{
		HostedZone:    hz.toHostedZone(2),
		ChangeInfo:    ci.toChangeInfo(),
		DelegationSet: hz.DelegationSet.toDelegationSet(),
	}

	return &result, core.ErrNone
}

func (s *Service) DeleteHostedZone(
	zone *aws53.DeleteHostedZoneInput) (*aws53.DeleteHostedZoneOutput, core.ErrorCode) {

	ci := ChangeInfoData{
		Id:          core.GenerateRandomString(14),
		Status:      awstypes.ChangeStatusInsync,
		SubmittedAt: time.Now().UTC().Format(time.RFC3339),
		Comment:     "Change is complete.",
	}

	err := s.dataStore.deleteHostedZone(aws.ToString(zone.Id), &ci)
	if err != core.ErrNone {

		return nil, err
	}

	result := &aws53.DeleteHostedZoneOutput{
		ChangeInfo: ci.toChangeInfo(),
	}

	return result, core.ErrNone
}

func (s *Service) GetHostedZoneCount() (*aws53.GetHostedZoneCountOutput, core.ErrorCode) {

	result, err := s.dataStore.getHostedZoneCount()
	if err != core.ErrNone {

		return nil, err
	}

	return &aws53.GetHostedZoneCountOutput{HostedZoneCount: aws.Int64(int64(result))}, core.ErrNone
}

func (s *Service) GetHostedZone(
	zone *aws53.GetHostedZoneInput) (*aws53.GetHostedZoneOutput, core.ErrorCode) {

	hz, err := s.dataStore.getHostedZone(aws.ToString(zone.Id))
	if err != core.ErrNone {

		return nil, err
	}

	count, cerr := s.dataStore.getRecordCount(hz.Id)
	if cerr != core.ErrNone {
		return nil, cerr
	}

	result := &aws53.GetHostedZoneOutput{
		HostedZone:    hz.toHostedZone(count),
		DelegationSet: hz.DelegationSet.toDelegationSet(),
	}

	return result, core.ErrNone
}

func (s *Service) GetChange(
	request *aws53.GetChangeInput) (*aws53.GetChangeOutput, core.ErrorCode) {

	ci, err := s.dataStore.getChange(aws.ToString(request.Id))
	if err != core.ErrNone {

		if err == core.ErrNotFound {

			ci = &ChangeInfoData{
				Comment:     "Expired - ChangeInfo expires after ~90 days",
				Id:          ChangeInfoPrefix + aws.ToString(request.Id),
				Status:      awstypes.ChangeStatusInsync,
				SubmittedAt: time.Now().UTC().Add(-90 * 24 * time.Hour).Format(time.RFC3339),
			}

		} else {

			return nil, err
		}
	}

	result := &aws53.GetChangeOutput{
		ChangeInfo: ci.toChangeInfo(),
	}

	return result, core.ErrNone
}

func (s *Service) ListHostedZones(
	_ *aws53.ListHostedZonesInput) (*aws53.ListHostedZonesOutput, core.ErrorCode) {

	zones, err := s.dataStore.findHostedZones(".*")
	if err != core.ErrNone {

		return nil, err
	}

	result := aws53.ListHostedZonesOutput{
		HostedZones: make([]awstypes.HostedZone, 0),
		IsTruncated: false,
	}

	for _, hz := range zones {
		count, cerr := s.dataStore.getRecordCount(hz.Id)
		if cerr != core.ErrNone {
			return nil, cerr
		}

		result.HostedZones = append(result.HostedZones, *hz.toHostedZone(count))
	}

	return &result, core.ErrNone
}

func (s *Service) ListResourceRecordSets(
	request *aws53.ListResourceRecordSetsInput) (*ListRecordSetsOutput, core.ErrorCode) {

	hz, err := s.dataStore.getHostedZone(aws.ToString(request.HostedZoneId))
	if err != core.ErrNone {

		return nil, err
	}

	options := listOptions{
		hostedZone:  hz,
		startRecord: aws.ToString(request.StartRecordName),
		startType:   request.StartRecordType,
		count:       int(aws.ToInt32(request.MaxItems)),
	}

	result, err := s.dataStore.getResourceRecordSets(&options)

	return result, core.ErrNone
}

func (s *Service) ListTagsForResource(
	request *aws53.ListTagsForResourceInput) (*aws53.ListTagsForResourceOutput, core.ErrorCode) {

	if request.ResourceType != awstypes.TagResourceTypeHostedzone {

		return nil, ErrInvalidInput
	}

	hz, err := s.dataStore.getHostedZone(aws.ToString(request.ResourceId))
	if err != core.ErrNone {

		return nil, err
	}

	result := aws53.ListTagsForResourceOutput{
		ResourceTagSet: &awstypes.ResourceTagSet{
			ResourceId:   request.ResourceId,
			ResourceType: request.ResourceType,
			Tags:         make([]awstypes.Tag, 0),
		},
	}

	for _, tag := range hz.Tags {

		awstag := awstypes.Tag{
			Key:   aws.String(tag.Key),
			Value: aws.String(tag.Value),
		}

		result.ResourceTagSet.Tags = append(result.ResourceTagSet.Tags, awstag)
	}

	return &result, core.ErrNone
}

func (s *Service) LogKeys(writer io.Writer) error {

	return core.LogKeys(s.dataStore.db, writer)
}

func (s *Service) UpdateHostedZoneComment(
	request *aws53.UpdateHostedZoneCommentInput) (*aws53.UpdateHostedZoneCommentOutput, core.ErrorCode) {

	hz, err := s.dataStore.getHostedZone(aws.ToString(request.Id))
	if err != core.ErrNone {

		return nil, err
	}

	hz.Config.Comment = aws.ToString(request.Comment)

	err = s.dataStore.updateHostedZone(hz)
	if err != core.ErrNone {
		return nil, err
	}

	count, err := s.dataStore.getRecordCount(hz.Id)
	if err != core.ErrNone {
		return nil, err
	}

	result := aws53.UpdateHostedZoneCommentOutput{
		HostedZone: hz.toHostedZone(count),
	}

	return &result, core.ErrNone
}
