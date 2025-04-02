package core

import (
	"context"
	"crypto/subtle"
	"home-fern/internal/awslib"
	"net/http"
)

type BasicCredentialsProvider struct {
	credentials map[string]string
}

func NewBasicCredentialsProvider(credentials []FernCredentials) *BasicCredentialsProvider {

	result := BasicCredentialsProvider{
		credentials: make(map[string]string),
	}

	for _, c := range credentials {
		result.credentials[c.AccessKey] = c.SecretKey
	}

	return &result
}

func (p *BasicCredentialsProvider) FindCredentials(accessKey string) (string, bool) {

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

	tpass, found := p.FindCredentials(user)
	if !found {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(pass), []byte(tpass)) == 1
}
