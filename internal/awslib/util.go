package awslib

import (
	"log/slog"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func LogEndpoint(r *http.Request, amztarget string, creds aws.Credentials) {

	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.RemoteAddr
	}

	slog.Info("Endpoint Hit",
		"Amazon-Target", amztarget, "Source", creds.Source, "ip", ip)
}
