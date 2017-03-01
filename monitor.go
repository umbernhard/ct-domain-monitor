// monitor.go

package main

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// Monitor contains Router and database
type Monitor struct {
	Router *mux.Router
	DB     *sql.DB
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func (a *Monitor) getDomain(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	domain := vars["domain"]

	p := record{Domain: domain}
	certs, err := p.getDomain(a.DB)

	if err != nil {
		switch err {
		case sql.ErrNoRows:
			respondWithError(w, http.StatusNotFound, "Domain not found")
		default:
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	} else if len(certs) < 1 {
		respondWithError(w, http.StatusNotFound, "Domain not found")
		return
	}

	respondWithJSON(w, http.StatusOK, certs)
}

func (a *Monitor) getDomains(w http.ResponseWriter, r *http.Request) {

	p := record{}
	d, err := p.getDomains(a.DB)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			respondWithError(w, http.StatusNotFound, "Domain not found")
		default:
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	respondWithJSON(w, http.StatusOK, d)
}

func (a *Monitor) createDomain(w http.ResponseWriter, r *http.Request) {
	var p record
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&p); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()

	p.createDomain(a.DB)

	respondWithJSON(w, http.StatusCreated, p)
}

func (a *Monitor) deleteDomain(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	domain := vars["domain"]

	p := record{Domain: domain}
	if err := p.deleteDomain(a.DB); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"result": "success"})
}

func (a *Monitor) getNewCerts(w http.ResponseWriter, r *http.Request) {

	p := record{}
	records, err := p.getNewCerts(a.DB)

	if err != nil {
		switch err {
		case sql.ErrNoRows:
			respondWithError(w, http.StatusNotFound, "Domain not found")
		default:
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	respondWithJSON(w, http.StatusOK, records)
}
func (a *Monitor) initializeRoutes() {
	a.Router.HandleFunc("/domains", a.getDomains).Methods("GET")
	a.Router.HandleFunc("/new_certificates", a.getNewCerts).Methods("GET")
	a.Router.HandleFunc("/domain", a.createDomain).Methods("POST")
	a.Router.HandleFunc("/domain/{domain:.+}", a.getDomain).Methods("GET")
	a.Router.HandleFunc("/domain/{domain:.+}", a.deleteDomain).Methods("DELETE")
	log.Debugf("Monitor: Initialized routes")
}

// Initialize the monitor
func (a *Monitor) Initialize(user, password, dbname string) {
	connectionString := "user='" + user + "' dbname='" + dbname + "' sslmode=disable"

	var err error
	a.DB, err = sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatalf("Monitor: Couldn't connect to the database: %v", err)
	}

	a.Router = mux.NewRouter()
	a.initializeRoutes()
}

// Run the monitor
func (a *Monitor) Run(addr string) {
	log.Fatal(http.ListenAndServe(addr, a.Router))
}
