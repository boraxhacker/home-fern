package route53

import (
	"home-fern/internal/awslib"
	"net/http"
)

type Api struct {
	credentials *awslib.CredentialsProvider
}

func NewRoute53Api(credentials *awslib.CredentialsProvider) *Api {

	return &Api{credentials: credentials}
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

}

func (api *Api) ListHostedZones(w http.ResponseWriter, r *http.Request) {

}
