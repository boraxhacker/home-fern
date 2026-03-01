package tfstate

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
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

func (s *StateApi) ListStateFiles() ([]string, error) {
	var files []string
	entries, err := os.ReadDir(s.storagePath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			files = append(files, entry.Name())
		}
	}
	return files, nil
}

func (s *StateApi) GetStateContent(filename string) ([]byte, error) {
	return os.ReadFile(filepath.Join(s.storagePath, filename))
}

func (s *StateApi) SaveStateContent(filename string, content []byte) error {
	return os.WriteFile(filepath.Join(s.storagePath, filename), content, 0644)
}

func (s *StateApi) DeleteAllStates() error {
	entries, err := os.ReadDir(s.storagePath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".json") || strings.HasSuffix(entry.Name(), ".lock")) {
			err := os.Remove(filepath.Join(s.storagePath, entry.Name()))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *StateApi) LogKeys(writer io.Writer) error {
	files, err := s.ListStateFiles()
	if err != nil {
		return err
	}

	for _, file := range files {
		_, err := fmt.Fprintf(writer, "%s\n", file)
		if err != nil {
			return err
		}
	}
	return nil
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
