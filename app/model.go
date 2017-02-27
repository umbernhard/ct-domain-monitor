// model.go

package main

import (
	"database/sql"
	"errors"
)

type record struct {
	Domain string `json:"name"`
	Cert   string `json:"cert"`
}

func (r *record) getDomain(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT domain, cert_pem FROM domains WHERE domain=$1", r.Domain)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	certs := make([]string, 0)

	for rows.Next() {
		var r record
		if err := rows.Scan(&r.Domain, &r.Cert); err != nil {
			return nil, err
		}
		certs = append(certs, r.Cert)
	}

	return certs, nil
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
