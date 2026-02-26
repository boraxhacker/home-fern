package route53

import (
	"encoding/xml"
	"home-fern/internal/awslib"
	"home-fern/internal/core"
	"log"
	"net/http"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	aws53 "github.com/aws/aws-sdk-go-v2/service/route53"
	awstypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/gorilla/mux"
)

type Api struct {
	credentials *awslib.CredentialsProvider
	service     *Service
}

func NewRoute53Api(service *Service, credentials *awslib.CredentialsProvider) *Api {

	return &Api{
		credentials: credentials,
		service:     service,
	}
}

func (api *Api) ChangeResourceRecordSets(w http.ResponseWriter, r *http.Request) {

	log.Println("Amazon-Target: Route53.ChangeResourceRecordSets")

	var request ChangeResourceRecordSetsRequest

	if err := xml.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Println("Error:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	vars := mux.Vars(r)
	request.HostedZoneId = vars["id"]

	response, err := api.service.ChangeResourceRecordSets(&request)
	if err != core.ErrNone {
		awslib.WriteErrorResponseXML(w, translateToApiError(err), r.URL, api.credentials.Region)
		return
	}

	awslib.WriteSuccessResponseXML(w, struct {
		XMLName xml.Name `xml:"ChangeResourceRecordSetsResponse"`
		*aws53.ChangeResourceRecordSetsOutput
	}{
		ChangeResourceRecordSetsOutput: response,
	})
}

func (api *Api) ChangeTagsForResource(w http.ResponseWriter, r *http.Request) {

	log.Println("Amazon-Target: Route53.ChangeTagsForResource")

	type ChangeTagsWrapper struct {
		XMLName       xml.Name       `xml:"ChangeTagsForResourceRequest"`
		AddTags       []awstypes.Tag `xml:"AddTags>Tag"`
		RemoveTagKeys []string       `xml:"RemoveTagKeys>Key"`
	}

	var changetags ChangeTagsWrapper

	if err := xml.NewDecoder(r.Body).Decode(&changetags); err != nil {
		log.Println("Error:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Parsing the path parameters
	vars := mux.Vars(r)

	request := aws53.ChangeTagsForResourceInput{
		ResourceId:    aws.String(vars["resourceId"]),
		ResourceType:  awstypes.TagResourceType(vars["resourceType"]),
		AddTags:       changetags.AddTags,
		RemoveTagKeys: changetags.RemoveTagKeys,
	}

	response, err := api.service.ChangeTagsForResource(&request)
	if err != core.ErrNone {
		awslib.WriteErrorResponseXML(w, translateToApiError(err), r.URL, api.credentials.Region)
		return
	}

	awslib.WriteSuccessResponseXML(w, struct {
		XMLName xml.Name `xml:"ChangeTagsForResourceResponse"`
		*aws53.ChangeTagsForResourceOutput
	}{
		ChangeTagsForResourceOutput: response,
	})
}

func (api *Api) CreateHostedZone(w http.ResponseWriter, r *http.Request) {

	log.Println("Amazon-Target: Route53.CreateHostedZone")

	var request aws53.CreateHostedZoneInput
	if err := xml.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Println("Error:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := api.service.CreateHostedZone(&request)
	if err != core.ErrNone {
		awslib.WriteErrorResponseXML(w, translateToApiError(err), r.URL, api.credentials.Region)
		return
	}

	awslib.WriteSuccessResponseXML(w, struct {
		XMLName xml.Name `xml:"CreateHostedZoneResponse"`
		*aws53.CreateHostedZoneOutput
	}{
		CreateHostedZoneOutput: response,
	})
}

func (api *Api) DeleteHostedZone(w http.ResponseWriter, r *http.Request) {

	// DELETE /2013-04-01/hostedzone/Id

	log.Println("Amazon-Target: Route53.DeleteHostedZone")

	var request aws53.DeleteHostedZoneInput

	// Parsing the path parameters
	vars := mux.Vars(r)
	request.Id = aws.String(vars["id"])

	response, err := api.service.DeleteHostedZone(&request)
	if err != core.ErrNone {
		awslib.WriteErrorResponseXML(w, translateToApiError(err), r.URL, api.credentials.Region)
		return
	}

	awslib.WriteSuccessResponseXML(w, struct {
		XMLName xml.Name `xml:"DeleteHostedZoneResponse"`
		*aws53.DeleteHostedZoneOutput
	}{
		DeleteHostedZoneOutput: response,
	})
}

func (api *Api) GetChange(w http.ResponseWriter, r *http.Request) {

	// GET /2013-04-01/change/Id

	log.Println("Amazon-Target: Route53.GetChange")

	var request aws53.GetChangeInput

	// Parsing the path parameters
	vars := mux.Vars(r)
	request.Id = aws.String(vars["id"])

	response, err := api.service.GetChange(&request)
	if err != core.ErrNone {
		awslib.WriteErrorResponseXML(w, translateToApiError(err), r.URL, api.credentials.Region)
		return
	}

	awslib.WriteSuccessResponseXML(w, struct {
		XMLName xml.Name `xml:"GetChangeResponse"`
		*aws53.GetChangeOutput
	}{
		GetChangeOutput: response,
	})
}

func (api *Api) GetHostedZone(w http.ResponseWriter, r *http.Request) {

	// GET /2013-04-01/hostedzone/Id

	log.Println("Amazon-Target: Route53.GetHostedZone")

	var request aws53.GetHostedZoneInput

	// Parsing the path parameters
	vars := mux.Vars(r)
	request.Id = aws.String(vars["id"])

	response, err := api.service.GetHostedZone(&request)
	if err != core.ErrNone {
		awslib.WriteErrorResponseXML(w, translateToApiError(err), r.URL, api.credentials.Region)
		return
	}

	type DelegationSetWrapper struct {
		NameServers     []string `xml:"NameServers>NameServer"`
		CallerReference *string  `xml:",omitempty"`
		Id              *string  `xml:",omitempty"`
	}

	awslib.WriteSuccessResponseXML(w, struct {
		XMLName       xml.Name `xml:"GetHostedZoneResponse"`
		HostedZone    *awstypes.HostedZone
		DelegationSet DelegationSetWrapper
	}{
		HostedZone: response.HostedZone,
		DelegationSet: DelegationSetWrapper{
			NameServers:     response.DelegationSet.NameServers,
			CallerReference: response.DelegationSet.CallerReference,
			Id:              response.DelegationSet.Id,
		},
	})
}

func (api *Api) GetHostedZoneCount(w http.ResponseWriter, r *http.Request) {

	// GET /2013-04-01/hostedzonecount

	log.Println("Amazon-Target: Route53.GetHostedZoneCount")

	response, err := api.service.GetHostedZoneCount()
	if err != core.ErrNone {
		awslib.WriteErrorResponseXML(w, translateToApiError(err), r.URL, api.credentials.Region)
		return
	}

	awslib.WriteSuccessResponseXML(w, struct {
		XMLName xml.Name `xml:"GetHostedZoneCountResponse"`
		*aws53.GetHostedZoneCountOutput
	}{
		GetHostedZoneCountOutput: response,
	})
}

func (api *Api) ListHostedZonesByName(w http.ResponseWriter, r *http.Request) {

	log.Println("Amazon-Target: Route53.GetHostedZonesByName")

	var request aws53.ListHostedZonesByNameInput

	// Parsing the query parameters
	query := r.URL.Query()
	request.HostedZoneId = aws.String(query.Get("hostedzoneid"))
	request.DNSName = aws.String(query.Get("dnsname"))

	mi, merr := strconv.Atoi(query.Get("maxitems"))
	if merr == nil {
		request.MaxItems = aws.Int32(int32(mi))
	}

	response, err := api.service.ListHostedZonesByName(&request)
	if err != core.ErrNone {
		awslib.WriteErrorResponseXML(w, translateToApiError(err), r.URL, api.credentials.Region)
		return
	}

	awslib.WriteSuccessResponseXML(w, struct {
		XMLName          xml.Name              `xml:"ListHostedZonesByNameResponse"`
		HostedZones      []awstypes.HostedZone `xml:"HostedZones>HostedZone"`
		IsTruncated      bool
		MaxItems         *int32
		DNSName          *string
		HostedZoneId     *string
		NextDNSName      *string `xml:",omitempty"`
		NextHostedZoneId *string `xml:",omitempty"`
	}{
		HostedZones:      response.HostedZones,
		IsTruncated:      response.IsTruncated,
		DNSName:          response.DNSName,
		HostedZoneId:     response.HostedZoneId,
		MaxItems:         response.MaxItems,
		NextDNSName:      response.NextDNSName,
		NextHostedZoneId: response.NextHostedZoneId,
	})
}

func (api *Api) ListHostedZones(w http.ResponseWriter, r *http.Request) {

	// GET /2013-04-01/hostedzone?delegationsetid=DelegationSetId&hostedzonetype=HostedZoneType&marker=Marker&maxitems=MaxItems

	log.Println("Amazon-Target: Route53.ListHostedZone")

	var request aws53.ListHostedZonesInput

	// Parsing the query parameters
	query := r.URL.Query()
	request.DelegationSetId = aws.String(query.Get("delegationsetid"))
	request.HostedZoneType = awstypes.HostedZoneType(query.Get("hostedzonetype"))
	request.Marker = aws.String(query.Get("marker"))

	mi, merr := strconv.Atoi(query.Get("maxitems"))
	if merr == nil {
		request.MaxItems = aws.Int32(int32(mi))
	}

	response, err := api.service.ListHostedZones(&request)
	if err != core.ErrNone {
		awslib.WriteErrorResponseXML(w, translateToApiError(err), r.URL, api.credentials.Region)
		return
	}

	awslib.WriteSuccessResponseXML(w, struct {
		XMLName     xml.Name              `xml:"ListHostedZoneResponse"`
		HostedZones []awstypes.HostedZone `xml:"HostedZones>HostedZone"`
		IsTruncated bool
		MaxItems    *int32
		Marker      *string `xml:",omitempty"`
		NextMarker  *string `xml:",omitempty"`
	}{
		HostedZones: response.HostedZones,
		IsTruncated: response.IsTruncated,
		Marker:      response.Marker,
		MaxItems:    response.MaxItems,
		NextMarker:  response.NextMarker,
	})
}

func (api *Api) ListResourceRecordSets(w http.ResponseWriter, r *http.Request) {

	log.Println("Amazon-Target: Route53.ListResourceRecordSets")

	var request aws53.ListResourceRecordSetsInput

	vars := mux.Vars(r)
	request.HostedZoneId = aws.String(vars["id"])

	// Parsing the query parameters
	query := r.URL.Query()
	request.StartRecordName = aws.String(query.Get("name"))
	startRecordType := query.Get("type")
	if startRecordType != "" {
		request.StartRecordType = awstypes.RRType(startRecordType)
	}
	mi, merr := strconv.Atoi(query.Get("maxitems"))
	if merr == nil {
		request.MaxItems = aws.Int32(int32(mi))
	}

	response, err := api.service.ListResourceRecordSets(&request)
	if err != core.ErrNone {
		awslib.WriteErrorResponseXML(w, translateToApiError(err), r.URL, api.credentials.Region)
		return
	}

	awslib.WriteSuccessResponseXML(w, struct {
		XMLName            xml.Name `xml:"ListResourceRecordSetsResponse"`
		IsTruncated        bool
		MaxItems           *int32
		NextRecordName     string                  `xml:",omitempty"`
		NextRecordType     awstypes.RRType         `xml:",omitempty"`
		ResourceRecordSets []ResourceRecordSetData `xml:"ResourceRecordSets>ResourceRecordSet"`
	}{
		MaxItems:           request.MaxItems,
		IsTruncated:        response.NextRecord != "",
		NextRecordName:     response.NextRecord,
		NextRecordType:     response.NexType,
		ResourceRecordSets: response.Records,
	})
}

func (api *Api) ListTagsForResource(w http.ResponseWriter, r *http.Request) {

	log.Println("Amazon-Target: Route53.ListTagsForResource")

	var request aws53.ListTagsForResourceInput

	// Parsing the path parameters
	vars := mux.Vars(r)
	request.ResourceId = aws.String(vars["resourceId"])
	request.ResourceType = awstypes.TagResourceType(vars["resourceType"])

	response, err := api.service.ListTagsForResource(&request)
	if err != core.ErrNone {
		awslib.WriteErrorResponseXML(w, translateToApiError(err), r.URL, api.credentials.Region)
		return
	}

	type ResourceTagSetWrapper struct {
		ResourceId   *string
		ResourceType awstypes.TagResourceType
		Tags         []awstypes.Tag `xml:"Tags>Tag"`
	}

	awslib.WriteSuccessResponseXML(w, struct {
		XMLName        xml.Name `xml:"ListTagsForResourceResponse"`
		ResourceTagSet *ResourceTagSetWrapper
	}{
		ResourceTagSet: &ResourceTagSetWrapper{
			ResourceId:   response.ResourceTagSet.ResourceId,
			ResourceType: response.ResourceTagSet.ResourceType,
			Tags:         response.ResourceTagSet.Tags,
		},
	})
}

func (api *Api) UpdateHostedZoneComment(w http.ResponseWriter, r *http.Request) {

	log.Println("Amazon-Target: Route53.UpdateHostedZoneComment")

	type UpdateHostedZoneCommentWrapper struct {
		XMLName xml.Name `xml:"UpdateHostedZoneCommentRequest"`
		aws53.UpdateHostedZoneCommentInput
	}

	var request UpdateHostedZoneCommentWrapper

	if err := xml.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Println("Error:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Parsing the path parameters
	vars := mux.Vars(r)
	request.Id = aws.String(vars["id"])

	response, err := api.service.UpdateHostedZoneComment(&request.UpdateHostedZoneCommentInput)
	if err != core.ErrNone {
		awslib.WriteErrorResponseXML(w, translateToApiError(err), r.URL, api.credentials.Region)
		return
	}

	awslib.WriteSuccessResponseXML(w, struct {
		XMLName xml.Name `xml:"UpdateHostedZoneCommentResponse"`
		*aws53.UpdateHostedZoneCommentOutput
	}{
		UpdateHostedZoneCommentOutput: response,
	})
}
