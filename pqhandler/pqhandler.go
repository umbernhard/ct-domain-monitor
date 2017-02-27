package postgres

import (
	"database/sql"
	// we rely on the sql library, but need the postgres driver
	_ "github.com/lib/pq"
)

var db *sql.DB

func ensureTableExists(tableCreationQuery string) error {
	if _, err := db.Exec(tableCreationQuery); err != nil {
		return err
	}
	return nil
}

// Open  Establish a connection to the database
func Open(user, dbname string) error {

	var err error
	// TODO Enable SSL?
	db, err = sql.Open("postgres", "user="+user+" dbname="+dbname+" sslmode=disable")
	if err != nil {
		return err
	}

	return nil
}

// Close Close the databse connection
func Close() error {
	return db.Close()

}

// queries for given domain, cert. Used for testing
func present(server, domain string, cert []byte) (bool, error) {
	rows, err := db.Query("select * from "+server+" where domain=$1 and cert_raw=$2", domain, cert)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	return rows.Next(), nil

}

// runs arbitrary query for testing purposes
func exec(query string) error {
	_, err := db.Exec(query)
	return err
}

// Submit Insert a domain, cert pair to the db table corresponding to CT log server
func Submit(server, domain string, cert []byte) error {
	stmt, err := db.Prepare("INSERT INTO " + server + " (domain,cert_raw) VALUES($1, $2)")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(domain, cert)
	if err != nil {
		return err
	}
	return nil
}

// Remove all instances of cert for the domain in the db table corresponding to CT log server
func RemoveCertFromDomain(server, domain string, cert []byte) error {
	stmt, err := db.Prepare("DELETE FROM " + server + " WHERE domain=$1 AND cert_raw=$2")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(domain, cert)
	if err != nil {
		return err
	}
	return nil
}

// Remove all certs for domain in the db table corresponding to CT log server
func RemoveDomain(server, domain string) error {
	stmt, err := db.Prepare("DELETE FROM " + server + " WHERE domain=$1")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(domain)
	if err != nil {
		return err
	}
	return nil
}
