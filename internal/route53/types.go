package route53

import (
	"home-fern/internal/core"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awstypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
)

type ChangeInfoData struct {
	Comment     string
	Id          string
	Status      awstypes.ChangeStatus
	SubmittedAt string
}

func (ci *ChangeInfoData) toChangeInfo() *awstypes.ChangeInfo {

	submitted, err := time.Parse(time.RFC3339, ci.SubmittedAt)
	if err != nil {
		submitted = time.Now().UTC()
	}

	return &awstypes.ChangeInfo{
		Id:          aws.String(ci.Id),
		Status:      ci.Status,
		SubmittedAt: aws.Time(submitted),
		Comment:     core.StringOrNil(ci.Comment),
	}
}

type HostedZoneConfigData struct {
	Comment     string `json:"Comment,omitempty"`
	PrivateZone bool
}

type DelegationSetData struct {
	NameServers     []string
	CallerReference string `json:"CallerReference,omitempty"`
	Id              string `json:"Id,omitempty"`
}

func (ds *DelegationSetData) toDelegationSet() *awstypes.DelegationSet {

	return &awstypes.DelegationSet{
		NameServers:     ds.NameServers,
		CallerReference: core.StringOrNil(ds.CallerReference),
		Id:              core.StringOrNil(ds.Id),
	}
}

type HostedZoneReference struct {
	Id string
}

type HostedZoneData struct {
	CallerReference string
	Config          HostedZoneConfigData
	DelegationSet   DelegationSetData
	Id              string
	Name            string
	Tags            []core.ResourceTag `json:",omitempty"`
}

func (hz *HostedZoneData) toHostedZone(rrcount int) *awstypes.HostedZone {

	result := awstypes.HostedZone{
		CallerReference: aws.String(hz.CallerReference),
		Id:              aws.String(hz.Id),
		Name:            aws.String(hz.Name),
		Config: &awstypes.HostedZoneConfig{
			Comment:     core.StringOrNil(hz.Config.Comment),
			PrivateZone: hz.Config.PrivateZone,
		},
		LinkedService:          nil,
		ResourceRecordSetCount: aws.Int64(int64(rrcount)),
	}

	return &result
}

type ResourceRecordSetData struct {
	Name                    string
	Type                    awstypes.RRType
	AliasTarget             *awstypes.AliasTarget              `json:",omitempty"`
	CidrRoutingConfig       *awstypes.CidrRoutingConfig        `json:",omitempty"`
	Failover                awstypes.ResourceRecordSetFailover `xml:",omitempty" json:",omitempty"`
	GeoLocation             *awstypes.GeoLocation              `json:",omitempty"`
	GeoProximityLocation    *awstypes.GeoProximityLocation     `json:",omitempty"`
	HealthCheckId           *string                            `json:",omitempty"`
	MultiValueAnswer        *bool                              `json:",omitempty"`
	Region                  awstypes.ResourceRecordSetRegion   `xml:",omitempty" json:",omitempty"`
	ResourceRecords         []awstypes.ResourceRecord          `xml:"ResourceRecords>ResourceRecord"`
	SetIdentifier           *string                            `json:",omitempty"`
	TTL                     *int64                             `json:",omitempty"`
	TrafficPolicyInstanceId *string                            `json:",omitempty"`
	Weight                  *int64                             `json:",omitempty"`
}

type ListRecordSetsOutput struct {
	Records    []ResourceRecordSetData
	NextRecord string
	NexType    awstypes.RRType
}

type ChangeData struct {
	Action            awstypes.ChangeAction
	ResourceRecordSet *ResourceRecordSetData
}

type ChangeBatchWrapper struct {
	Changes []ChangeData `xml:"Changes>Change"`
	Comment *string
}

type ChangeResourceRecordSetsRequest struct {
	ChangeBatch  ChangeBatchWrapper
	HostedZoneId string
}

type HostedZoneExport struct {
	HostedZone HostedZoneData
	RecordSets []ResourceRecordSetData
}
