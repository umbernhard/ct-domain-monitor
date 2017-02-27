// monitor.go

package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// Monitor contains Router and database
type Monitor struct {
	Router *mux.Router
	DB     *sql.DB
}

// Initialize the monitor
func (a *Monitor) Initialize(user, password, dbname string) {
	connectionString :=
		fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable ", user, password, dbname)

	var err error
	a.DB, err = sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatal(err)
	}

	a.Router = mux.NewRouter()
}

// Run the monitor
func (a *Monitor) Run(addr string) {}
