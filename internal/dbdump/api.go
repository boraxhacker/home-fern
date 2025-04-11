package dbdump

import (
	"github.com/gorilla/mux"
	"home-fern/internal/core"
	"log"
	"net/http"
)

type Api struct {
	Loggers map[string]core.DatabaseDumper
}

func (api *Api) LogKeys(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	dumper, ok := api.Loggers[vars["service"]]
	if !ok {
		log.Printf("Error - Service %s does not exist.\n", vars["service"])
		http.Error(w, "An error occurred.", http.StatusInternalServerError)
		return
	}

	err := dumper.LogKeys(w)
	if err != nil {
		log.Println("Error", err)
		http.Error(w, "An error occurred.", http.StatusInternalServerError)
		return
	}
}
