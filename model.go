// model.go

package main

import (
	"database/sql"
	//	"errors"
)

type record struct {
	Domain   string `json:"domain"`
	Cert     string `json:"cert"`
	CTServer string `json:"server"`
	Created  string `json:"created_at"`
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

func (r *record) deleteDomain(db *sql.DB) error {
	_, err := db.Exec("DELETE FROM domains WHERE domain=$1", r.Domain)

	return err
}

func (r *record) createDomain(db *sql.DB) {
	log.Debugf("Creating domain %v", r.Domain)
	if newHostNames == nil {
		newHostNames = make(map[string][]string)
	}
	newHostNames[r.CTServer] = append(newHostNames[r.CTServer], r.Domain)

	if hostnames == nil {
		hostnames = make(map[string][]string)
	}
	hostnames[r.CTServer] = append(hostnames[r.CTServer], r.Domain)
}

func (r *record) getDomains(db *sql.DB) ([]string, error) {

	rows, err := db.Query("SELECT domain FROM domains")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	domains := make([]string, 0)

	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		domains = append(domains, d)
	}

	return domains, nil
}

func (r *record) getNewCerts(db *sql.DB) ([]record, error) {

	rows, err := db.Query("SELECT * FROM domains ORDER BY created_at DESC LIMIT 10")

	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := make([]record, 0)

	for rows.Next() {
		var r record
		if err := rows.Scan(&r.Domain, &r.Cert, &r.Created); err != nil {
			return nil, err
		}
		records = append(records, r)
	}

	return records, nil
}
