package awslib

// borrowed from Minio project, dumped in this package

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

// mimeType represents various MIME type used API responses.
type mimeType string

const (
	// Means no response type.
	mimeNone mimeType = ""
	// Means response type is JSON.
	mimeJSON mimeType = "application/json"
	mimeXML  mimeType = "application/xml"
)

// getErrorResponse gets in standard error and resource value and
// provides a encodable populated response values
func createAPIErrorResponse(
	err ApiError, resource, requestID, hostID string, region string) apiErrorResponse {

	return apiErrorResponse{
		Code:      err.Code,
		Message:   err.Description,
		Resource:  resource,
		Region:    region,
		RequestID: requestID,
		HostID:    hostID,
	}
}

func WriteSuccessResponseJSON(w http.ResponseWriter, response interface{}) {

	encodedResponse := encodeResponseJSON(response)
	// log.Println("Response", string(encodedResponse))

	writeResponse(w, http.StatusOK, encodedResponse, mimeJSON)
}

func WriteSuccessResponseXML(w http.ResponseWriter, response interface{}) {

	encodedResponse := encodeResponseXML(response)
	// log.Println("Response", string(encodedResponse))

	writeResponse(w, http.StatusOK, encodedResponse, mimeXML)
}

// WriteErrorResponseJSON  - writes error response in JSON format;
// useful for admin APIs.
func WriteErrorResponseJSON(w http.ResponseWriter, err ApiError, reqURL *url.URL, region string) {

	// Generate error response.
	errorResponse := createAPIErrorResponse(err, reqURL.Path,
		w.Header().Get(headerAmzRequestID), w.Header().Get(headerAmzRequestHostID), region)

	encodedErrorResponse := encodeResponseJSON(errorResponse)
	writeResponse(w, err.HTTPStatusCode, encodedErrorResponse, mimeJSON)
}

func WriteErrorResponseXML(w http.ResponseWriter, err ApiError, reqURL *url.URL, region string) {

	// Generate error response.
	resp := createAPIErrorResponse(err, reqURL.Path,
		w.Header().Get(headerAmzRequestID), w.Header().Get(headerAmzRequestHostID), region)

	encodedErrorResponse := encodeResponseXML(struct {
		XMLName xml.Name `xml:"ErrorResponse"`
		Error   struct {
			Type    string
			Code    string
			Message string
		}
		RequestId string
		HostId    string
	}{
		Error: struct {
			Type    string
			Code    string
			Message string
		}{
			Type:    "Sender",
			Code:    resp.Code,
			Message: resp.Message,
		},
		RequestId: resp.RequestID,
		HostId:    resp.HostID,
	})

	writeResponse(w, err.HTTPStatusCode, encodedErrorResponse, mimeXML)
}

func writeResponse(w http.ResponseWriter, statusCode int, response []byte, mType mimeType) {
	if statusCode == 0 {
		statusCode = 200
	}
	// Similar check to http.checkWriteHeaderCode
	if statusCode < 100 || statusCode > 999 {
		log.Printf("invalid WriteHeader code %v", statusCode)
		statusCode = http.StatusInternalServerError
	}
	w.WriteHeader(statusCode)

	w.Header().Set(headerServerInfo, "home-fern")
	w.Header().Set(headerAcceptRanges, "bytes")
	if mType != mimeNone {
		w.Header().Set(headerContentType, string(mType))
	}
	w.Header().Set(headerContentLength, strconv.Itoa(len(response)))

	if response != nil {
		w.Write(response)
	}
}

// Encodes the response headers into JSON format.
func encodeResponseJSON(response interface{}) []byte {

	var bytesBuffer bytes.Buffer
	e := json.NewEncoder(&bytesBuffer)
	e.Encode(response)
	return bytesBuffer.Bytes()
}

func encodeResponseXML(response interface{}) []byte {

	var bytesBuffer bytes.Buffer

	bytesBuffer.Write([]byte(xml.Header))

	e := xml.NewEncoder(&bytesBuffer)
	e.Encode(response)
	return bytesBuffer.Bytes()
}
