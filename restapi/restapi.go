package restapi

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

var signal chan bool

// AddDomainEndPoint allows us to add the requested domain
func AddDomainEndPoint(w http.ResponseWriter, req *http.Request) {
	// update hostnames list
	// restart the CT log crawler using the channel
	params := mux.Vars(req)

	log.Fatalf("Not implemented %s", params)
}

// GetDomainsEndPoint returns all monitored domains
func GetDomainsEndPoint(w http.ResponseWriter, req *http.Request) {
	// from DB or just from controller/config?
	params := mux.Vars(req)

	log.Fatalf("Not implemented %s", params)
}

// GetDomainEndPoint returns info for specified domain
func GetDomainEndPoint(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)

	log.Fatalf("Not implemented %s", params)
}

// GetNewCertificatesEndPoint returns all new certificates found
func GetNewCertificatesEndPoint(w http.ResponseWriter, req *http.Request) {
	// TODO define "new"?
	params := mux.Vars(req)

	log.Fatalf("Not implemented %s", params)
}

// RemoveDomainEndPoint removes specified domain
func RemoveDomainEndPoint(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	log.Fatalf("Not implemented %s", params)
}

// Init initializes rest api with a channel to signal changes back to the controller
func Init(signal chan bool) {
	// TODO get handle to data in controller?

	log.Fatal("not implemented")
	signal = signal
}

// Run the rest api, handle connections, update host lists, etc.
func Run() {
	log.Fatal("not implemented")
}
