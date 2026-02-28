package route53

import (
	"home-fern/internal/core"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	aws53 "github.com/aws/aws-sdk-go-v2/service/route53"
	awstypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
)

/*
/{id}/hostedzone
/{id}/change/{id}
/{id}/record/{name}
*/

type Service struct {
	dataStore  *dataStore
	soaDefault string
	nsRecords  []string
}

func NewService(dns *core.DnsDefaults, dataPath string) *Service {

	databasePath := dataPath + "/route53"
	dataStore := newDataStore(databasePath)

	result := Service{
		dataStore:  dataStore,
		soaDefault: dns.Soa,
		nsRecords:  dns.NameServers,
	}

	return &result
}

func (s *Service) Close() {

	if s.dataStore != nil {
		s.dataStore.Close()
	}
}

func (s *Service) ChangeResourceRecordSets(
	request *ChangeResourceRecordSetsRequest) (*aws53.ChangeResourceRecordSetsOutput, core.ErrorCode) {

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

func (s *Service) ChangeTagsForResource(
	request *aws53.ChangeTagsForResourceInput) (*aws53.ChangeTagsForResourceOutput, core.ErrorCode) {

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
			NameServers: s.nsRecords,
		},
	}

	if zone.HostedZoneConfig != nil {
		hz.Config.Comment = aws.ToString(zone.HostedZoneConfig.Comment)
		hz.Config.PrivateZone = zone.HostedZoneConfig.PrivateZone
	}

	if !strings.HasSuffix(hz.Name, ".") {
		hz.Name = hz.Name + "."
	}

	nsResourceRecords := make([]awstypes.ResourceRecord, 0, len(s.nsRecords))
	for _, ns := range s.nsRecords {
		nsResourceRecords = append(nsResourceRecords, awstypes.ResourceRecord{Value: aws.String(ns)})
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
			ResourceRecords: nsResourceRecords,
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

	hz, herr := s.dataStore.getHostedZone(aws.ToString(zone.Id))
	if herr != core.ErrNone {

		return nil, herr
	}

	count, cerr := s.dataStore.getRecordCount(hz.Id)
	if cerr != core.ErrNone {
		return nil, cerr
	}

	if count > 2 {

		return nil, ErrHostedZoneNotEmpty

	} else if count > 0 {

		recs, rerr := s.dataStore.getResourceRecordSets(hz.Id)
		if rerr != core.ErrNone {
			return nil, rerr
		}

		for _, rec := range recs {

			if rec.Type != awstypes.RRTypeNs && rec.Type != awstypes.RRTypeSoa {

				return nil, ErrHostedZoneNotEmpty
			}
		}
	}

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

	awshz, cerr := s.populateRecordCount(hz)
	if cerr != core.ErrNone {
		return nil, cerr
	}

	result := &aws53.GetHostedZoneOutput{
		HostedZone:    awshz,
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
	request *aws53.ListHostedZonesInput) (*aws53.ListHostedZonesOutput, core.ErrorCode) {

	zones, err := s.dataStore.findHostedZones(nil)
	if err != core.ErrNone {
		return nil, err
	}

	slices.SortFunc(zones, func(a, b HostedZoneData) int {
		return strings.Compare(a.Id, b.Id)
	})

	startIndex := s.findHostedZoneByMarker(zones, request.Marker)
	paginatedZones, nextZone := paginate(zones[startIndex:], request.MaxItems)

	awsZones, err := s.populateRecordCounts(paginatedZones)
	if err != core.ErrNone {
		return nil, err
	}

	var nextMarker *string
	if nextZone != nil {
		nextMarker = &nextZone.Id
	}

	return &aws53.ListHostedZonesOutput{
		HostedZones: awsZones,
		IsTruncated: nextZone != nil,
		Marker:      request.Marker,
		MaxItems:    request.MaxItems,
		NextMarker:  nextMarker,
	}, core.ErrNone
}

func (s *Service) ListHostedZonesByName(
	request *aws53.ListHostedZonesByNameInput) (*aws53.ListHostedZonesByNameOutput, core.ErrorCode) {

	if aws.ToString(request.HostedZoneId) != "" && aws.ToString(request.DNSName) == "" {
		return nil, ErrInvalidInput
	}

	zones, err := s.dataStore.findHostedZones(nil)
	if err != core.ErrNone {
		return nil, err
	}

	slices.SortFunc(zones, func(a, b HostedZoneData) int {
		n := strings.Compare(a.Name, b.Name)
		if n == 0 {
			return strings.Compare(a.Id, b.Id)
		}
		return n
	})

	startIndex := s.findHostedZoneByName(zones, request.DNSName, request.HostedZoneId)
	paginatedZones, nextZone := paginate(zones[startIndex:], request.MaxItems)

	awsZones, err := s.populateRecordCounts(paginatedZones)
	if err != core.ErrNone {
		return nil, err
	}

	var nextDNSName, nextHostedZoneId *string
	if nextZone != nil {
		nextDNSName = &nextZone.Name
		nextHostedZoneId = &nextZone.Id
	}

	return &aws53.ListHostedZonesByNameOutput{
		HostedZones:      awsZones,
		IsTruncated:      nextZone != nil,
		DNSName:          request.DNSName,
		HostedZoneId:     request.HostedZoneId,
		MaxItems:         request.MaxItems,
		NextDNSName:      nextDNSName,
		NextHostedZoneId: nextHostedZoneId,
	}, core.ErrNone
}

func (s *Service) ListResourceRecordSets(
	request *aws53.ListResourceRecordSetsInput) (*ListRecordSetsOutput, core.ErrorCode) {

	hz, err := s.dataStore.getHostedZone(aws.ToString(request.HostedZoneId))
	if err != core.ErrNone {
		return nil, err
	}

	records, err := s.dataStore.getResourceRecordSets(hz.Id)
	if err != core.ErrNone {
		return nil, err
	}

	slices.SortFunc(records, func(a, b ResourceRecordSetData) int {
		n := strings.Compare(a.Name, b.Name)
		if n == 0 {
			return strings.Compare(string(a.Type), string(b.Type))
		}
		return n
	})

	startIndex := s.findRecordSetIndex(records, request.StartRecordName, request.StartRecordType)
	paginatedRecords, nextRecord := paginate(records[startIndex:], request.MaxItems)

	result := ListRecordSetsOutput{
		Records: paginatedRecords,
	}

	if nextRecord != nil {
		result.NextRecord = nextRecord.Name
		result.NexType = nextRecord.Type
	}

	return &result, core.ErrNone
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

func (s *Service) ExportHostedZones() ([]HostedZoneExport, core.ErrorCode) {

	zones, err := s.dataStore.findHostedZones(nil)
	if err != core.ErrNone {
		return nil, err
	}

	var result []HostedZoneExport

	for _, zone := range zones {

		records, err := s.dataStore.getResourceRecordSets(zone.Id)
		if err != core.ErrNone {
			return nil, err
		}

		result = append(result, HostedZoneExport{
			HostedZone: zone,
			RecordSets: records,
		})
	}

	return result, core.ErrNone
}

func (s *Service) ImportHostedZones(
	zones []HostedZoneExport, overwrite bool) ([]string, core.ErrorCode) {

	var failures []string

	ci := ChangeInfoData{
		Id:          ChangeInfoPrefix + core.GenerateRandomString(14),
		Status:      awstypes.ChangeStatusInsync,
		SubmittedAt: time.Now().UTC().Format(time.RFC3339),
		Comment:     "Import is complete.",
	}

	for _, zone := range zones {

		var rsetChanges []ChangeData
		for _, record := range zone.RecordSets {

			// we need to create a copy of the record to avoid pointer issues
			// in the loop
			r := record
			rsetChanges = append(rsetChanges, ChangeData{
				Action:            awstypes.ChangeActionUpsert,
				ResourceRecordSet: &r,
			})
		}

		if overwrite {
			// if overwrite is true, we update the hosted zone
			// and upsert the records
			err := s.dataStore.updateHostedZone(&zone.HostedZone)
			if err != core.ErrNone {
				failures = append(failures, zone.HostedZone.Name)
				continue
			}

			err = s.dataStore.putRecordSets(&zone.HostedZone, rsetChanges, &ci)
			if err != core.ErrNone {
				failures = append(failures, zone.HostedZone.Name)
				continue
			}

		} else {
			// if overwrite is false, we try to create the hosted zone
			// if it fails, we add it to failures
			err := s.dataStore.putHostedZone(&zone.HostedZone, rsetChanges, &ci)
			if err != core.ErrNone {
				failures = append(failures, zone.HostedZone.Name)
				continue
			}
		}
	}

	return failures, core.ErrNone
}

func (s *Service) populateRecordCounts(zones []HostedZoneData) ([]awstypes.HostedZone, core.ErrorCode) {

	awsZones := make([]awstypes.HostedZone, 0, len(zones))
	for _, hz := range zones {
		awshz, cerr := s.populateRecordCount(&hz)
		if cerr != core.ErrNone {
			return nil, cerr
		}
		awsZones = append(awsZones, *awshz)
	}

	return awsZones, core.ErrNone
}

func (s *Service) populateRecordCount(hz *HostedZoneData) (*awstypes.HostedZone, core.ErrorCode) {

	count, cerr := s.dataStore.getRecordCount(hz.Id)
	if cerr != core.ErrNone {
		return nil, cerr
	}

	return hz.toHostedZone(count), core.ErrNone
}

func (s *Service) findHostedZoneByMarker(zones []HostedZoneData, marker *string) int {
	if marker == nil {
		return 0
	}

	m := aws.ToString(marker)
	if !strings.HasPrefix(m, HostedZonePrefix) {
		m = HostedZonePrefix + m
	}

	for i, zone := range zones {
		if zone.Id >= m {
			return i
		}
	}
	return len(zones)
}

func (s *Service) findHostedZoneByName(
	zones []HostedZoneData, dnsName *string, hostedZoneId *string) int {
	if dnsName == nil {
		return 0
	}

	name := strings.ToLower(aws.ToString(dnsName))
	if !strings.HasSuffix(name, ".") {
		name += "."
	}

	id := aws.ToString(hostedZoneId)
	if id != "" && !strings.HasPrefix(id, HostedZonePrefix) {
		id = HostedZonePrefix + id
	}

	for i, zone := range zones {
		if zone.Name > name {
			return i
		}
		if zone.Name == name {
			if id == "" || zone.Id >= id {
				return i
			}
		}
	}
	return len(zones)
}

func (s *Service) findRecordSetIndex(
	records []ResourceRecordSetData, startName *string, startType awstypes.RRType) int {
	if startName == nil || *startName == "" {
		return 0
	}

	name := strings.ToLower(aws.ToString(startName))
	if !strings.HasSuffix(name, ".") {
		name += "."
	}

	for i, rr := range records {
		if rr.Name > name {
			return i
		}
		if rr.Name == name {
			if startType == "" || string(rr.Type) >= string(startType) {
				return i
			}
		}
	}
	return len(records)
}

func paginate[T any](items []T, maxItems *int32) ([]T, *T) {
	limit := 100
	if maxItems != nil {
		limit = int(*maxItems)
	}

	if len(items) > limit {
		return items[:limit], &items[limit]
	}

	return items, nil
}
