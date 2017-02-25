package postgres

import (
	"database/sql"
	_ "github.com/lib/pq"
)

var db *sql.DB

func Open(user, dbname string) error {

	var err error
	db, err = sql.Open("postgres", "user="+user+" dbname="+dbname+" sslmode=verify-full")
	if err != nil {
		return err
	}

	return nil
}

func Close() error {
	return db.Close()

}

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
