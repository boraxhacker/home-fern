package awslib

import (
	"encoding/xml"
	"net/http"
)

type AwsErrorResponse struct {
	XMLName xml.Name `xml:"Error"`
	Code    string   `xml:"Code"`
	Message string   `xml:"Message"`
}

func WriteAwsError(w http.ResponseWriter, httpStatus int, errorResponse AwsErrorResponse) {
	w.WriteHeader(httpStatus)
	w.Header().Set("Content-Type", "application/xml")
	if err := xml.NewEncoder(w).Encode(errorResponse); err != nil {
		http.Error(w, "Failed to serialize error response", http.StatusInternalServerError)
	}
}
