package dbfcns

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"home-fern/internal/awslib"
	"home-fern/internal/core"
	"home-fern/internal/route53"
	"home-fern/internal/ssm"
	"home-fern/internal/tfstate"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
)

type Api struct {
	Ssm         *ssm.Service
	Route53     *route53.Service
	TfState     *tfstate.StateApi
	Credentials *core.BasicCredentialsProvider
}

func (api *Api) Keys(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	service, ok := vars["service"]
	if !ok {
		http.Error(w, "Service not specified", http.StatusBadRequest)
		return
	}

	var loggers = map[string]core.DatabaseDumper{}

	if service == "all" {
		loggers["ssm"] = api.Ssm
		loggers["route53"] = api.Route53
		loggers["tfstate"] = api.TfState
	} else if service == "ssm" {
		loggers["ssm"] = api.Ssm
	} else if service == "route53" {
		loggers["route53"] = api.Route53
	} else if service == "tfstate" {
		loggers["tfstate"] = api.TfState
	} else {
		http.Error(w, "Unsupported service for export", http.StatusBadRequest)
		return
	}

	for name, dumper := range loggers {
		fmt.Fprintf(w, "--- %s ---\n", name)
		err := dumper.LogKeys(w)
		if err != nil {
			log.Printf("Error dumping keys for %s: %v", name, err)
		}
		fmt.Fprintf(w, "\n")
	}
}

func (api *Api) Export(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	service, ok := vars["service"]
	if !ok {
		http.Error(w, "Service not specified", http.StatusBadRequest)
		return
	}

	if service == "all" {
		api.exportAll(w, r)
	} else if service == "ssm" {
		api.exportSsm(w, r)
	} else if service == "route53" {
		api.exportRoute53(w, r)
	} else {
		http.Error(w, "Unsupported service for export", http.StatusBadRequest)
	}
}

func (api *Api) Import(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	service, ok := vars["service"]
	if !ok {
		http.Error(w, "Service not specified", http.StatusBadRequest)
		return
	}

	if service == "all" {
		api.importAll(w, r)
	} else if service == "ssm" {
		api.importSsm(w, r)
	} else if service == "route53" {
		api.importRoute53(w, r)
	} else {
		http.Error(w, "Unsupported service for import", http.StatusBadRequest)
	}
}

func (api *Api) exportSsm(w http.ResponseWriter, r *http.Request) {
	parameters, err := api.Ssm.GetAllParameters()
	if err != core.ErrNone {
		log.Println("Error:", err)
		http.Error(w, "An error occurred", http.StatusInternalServerError)
		return
	}
	awslib.WriteSuccessResponseJSON(w, parameters)
}

func (api *Api) exportRoute53(w http.ResponseWriter, r *http.Request) {
	zones, err := api.Route53.ExportHostedZones()
	if err != core.ErrNone {
		log.Println("Error:", err)
		http.Error(w, "An error occurred", http.StatusInternalServerError)
		return
	}
	awslib.WriteSuccessResponseJSON(w, zones)
}

func (api *Api) exportAll(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\"home-fern-export.zip\"")

	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	// Export SSM
	ssmParams, err := api.Ssm.GetAllParameters()
	if err != core.ErrNone {
		log.Println("Error exporting SSM:", err)
		http.Error(w, "Error exporting SSM", http.StatusInternalServerError)
		return
	}
	if err := writeJsonToZip(zipWriter, "ssm.json", ssmParams); err != nil {
		log.Println("Error writing SSM to zip:", err)
		http.Error(w, "Error writing SSM to zip", http.StatusInternalServerError)
		return
	}

	// Export Route53
	r53Zones, err := api.Route53.ExportHostedZones()
	if err != core.ErrNone {
		log.Println("Error exporting Route53:", err)
		http.Error(w, "Error exporting Route53", http.StatusInternalServerError)
		return
	}
	if err := writeJsonToZip(zipWriter, "route53.json", r53Zones); err != nil {
		log.Println("Error writing Route53 to zip:", err)
		http.Error(w, "Error writing Route53 to zip", http.StatusInternalServerError)
		return
	}

	// Export TFState
	tfStateFiles, terr := api.TfState.ListStateFiles()
	if terr != nil {
		log.Println("Error listing TFState files:", terr)
		http.Error(w, "Error listing TFState files", http.StatusInternalServerError)
		return
	}

	for _, file := range tfStateFiles {
		content, err := api.TfState.GetStateContent(file)
		if err != nil {
			log.Println("Error reading TFState file:", file, err)
			continue
		}

		f, err := zipWriter.Create("tfstate/" + file)
		if err != nil {
			log.Println("Error creating zip entry for TFState:", file, err)
			continue
		}
		_, err = f.Write(content)
		if err != nil {
			log.Println("Error writing TFState content to zip:", file, err)
			continue
		}
	}
}

func (api *Api) importSsm(w http.ResponseWriter, r *http.Request) {
	requestUser := r.Context().Value(awslib.RequestUser)
	if requestUser == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	creds, ok := api.Credentials.FindCredentials(fmt.Sprintf("%v", requestUser))
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var parameters []ssm.ParameterData
	if err := json.NewDecoder(r.Body).Decode(&parameters); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	overwrite := r.Method == http.MethodPut

	failures, err := api.Ssm.ImportParameters(&creds, parameters, overwrite)
	if err != core.ErrNone {
		log.Println("Error:", err)
		http.Error(w, "An error occurred", http.StatusInternalServerError)
		return
	}

	awslib.WriteSuccessResponseJSON(w, map[string]interface{}{"failures": failures})
}

func (api *Api) importRoute53(w http.ResponseWriter, r *http.Request) {
	var zones []route53.HostedZoneExport
	if err := json.NewDecoder(r.Body).Decode(&zones); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	overwrite := r.Method == http.MethodPut

	failures, err := api.Route53.ImportHostedZones(zones, overwrite)
	if err != core.ErrNone {
		log.Println("Error:", err)
		http.Error(w, "An error occurred", http.StatusInternalServerError)
		return
	}

	awslib.WriteSuccessResponseJSON(w, map[string]interface{}{"failures": failures})
}

func (api *Api) importAll(w http.ResponseWriter, r *http.Request) {
	requestUser := r.Context().Value(awslib.RequestUser)
	if requestUser == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	creds, ok := api.Credentials.FindCredentials(fmt.Sprintf("%v", requestUser))
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		log.Println("Error retrieving file from request:", err)
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	tempFile, err := os.CreateTemp("", "import-*.zip")
	if err != nil {
		log.Println("Error creating temp file:", err)
		http.Error(w, "Error creating temp file", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tempFile.Name())

	_, err = io.Copy(tempFile, file)
	if err != nil {
		log.Println("Error saving uploaded file:", err)
		http.Error(w, "Error saving uploaded file", http.StatusInternalServerError)
		return
	}

	zipReader, err := zip.OpenReader(tempFile.Name())
	if err != nil {
		log.Println("Error opening zip file:", err)
		http.Error(w, "Error opening zip file", http.StatusBadRequest)
		return
	}
	defer zipReader.Close()

	// Delete existing data before import
	// Note: This is a destructive operation as requested
	// "basically deleting everything first and then importing"

	// Delete TFState
	if err := api.TfState.DeleteAllStates(); err != nil {
		log.Println("Error deleting TFState:", err)
		http.Error(w, "Error deleting TFState data", http.StatusInternalServerError)
		return
	}

	// Wipe SSM
	if err := api.Ssm.DeleteAllData(); err != core.ErrNone {
		log.Println("Error deleting SSM data:", err)
		http.Error(w, "Error deleting SSM data", http.StatusInternalServerError)
		return
	}

	// Wipe Route53
	if err := api.Route53.DeleteAllData(); err != core.ErrNone {
		log.Println("Error deleting Route53 data:", err)
		http.Error(w, "Error deleting Route53 data", http.StatusInternalServerError)
		return
	}

	var ssmFailures []string
	var r53Failures []string

	for _, f := range zipReader.File {
		rc, err := f.Open()
		if err != nil {
			log.Println("Error opening file in zip:", f.Name, err)
			http.Error(w, "Error opening file in zip", http.StatusInternalServerError)
			return
		}

		if f.Name == "ssm.json" {
			var params []ssm.ParameterData
			if err := json.NewDecoder(rc).Decode(&params); err != nil {
				log.Println("Error decoding ssm.json:", err)
				http.Error(w, "Error decoding ssm.json", http.StatusInternalServerError)
				return
			} else {
				failures, err := api.Ssm.ImportParameters(&creds, params, true)
				if err != core.ErrNone {
					log.Println("Error importing SSM:", err)
				}
				ssmFailures = append(ssmFailures, failures...)
			}
		} else if f.Name == "route53.json" {
			var zones []route53.HostedZoneExport
			if err := json.NewDecoder(rc).Decode(&zones); err != nil {
				log.Println("Error decoding route53.json:", err)
				http.Error(w, "Error decoding route53.json", http.StatusInternalServerError)
				return
			} else {
				failures, err := api.Route53.ImportHostedZones(zones, true)
				if err != core.ErrNone {
					log.Println("Error importing Route53:", err)
				}
				r53Failures = append(r53Failures, failures...)
			}
		} else if strings.HasPrefix(f.Name, "tfstate/") {
			// Import TFState
			filename := filepath.Base(f.Name)
			// We need to read content
			content, err := io.ReadAll(rc)
			if err == nil {
				if err := api.TfState.SaveStateContent(filename, content); err != nil {
					log.Println("Error saving TFState:", filename, err)
					http.Error(w, "Error saving TFState", http.StatusInternalServerError)
					return
				}
			}
			rc.Close()
		}
	}

	awslib.WriteSuccessResponseJSON(w, map[string]interface{}{
		"message":     "Import processed",
		"ssmFailures": ssmFailures,
		"r53Failures": r53Failures,
	})
}

func writeJsonToZip(zipWriter *zip.Writer, filename string, data interface{}) error {
	f, err := zipWriter.Create(filename)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(f)
	return encoder.Encode(data)
}
