package route53

import (
	"encoding/json"
	aws53 "github.com/aws/aws-sdk-go-v2/service/route53"
	"home-fern/internal/awslib"
	"home-fern/internal/core"
	"log"
	"net/http"
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

/*
* create-hosted-zone
* delete-hosted-zone
* get-hosted-zone
* list-hosted-zone
* change-tags-for-resource
* list-tags-for-resource
* list-tags-for-resources
* change-resource-record-sets
 */

func (api *Api) CreateHostedZone(w http.ResponseWriter, r *http.Request) {

	var request aws53.CreateHostedZoneInput
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := api.service.CreateHostedZone(&request)
	if err != core.ErrNone {
		log.Println("Error:", err)
		awslib.WriteErrorResponseJSON(w, translateToApiError(err), r.URL, api.credentials.Region)
		return
	}

	awslib.WriteSuccessResponseJSON(w, response)
}

func (api *Api) ListHostedZones(w http.ResponseWriter, r *http.Request) {

	var request aws53.ListHostedZonesInput
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := api.service.ListHostedZones(&request)
	if err != core.ErrNone {
		log.Println("Error:", err)
		awslib.WriteErrorResponseJSON(w, translateToApiError(err), r.URL, api.credentials.Region)
		return
	}

	awslib.WriteSuccessResponseJSON(w, response)
}

func translateToApiError(_ core.ErrorCode) awslib.ApiError {

	return awslib.ErrorCodes[awslib.ErrInternalError]
}
