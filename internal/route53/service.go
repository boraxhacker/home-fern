package route53

import (
	"errors"
	"home-fern/internal/core"
	"home-fern/internal/datastore"
	"io"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	aws53 "github.com/aws/aws-sdk-go-v2/service/route53"
	awstypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
)

type Service struct {
	dataStore  *dataStore
	soaDefault string
	nsRecords  []string
}

func NewService(dns *core.DnsDefaults, ds *datastore.Datastore) *Service {
	dataStore := newDataStore(ds)

	result := Service{
		dataStore:  dataStore,
		soaDefault: dns.Soa,
		nsRecords:  dns.NameServers,
	}

	return &result
}

func (s *Service) ChangeResourceRecordSets(
	request *ChangeResourceRecordSetsRequest) (*aws53.ChangeResourceRecordSetsOutput, error) {

	hz, err := s.dataStore.getHostedZone(request.HostedZoneId)
	if err != nil {
		return nil, err
	}

	ci := ChangeInfoData{
		Id:          ChangeInfoPrefix + core.GenerateRandomString(14),
		Status:      awstypes.ChangeStatusInsync,
		SubmittedAt: time.Now().UTC().Format(time.RFC3339),
		Comment:     aws.ToString(request.ChangeBatch.Comment),
	}

	err = s.dataStore.putRecordSets(hz, request.ChangeBatch.Changes, &ci)
	if err != nil {
		return nil, err
	}

	result := aws53.ChangeResourceRecordSetsOutput{
		ChangeInfo: ci.toChangeInfo(),
	}

	return &result, nil
}

func (s *Service) ChangeTagsForResource(
	request *aws53.ChangeTagsForResourceInput) (*aws53.ChangeTagsForResourceOutput, error) {

	if request.ResourceType != awstypes.TagResourceTypeHostedzone {
		return nil, ErrInvalidInput
	}

	hz, err := s.dataStore.getHostedZone(aws.ToString(request.ResourceId))
	if err != nil {
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
	if err != nil {
		return nil, err
	}

	return &aws53.ChangeTagsForResourceOutput{}, nil
}

func (s *Service) CreateHostedZone(
	zone *aws53.CreateHostedZoneInput) (*aws53.CreateHostedZoneOutput, error) {

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
	if err != nil {
		return nil, err
	}

	result := aws53.CreateHostedZoneOutput{
		HostedZone:    hz.toHostedZone(2),
		ChangeInfo:    ci.toChangeInfo(),
		DelegationSet: hz.DelegationSet.toDelegationSet(),
	}

	return &result, nil
}

func (s *Service) DeleteHostedZone(
	zone *aws53.DeleteHostedZoneInput) (*aws53.DeleteHostedZoneOutput, error) {

	hz, herr := s.dataStore.getHostedZone(aws.ToString(zone.Id))
	if herr != nil {
		return nil, herr
	}

	count, cerr := s.dataStore.getRecordCount(hz.Id)
	if cerr != nil {
		return nil, cerr
	}

	if count > 2 {
		return nil, ErrHostedZoneNotEmpty
	} else if count > 0 {
		recs, rerr := s.dataStore.getResourceRecordSets(hz.Id)
		if rerr != nil {
			return nil, rerr
		}

		for _, rec := range recs {
			if rec.Type != awstypes.RRTypeNs && rec.Type != awstypes.RRTypeSoa {
				return nil, ErrHostedZoneNotEmpty
			}
		}
	}

	ci := ChangeInfoData{
		Id:          ChangeInfoPrefix + core.GenerateRandomString(14),
		Status:      awstypes.ChangeStatusInsync,
		SubmittedAt: time.Now().UTC().Format(time.RFC3339),
		Comment:     "Change is complete.",
	}

	err := s.dataStore.deleteHostedZone(aws.ToString(zone.Id), &ci)
	if err != nil {
		return nil, err
	}

	result := &aws53.DeleteHostedZoneOutput{
		ChangeInfo: ci.toChangeInfo(),
	}

	return result, nil
}

func (s *Service) GetHostedZoneCount() (*aws53.GetHostedZoneCountOutput, error) {

	result, err := s.dataStore.getHostedZoneCount()
	if err != nil {
		return nil, err
	}

	return &aws53.GetHostedZoneCountOutput{HostedZoneCount: aws.Int64(int64(result))}, nil
}

func (s *Service) GetHostedZone(
	zone *aws53.GetHostedZoneInput) (*aws53.GetHostedZoneOutput, error) {

	hz, err := s.dataStore.getHostedZone(aws.ToString(zone.Id))
	if err != nil {
		return nil, err
	}

	awshz, cerr := s.populateRecordCount(hz)
	if cerr != nil {
		return nil, cerr
	}

	result := &aws53.GetHostedZoneOutput{
		HostedZone:    awshz,
		DelegationSet: hz.DelegationSet.toDelegationSet(),
	}

	return result, nil
}

func (s *Service) GetChange(
	request *aws53.GetChangeInput) (*aws53.GetChangeOutput, error) {

	ci, err := s.dataStore.getChange(aws.ToString(request.Id))
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			ci = &ChangeInfoData{
				Id:          ChangeInfoPrefix + aws.ToString(request.Id),
				Comment:     "Expired - ChangeInfo expires after ~90 days",
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

	return result, nil
}

func (s *Service) ListHostedZones(
	request *aws53.ListHostedZonesInput) (*aws53.ListHostedZonesOutput, error) {

	zones, err := s.dataStore.findHostedZones(nil)
	if err != nil {
		return nil, err
	}

	slices.SortFunc(zones, func(a, b HostedZoneData) int {
		return strings.Compare(a.Id, b.Id)
	})

	startIndex := s.findHostedZoneByMarker(zones, request.Marker)
	paginatedZones, nextZone := paginate(zones[startIndex:], request.MaxItems)

	awsZones, err := s.populateRecordCounts(paginatedZones)
	if err != nil {
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
	}, nil
}

func (s *Service) ListHostedZonesByName(
	request *aws53.ListHostedZonesByNameInput) (*aws53.ListHostedZonesByNameOutput, error) {

	if aws.ToString(request.HostedZoneId) != "" && aws.ToString(request.DNSName) == "" {
		return nil, ErrInvalidInput
	}

	zones, err := s.dataStore.findHostedZones(nil)
	if err != nil {
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
	if err != nil {
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
	}, nil
}

func (s *Service) ListResourceRecordSets(
	request *aws53.ListResourceRecordSetsInput) (*ListRecordSetsOutput, error) {

	hz, err := s.dataStore.getHostedZone(aws.ToString(request.HostedZoneId))
	if err != nil {
		return nil, err
	}

	records, err := s.dataStore.getResourceRecordSets(hz.Id)
	if err != nil {
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

	return &result, nil
}

func (s *Service) ListTagsForResource(
	request *aws53.ListTagsForResourceInput) (*aws53.ListTagsForResourceOutput, error) {

	if request.ResourceType != awstypes.TagResourceTypeHostedzone {
		return nil, ErrInvalidInput
	}

	hz, err := s.dataStore.getHostedZone(aws.ToString(request.ResourceId))
	if err != nil {
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

	return &result, nil
}

func (s *Service) LogKeys(writer io.Writer) error {
	return s.dataStore.logKeys(writer)
}

func (s *Service) UpdateHostedZoneComment(
	request *aws53.UpdateHostedZoneCommentInput) (*aws53.UpdateHostedZoneCommentOutput, error) {

	hz, err := s.dataStore.getHostedZone(aws.ToString(request.Id))
	if err != nil {
		return nil, err
	}

	hz.Config.Comment = aws.ToString(request.Comment)

	err = s.dataStore.updateHostedZone(hz)
	if err != nil {
		return nil, err
	}

	count, err := s.dataStore.getRecordCount(hz.Id)
	if err != nil {
		return nil, err
	}

	result := aws53.UpdateHostedZoneCommentOutput{
		HostedZone: hz.toHostedZone(count),
	}

	return &result, nil
}

func (s *Service) ExportHostedZones() ([]HostedZoneExport, error) {

	zones, err := s.dataStore.findHostedZones(nil)
	if err != nil {
		return nil, err
	}

	var result []HostedZoneExport

	for _, zone := range zones {

		records, err := s.dataStore.getResourceRecordSets(zone.Id)
		if err != nil {
			return nil, err
		}

		result = append(result, HostedZoneExport{
			HostedZone: zone,
			RecordSets: records,
		})
	}

	return result, nil
}

func (s *Service) DeleteAllData() error {
	return s.dataStore.deleteAll()
}

func (s *Service) ImportHostedZones(
	zones []HostedZoneExport, overwrite bool) ([]string, error) {

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
			if err != nil {
				log.Println("Error importing zone:", err)
				failures = append(failures, zone.HostedZone.Name)
				continue
			}

			ci.Id = ChangeInfoPrefix + core.GenerateRandomString(14)
			err = s.dataStore.putRecordSets(&zone.HostedZone, rsetChanges, &ci)
			if err != nil {
				log.Println("Error importing records:", err)
				failures = append(failures, zone.HostedZone.Name)
				continue
			}

		} else {
			// if overwrite is false, we try to create the hosted zone
			// if it fails, we add it to failures
			ci.Id = ChangeInfoPrefix + core.GenerateRandomString(14)
			err := s.dataStore.putHostedZone(&zone.HostedZone, rsetChanges, &ci)
			if err != nil {
				log.Println("Error importing zone:", err)
				failures = append(failures, zone.HostedZone.Name)
				continue
			}
		}
	}

	return failures, nil
}

func (s *Service) populateRecordCounts(zones []HostedZoneData) ([]awstypes.HostedZone, error) {

	awsZones := make([]awstypes.HostedZone, 0, len(zones))
	for _, hz := range zones {
		awshz, cerr := s.populateRecordCount(&hz)
		if cerr != nil {
			return nil, cerr
		}
		awsZones = append(awsZones, *awshz)
	}

	return awsZones, nil
}

func (s *Service) populateRecordCount(hz *HostedZoneData) (*awstypes.HostedZone, error) {

	count, cerr := s.dataStore.getRecordCount(hz.Id)
	if cerr != nil {
		return nil, cerr
	}

	return hz.toHostedZone(count), nil
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
