package core

import (
	"context"
	"crypto/subtle"
	"home-fern/internal/awslib"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
)

type BasicCredentialsProvider struct {
	Region      string
	credentials map[string]aws.Credentials
}

func NewBasicCredentialsProvider(region string, credentials []FernCredentials) *BasicCredentialsProvider {

	result := BasicCredentialsProvider{
		Region:      region,
		credentials: make(map[string]aws.Credentials),
	}

	for _, c := range credentials {
		result.credentials[c.AccessKey] = aws.Credentials{
			AccessKeyID:     c.AccessKey,
			SecretAccessKey: c.SecretKey,
			Source:          c.Username,
			AccountID:       ZeroAccountId,
		}
	}

	return &result
}

func (p *BasicCredentialsProvider) FindCredentials(accessKey string) (aws.Credentials, bool) {

	v, ok := p.credentials[accessKey]

	return v, ok
}

func (p *BasicCredentialsProvider) WithBasicAuth(next http.HandlerFunc) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		user, pass, ok := r.BasicAuth()
		if !ok || !p.checkBasicAuth(user, pass) {

			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), awslib.RequestUser, user)

		// Call the next handler.
		next(w, r.WithContext(ctx))
	}
}

func (p *BasicCredentialsProvider) checkBasicAuth(user string, pass string) bool {

	creds, found := p.FindCredentials(user)
	if !found {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(pass), []byte(creds.SecretAccessKey)) == 1
}
