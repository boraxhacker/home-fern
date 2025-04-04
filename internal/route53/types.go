package route53

import awstypes "github.com/aws/aws-sdk-go-v2/service/route53/types"

type HostedZoneData struct {
	CallerReference string
	Config          struct {
		Comment     string
		PrivateZone bool
	}
	Id   string
	Name string
}

type ChangeInfoData struct {
	Comment     string
	Id          string
	Status      awstypes.ChangeStatus
	SubmittedAt string
}
