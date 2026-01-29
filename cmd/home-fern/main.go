package main

import (
	"flag"
	"home-fern/internal/awslib"
	"home-fern/internal/core"
	"home-fern/internal/dbdump"
	"home-fern/internal/kms"
	"home-fern/internal/route53"
	"home-fern/internal/ssm"
	"home-fern/internal/tfstate"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"
)

func main() {

	configFilePtr :=
		flag.String("config", ".home-fern-config.yaml", "Path to the home-fern config file.")
	dataPathPtr :=
		flag.String("data-path", ".home-fern-data", "Path to data store folder.")
	webPathPtr :=
		flag.String("web-path", "./web/dist/home-fern-web/browser", "Path to web files.")
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
	kmsCredentials := awslib.NewCredentialsProvider(awslib.ServiceKms, fernConfig.Region, credentials)

	kmssvc := kms.NewService(fernConfig, core.ZeroAccountId)

	kmsApi := kms.NewKmsApi(kmssvc, kmsCredentials)

	ssmCredentials := awslib.NewCredentialsProvider(awslib.ServiceSsm, fernConfig.Region, credentials)

	ssmsvc := ssm.NewService(fernConfig, core.ZeroAccountId, *dataPathPtr)
	defer ssmsvc.Close()

	ssmApi := ssm.NewParameterApi(ssmsvc, ssmCredentials)

	r53svc := route53.NewService(&fernConfig.DnsDefaults, *dataPathPtr)
	defer r53svc.Close()

	route53Credentials := awslib.NewCredentialsProvider(awslib.ServiceRoute53, fernConfig.Region, credentials)

	route53Api := route53.NewRoute53Api(r53svc, route53Credentials)

	basicProvider := core.NewBasicCredentialsProvider(fernConfig.Credentials)

	stateApi := tfstate.NewStateApi(*dataPathPtr + "/tfstate")

	var dumpApi = dbdump.Api{
		Loggers: map[string]core.DatabaseDumper{
			"route53": r53svc,
			"ssm":     ssmsvc,
		},
	}

	router := mux.NewRouter()

	// dump
	router.HandleFunc("/keys/{service}",
		basicProvider.WithBasicAuth(dumpApi.LogKeys)).Methods("GET")

	// KMS
	router.HandleFunc("/kms{slash:/?}",
		kmsCredentials.WithSigV4(kmsApi.Handle)).Methods("POST")

	// SSM
	router.HandleFunc("/ssm{slash:/?}",
		ssmCredentials.WithSigV4(ssmApi.Handle)).Methods("POST")

	// Route53
	router.HandleFunc("/route53/2013-04-01/hostedzonesbyname",
		route53Credentials.WithSigV4(route53Api.ListHostedZonesByName)).Methods("GET")
	router.HandleFunc("/route53/2013-04-01/hostedzone/{id}/rrset",
		route53Credentials.WithSigV4(route53Api.ListResourceRecordSets)).Methods("GET")
	router.HandleFunc("/route53/2013-04-01/hostedzone/{id}/rrset{slash:/?}",
		route53Credentials.WithSigV4(route53Api.ChangeResourceRecordSets)).Methods("POST")
	router.HandleFunc("/route53/2013-04-01/hostedzone/{id}",
		route53Credentials.WithSigV4(route53Api.UpdateHostedZoneComment)).Methods("POST")
	router.HandleFunc("/route53/2013-04-01/hostedzone/{id}",
		route53Credentials.WithSigV4(route53Api.DeleteHostedZone)).Methods("DELETE")
	router.HandleFunc("/route53/2013-04-01/hostedzone/{id}",
		route53Credentials.WithSigV4(route53Api.GetHostedZone)).Methods("GET")
	router.HandleFunc("/route53/2013-04-01/hostedzone{slash:/?}",
		route53Credentials.WithSigV4(route53Api.CreateHostedZone)).Methods("POST")
	router.HandleFunc("/route53/2013-04-01/hostedzone",
		route53Credentials.WithSigV4(route53Api.ListHostedZones)).Methods("GET")
	router.HandleFunc("/route53/2013-04-01/hostedzonecount",
		route53Credentials.WithSigV4(route53Api.GetHostedZoneCount)).Methods("GET")
	router.HandleFunc("/route53/2013-04-01/change/{id}",
		route53Credentials.WithSigV4(route53Api.GetChange)).Methods("GET")
	router.HandleFunc("/route53/2013-04-01/tags/{resourceType}/{resourceId}",
		route53Credentials.WithSigV4(route53Api.ListTagsForResource)).Methods("GET")
	router.HandleFunc("/route53/2013-04-01/tags/{resourceType}/{resourceId}",
		route53Credentials.WithSigV4(route53Api.ChangeTagsForResource)).Methods("POST")

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

	router.PathPrefix("/").Handler(http.FileServer(http.Dir(*webPathPtr)))

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

	log.Println("Kms:")
	for i, key := range config.Keys {
		log.Printf("\tKMS Key %02d: alias/%s\n", i+1, key.Alias)
	}

	log.Println("Dns:")
	log.Printf("\tSOA: %s\n", config.DnsDefaults.Soa)
	for i, ns := range config.DnsDefaults.NameServers {
		log.Printf("\tNameserver %02d: %s\n", i+1, ns)
	}
}
