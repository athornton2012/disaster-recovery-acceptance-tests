package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"code.cloudfoundry.org/credhub-cli/credhub"
	"code.cloudfoundry.org/credhub-cli/credhub/auth"
)

const credhubBaseURL = "https://credhub.service.cf.internal:8844"
const credentialName = "SECRET_PASSWORD"

func main() {
	s, err := newServer()
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/create", s.Create)
	http.HandleFunc("/list", s.List)

	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), nil))
}

type Server struct {
	client  *credhub.CredHub
	counter int
}

func (s *Server) Create(w http.ResponseWriter, r *http.Request) {
	s.counter++
	resp, err := s.client.Request("POST", fmt.Sprintf("%s/api/v1/data", credhubBaseURL),nil,
		strings.NewReader(fmt.Sprintf(`{"name": "/%s/%d", "type": "password"}`, credentialName, s.counter)),true)
	if ok := handleBadResponses(w, resp, err); !ok {
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) List(w http.ResponseWriter, r *http.Request) {
	resp, err := s.client.Request("GET", fmt.Sprintf("%s/api/v1/data?path=%s", credhubBaseURL, credentialName),nil,nil,true)
	if ok := handleBadResponses(w, resp, err); !ok {
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Encountered error reading response body from credhub: [%s]", err)
	}
	defer resp.Body.Close()

	fmt.Fprintf(w, string(body))
}

func handleBadResponses(w http.ResponseWriter, resp *http.Response, err error) bool {
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Encountered error from credhub: [%s]", err)
		return false
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		w.WriteHeader(resp.StatusCode)
		fmt.Fprintf(w, "Encountered bad response code from credhub: %d", resp.StatusCode)
		return false
	}
	return true
}

func newServer() (*Server, error) {
	clientCertPath := os.Getenv("CF_INSTANCE_CERT")
	clientKeyPath := os.Getenv("CF_INSTANCE_KEY")

	_, err := os.Stat(clientCertPath)
	if err != nil {
		return nil, err
	}
	_, err = os.Stat(clientKeyPath)
	if err != nil {
		return nil, err
	}

	client, err := credhub.New(
		credhubBaseURL,
		credhub.SkipTLSValidation(true),
		credhub.Auth(auth.UaaClientCredentials(os.Getenv("CREDHUB_CLIENT"), os.Getenv("CREDHUB_SECRET"))))

	return &Server{client: client}, err
}
