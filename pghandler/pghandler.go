package postgres

import (
	"database/sql"
	// we rely on the sql library, but need the postgres driver
	_ "github.com/lib/pq"
)

var db *sql.DB

// Open  Establish a connection to the database
func Open(user, dbname string) error {

	var err error
	db, err = sql.Open("postgres", "user="+user+" dbname="+dbname+" sslmode=verify-full")
	if err != nil {
		return err
	}

	return nil
}

// Close Close the databse connection
func Close() error {
	return db.Close()

}

// Submit Insert a domain, cert pair to the db
func Submit(server, domain string, cert []byte) error {
	stmt, err := db.Prepare("INSERT INTO " + server + "(name) VALUES($1, $2)")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(domain, cert)
	if err != nil {
		return err
	}
	return nil
}
