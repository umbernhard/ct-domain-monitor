package postgres

import (
	"bytes"
	"encoding/pem"
	"github.com/zmap/zgrab/ztools/x509"
	"io"
	"os"
	"testing"
)

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
		t.Errorf("Couldn't submit certificiate: %s", err)
	}

	Close()
}
