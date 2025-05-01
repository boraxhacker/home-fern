package kms

import (
	"encoding/json"
	awskms "github.com/aws/aws-sdk-go-v2/service/kms"
	"home-fern/internal/awslib"
	"home-fern/internal/core"
	"log"
	"net/http"
)

type Api struct {
	service     *Service
	credentials *awslib.CredentialsProvider
}

func NewKmsApi(service *Service, credentials *awslib.CredentialsProvider) *Api {

	return &Api{service: service, credentials: credentials}
}

/*
o encrypt
o decrypt
*/

func (api *Api) Handle(w http.ResponseWriter, r *http.Request) {

	requestUser := r.Context().Value(awslib.RequestUser)
	if requestUser == nil {
		awslib.WriteErrorResponseJSON(w, awslib.ErrorCodes[awslib.ErrInternalError], r.URL, api.credentials.Region)
		return
	}

	amztarget := r.Header.Get("X-Amz-Target")
	log.Printf("X-Amz-Target: %s\n", amztarget)
	if amztarget == "TrentService.Encrypt" {

		api.encrypt(w, r)

	} else if amztarget == "TrentService.Decrypt" {

		api.decrypt(w, r)
	}
}

func (api *Api) decrypt(w http.ResponseWriter, r *http.Request) {

	var request awskms.DecryptInput
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := api.service.Decrypt(&request)
	if err != core.ErrNone {

		log.Println("Error:", err)
		awslib.WriteErrorResponseJSON(w, translateToApiError(err), r.URL, api.credentials.Region)
		return
	}

	awslib.WriteSuccessResponseJSON(w, response)
}

func (api *Api) encrypt(w http.ResponseWriter, r *http.Request) {

	var request awskms.EncryptInput
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := api.service.Encrypt(&request)
	if err != core.ErrNone {

		log.Println("Error:", err)
		awslib.WriteErrorResponseJSON(w, translateToApiError(err), r.URL, api.credentials.Region)
		return
	}

	awslib.WriteSuccessResponseJSON(w, response)
}
