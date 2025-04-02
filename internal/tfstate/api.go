package tfstate

import (
	"github.com/gorilla/mux"
	"io"
	"log"
	"net/http"
	"os"
)

type StateApi struct {
	storagePath string
}

func NewStateApi(storagePath string) *StateApi {

	info, err := os.Stat(storagePath)
	if os.IsNotExist(err) {

		err = os.Mkdir(storagePath, 0755)
		if err != nil {
			log.Panicln("Unable to create TF State storage path.", err)
		}
	}

	info, _ = os.Stat(storagePath)
	if !info.IsDir() {

		log.Panicln("TF State storage path is not a directory.")
	}

	return &StateApi{
		storagePath: storagePath,
	}
}

func (s *StateApi) createFileName(project string) string {

	return s.storagePath + "/" + project + ".json"
}

func (s *StateApi) createLockFileName(project string) string {

	return s.createFileName(project) + ".lock"
}

func (s *StateApi) GetState(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	filePath := s.createFileName(vars["project"])
	log.Println("GetState", filePath)

	w.Header().Set("Content-Type", "application/json")

	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		// send back nothing
		w.WriteHeader(http.StatusOK)
		return
	}

	err = readFile(filePath, w)
	if err != nil {
		log.Println("Error", err)
		http.Error(w, "An error occurred.", http.StatusInternalServerError)
		return
	}
}

func (s *StateApi) SaveState(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	filePath := s.createFileName(vars["project"])
	log.Println("SaveState", filePath)

	err := saveFile(filePath, r)
	if err != nil {
		log.Println("Error", err)
		http.Error(w, "An error occurred.", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *StateApi) LockState(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	filePath := s.createLockFileName(vars["project"])

	log.Println("LockState", filePath)

	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {

		err := saveFile(filePath, r)
		if err != nil {
			log.Println("Error", err)
			http.Error(w, "An error occurred.", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		return
	}

	// lock already exists
	w.Header().Set("Content-Type", "application/json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Println("Error", err)
		http.Error(w, "An error occurred.", http.StatusInternalServerError)
		return
	}

	w.Write(data)
	w.WriteHeader(http.StatusLocked)
}

func (s *StateApi) DeleteState(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	filePath := s.createFileName(vars["project"])

	log.Println("DeleteState", filePath)

	err := os.Remove(filePath)
	if err != nil {
		log.Println("Error", err)
		http.Error(w, "An error occurred.", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *StateApi) UnlockState(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	filePath := s.createLockFileName(vars["project"])

	log.Println("UnlockState", filePath)

	err := os.Remove(filePath)
	if err != nil {
		log.Println("Error", err)
		http.Error(w, "An error occurred.", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func readFile(filePath string, w http.ResponseWriter) error {

	// Open the file for reading.
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Copy the file's contents to the HTTP response writer.
	_, err = io.Copy(w, file)
	if err != nil {
		return err
	}

	return nil
}

func saveFile(filePath string, r *http.Request) error {

	// Open the file for writing, creating it if it doesn't exist, and truncating it if it does.
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Read the request body and write it to the file.
	_, err = io.Copy(file, r.Body)
	if err != nil {
		return err
	}

	return nil
}

type CustomWriter struct {
	http.ResponseWriter
}

func (w CustomWriter) Write(p []byte) (n int, err error) {
	return w.ResponseWriter.Write(p)
}

func (w CustomWriter) WriteHeader(statusCode int) {
	// do nothing
}
