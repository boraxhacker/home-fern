package route53

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	awstypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"home-fern/internal/core"
	"time"
)

const (
	NameServers string = "ns-1.example.com,ns-2.example.com,ns-3.example.com,ns-4.example.com"
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
	Tags            []core.ResourceTag
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
	AliasTarget             *awstypes.AliasTarget
	CidrRoutingConfig       *awstypes.CidrRoutingConfig
	Failover                awstypes.ResourceRecordSetFailover `xml:",omitempty"`
	GeoLocation             *awstypes.GeoLocation
	GeoProximityLocation    *awstypes.GeoProximityLocation
	HealthCheckId           *string
	MultiValueAnswer        *bool
	Region                  awstypes.ResourceRecordSetRegion `xml:",omitempty"`
	ResourceRecords         []awstypes.ResourceRecord        `xml:"ResourceRecords>ResourceRecord"`
	SetIdentifier           *string
	TTL                     *int64
	TrafficPolicyInstanceId *string
	Weight                  *int64
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
