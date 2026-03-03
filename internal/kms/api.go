package kms

import (
	"encoding/json"
	"fmt"
	"home-fern/internal/awslib"
	"log"
	"net/http"

	awskms "github.com/aws/aws-sdk-go-v2/service/kms"
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

	creds, _ := api.credentials.FindCredentials(fmt.Sprintf("%v", requestUser))

	amztarget := r.Header.Get("X-Amz-Target")

	awslib.LogEndpoint(r, amztarget, creds)

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
	if err != nil {

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
	if err != nil {

		log.Println("Error:", err)
		awslib.WriteErrorResponseJSON(w, translateToApiError(err), r.URL, api.credentials.Region)
		return
	}

	awslib.WriteSuccessResponseJSON(w, response)
}

func translateToApiError(err error) awslib.ApiError {

	switch err {
	case ErrInvalidKeyId:
		return awslib.ApiError{
			Code:           "InvalidKeyIdException",
			Description:    err.Error(),
			HTTPStatusCode: http.StatusBadRequest,
		}
	case ErrInvalidCiphertextException:
		return awslib.ApiError{
			Code:           "InvalidCiphertextException",
			Description:    err.Error(),
			HTTPStatusCode: http.StatusBadRequest,
		}
	case ErrKMSInternalException:
		return awslib.ApiError{
			Code:           "KMSInternalException",
			Description:    err.Error(),
			HTTPStatusCode: http.StatusInternalServerError,
		}
	default:
		return awslib.ErrorCodes[awslib.ErrInternalError]
	}
}
