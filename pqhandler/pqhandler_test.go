package postgres

import (
	"bytes"
	"encoding/pem"
	"fmt"
	"github.com/zmap/zgrab/ztools/x509"
	"io"
	"os"
	"testing"
)

const tableCreationQuery = `CREATE TABLE IF NOT EXISTS test
(
	domain varchar (253) NOT NULL,
	cert_raw bytea NOT NULL
)`

func TestOpen(t *testing.T) {
	err := Open("test", "test")

	if err != nil {
		t.Errorf("Couldn't open DB: %s", err)
	}

	defer Close()

}

func TestRentrantClose(t *testing.T) {
	err := Close()
	if err != nil {
		t.Errorf("Close failed: %s", err)
	}
}

func TestSubmit(t *testing.T) {
	err := Open("test", "test")
	//	t.Errorf("Not implemented")
	f, err := os.Open("test.pem")
	if err != nil {
		t.Errorf("Could not open specified certificate: %s", err)
	}
	buf := bytes.NewBuffer(nil)
	io.Copy(buf, f)

	p, _ := pem.Decode(buf.Bytes())
	if p == nil {
		t.Errorf("Unable to parse PEM file: %s", err)
	}
	x509Cert, err := x509.ParseCertificate(p.Bytes)
	if err != nil {
		t.Errorf("Unable to parse certificate: %s", err)
	}

	err = Submit("test", "test.com", x509Cert.Raw)

	if err != nil {
		t.Errorf("Couldn't submit certificate: %s", err)
	}

	present, err := present("test", "test.com", x509Cert.Raw)
	if err != nil {
		t.Errorf("Couldn't submit certificate: %s", err)
	}
	if !present {
		t.Errorf("Couldn't submit certificate, unknown reason")
	}

	Close()
}

func TestRemoveCertFromDomain(t *testing.T) {
	err := Open("test", "test")

	f, err := os.Open("test.pem")
	if err != nil {
		t.Errorf("Could not open specified certificate: %s", err)
	}
	buf := bytes.NewBuffer(nil)
	io.Copy(buf, f)

	p, _ := pem.Decode(buf.Bytes())
	if p == nil {
		t.Errorf("Unable to parse PEM file: %s", err)
	}
	x509Cert, err := x509.ParseCertificate(p.Bytes)
	if err != nil {
		t.Errorf("Unable to parse certificate: %s", err)
	}

	err = Submit("test", "test.com", x509Cert.Raw)

	if err != nil {
		t.Errorf("Couldn't submit certificate: %s", err)
	}

	err = RemoveCertFromDomain("test", "test.com", x509Cert.Raw)
	if err != nil {
		t.Errorf("Couldn't remove certificate: %s", err)
	}

	present, err := present("test", "test.com", x509Cert.Raw)
	if err != nil {
		t.Errorf("Couldn't remove certificate: %s", err)
	}
	if present {
		t.Errorf("Couldn't remove certificate, unknown reason")
	}

	Close()
}

func TestRemoveDomain(t *testing.T) {
	err := Open("test", "test")
	//	t.Errorf("Not implemented")
	f, err := os.Open("test.pem")
	if err != nil {
		t.Errorf("Could not open specified certificate: %s", err)
	}
	buf := bytes.NewBuffer(nil)
	io.Copy(buf, f)

	p, _ := pem.Decode(buf.Bytes())
	if p == nil {
		t.Errorf("Unable to parse PEM file: %s", err)
	}
	x509Cert, err := x509.ParseCertificate(p.Bytes)
	if err != nil {
		t.Errorf("Unable to parse certificate: %s", err)
	}

	err = Submit("test", "test.com", x509Cert.Raw)

	if err != nil {
		t.Errorf("Couldn't submit certificate: %s", err)
	}

	err = RemoveDomain("test", "test.com")
	if err != nil {
		t.Errorf("Couldn't remove certificate: %s", err)
	}

	present, err := present("test", "test.com", x509Cert.Raw)
	if err != nil {
		t.Errorf("Couldn't remove certificate: %s", err)
	}
	if present {
		t.Errorf("Couldn't remove certificate, unknown reason")
	}

	Close()
}

func TestMain(m *testing.M) {
	err := Open("test", "test")

	if err != nil {
		fmt.Printf("Couldn't open DB: %s", err)
		os.Exit(1)
	}

	defer Close()
	if err := ensureTableExists(tableCreationQuery); err != nil {
		fmt.Printf("Couldn't ensure table: %s", err)
		os.Exit(1)
	}

	code := m.Run()

	exec("DELETE FROM test")

	os.Exit(code)

}
