package main

import (
	"flag"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"
	"home-fern/internal/awslib"
	"home-fern/internal/core"
	"home-fern/internal/route53"
	"home-fern/internal/ssm"
	"home-fern/internal/tfstate"
	"log"
	"net/http"
	"os"
)

func main() {

	configFilePtr :=
		flag.String("config", ".home-fern-config.yaml", "Path to the home-fern config file.")
	dataPathPtr :=
		flag.String("data-path", ".home-fern-data", "Path to data store folder.")
	flag.Parse()

	fernConfig := readAuthCredsOrDie(*configFilePtr)
	simplePrintConfig(fernConfig)

	var credentials []aws.Credentials
	for _, cred := range fernConfig.Credentials {
		credentials = append(credentials, aws.Credentials{
			AccessKeyID:     cred.AccessKey,
			SecretAccessKey: cred.SecretKey,
			Source:          cred.Username,
			AccountID:       core.ZeroAccountId,
		})
	}
	credentialsProvider := awslib.NewCredentialsProvider(awslib.ServiceSsm, fernConfig.Region, credentials)

	ssmsvc := ssm.NewService(fernConfig, core.ZeroAccountId, *dataPathPtr)
	defer ssmsvc.Close()

	ssmApi := ssm.NewParameterApi(ssmsvc, credentialsProvider)

	r53svc := route53.NewService(*dataPathPtr)
	defer r53svc.Close()

	route53Api := route53.NewRoute53Api(r53svc, credentialsProvider)

	basicProvider := core.NewBasicCredentialsProvider(fernConfig.Credentials)

	stateApi := tfstate.NewStateApi(*dataPathPtr + "/tfstate")

	router := mux.NewRouter()

	// SSM
	router.HandleFunc("/ssm",
		credentialsProvider.WithSigV4(ssmApi.Handle)).Methods("POST")

	// Route53
	router.HandleFunc("/route53/2013-04-01/hostedzone",
		credentialsProvider.WithSigV4(route53Api.CreateHostedZone)).Methods("POST")
	router.HandleFunc("/route53/2013-04-01/hostedzone",
		credentialsProvider.WithSigV4(route53Api.ListHostedZones)).Methods("GET")

	//router.HandleFunc("/route53/2013-04-01/change/{id}",
	//	credentialsProvider.WithSigV4(route53Api.GetChange)).Methods("GET")
	//router.HandleFunc("/route53/2013-04-01/hostedzone/{id}/rrset",
	//	credentialsProvider.WithSigV4(route53Api.ChangeRecordSets)).Methods("POST")
	//router.HandleFunc("/route53/2013-04-01/hostedzone/{id}",
	//	credentialsProvider.WithSigV4(route53Api.GetHostedZone)).Methods("GET")
	//router.HandleFunc("/route53/2013-04-01/hostedzone/{id}",
	//	credentialsProvider.WithSigV4(route53Api.DeleteHostedZone)).Methods("DELETE")

	// TF State
	router.HandleFunc("/tfstate/{project}",
		basicProvider.WithBasicAuth(stateApi.GetState)).Methods("GET")
	router.HandleFunc("/tfstate/{project}",
		basicProvider.WithBasicAuth(stateApi.SaveState)).Methods("POST")
	router.HandleFunc("/tfstate/{project}",
		basicProvider.WithBasicAuth(stateApi.DeleteState)).Methods("DELETE")
	router.HandleFunc("/tfstate/{project}/lock",
		basicProvider.WithBasicAuth(stateApi.LockState)).Methods("LOCK")
	router.HandleFunc("/tfstate/{project}/unlock",
		basicProvider.WithBasicAuth(stateApi.UnlockState)).Methods("UNLOCK")

	addr := ":9080"
	log.Printf("Listening on %s", addr)
	http.Handle("/", router)
	http.ListenAndServe(addr, nil)
}

func readAuthCredsOrDie(configFileName string) *core.FernConfig {

	configFile, err := os.ReadFile(configFileName) // Replace with your yaml file name/path
	if err != nil {
		log.Panicln("Error reading config file:", err)
	}

	var fernConfig core.FernConfig
	err = yaml.Unmarshal(configFile, &fernConfig)
	if err != nil {
		log.Panicln("Error unmarshalling config:", err)
	}

	return &fernConfig
}

func simplePrintConfig(config *core.FernConfig) {

	log.Println("Region:", config.Region)

	log.Println("Credentials:")
	for i, cred := range config.Credentials {
		log.Printf("\tAccessKey %02d: %s\n", i+1, cred.AccessKey)
	}

	log.Println("Keys:")
	for i, key := range config.Keys {
		log.Printf("\tKMS Key %02d: alias/%s\n", i+1, key.Alias)
	}
}
