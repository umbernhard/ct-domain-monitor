package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

// AddDomainEndpoint allows us to add the requested domain
func AddDomainEndpoint(w http.ResponseWriter, req *http.Request) {
	// update hostnames list
	// restart the CT log crawler using the channel
	domain := mux.Vars(req)["domain"]

	log.Fatalf("Not implemented %s", params)
}

// GetDomainsEndpoint returns all monitored domains
func GetDomainsEndpoint(w http.ResponseWriter, req *http.Request) {
	// from DB or just from controller/config?
	params := mux.Vars(req)

	log.Fatalf("Not implemented %s", params)
}

// GetDomainEndpoint returns info for specified domain
func GetDomainEndpoint(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)

	log.Fatalf("Not implemented %s", params)
}

// GetNewCertificatesEndpoint returns all new certificates found
func GetNewCertificatesEndpoint(w http.ResponseWriter, req *http.Request) {
	// TODO define "new"?
	params := mux.Vars(req)

	log.Fatalf("Not implemented %s", params)
}

// RemoveDomainEndpoint removes specified domain
func RemoveDomainEndpoint(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	log.Fatalf("Not implemented %s", params)
}

// RestAPIInit initializes rest api with a channel to signal changes back to the controller
func RestAPIInit() {
	// TODO get handle to data in controller?

	log.Fatal("not implemented")
}

// RestAPIRun the rest api, handle connections, update host lists, etc.
func RestAPIRun() {
	router := mux.NewRouter()

	router.HandleFunc("/domains", GetDomainsEndpoint).Methods("GET")
	router.HandleFunc("/domains/{domain}", GetDomainEndpoint).Methods("GET")
	router.HandleFunc("/new_certificates", GetNewCertificatesEndpoint).Methods("GET")
	router.HandleFunc("/domais/{domain}", AddDomainEndpoint).Methods("POST")
	router.HandleFunc("/domains/{domain}", RemoveDomainEndpoint).Methods("DELETE")

	log.Fatal(http.ListenAndServe(":80", router))
}
