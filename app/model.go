// model.go

package main

import (
	"database/sql"
	"errors"
)

type record struct {
	Domain string `json:"name"`
	Certs  []byte `json:"certs"`
}

func (r *record) getDomain(db *sql.DB) error {
	return errors.New("Not implemented")
}

func (r *record) updateDomain(db *sql.DB) error {
	return errors.New("Not implemented")
}

func (r *record) deleteDomain(db *sql.DB) error {
	return errors.New("Not implemented")
}

func (r *record) createDomain(db *sql.DB) error {
	return errors.New("Not implemented")
}

func getDomains(db *sql.DB) ([]record, error) {
	return nil, errors.New("Not implemented")
}

func getNewCerts(db *sql.DB) ([]record, error) {
	return nil, errors.New("Not implemented")
}
