// main_test.go

package main_test

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/umbernhard/ct-domain-monitor"
)

var a main.Monitor

func ensureTableExists() {
	if _, err := a.DB.Exec(tableCreationQuery); err != nil {
		log.Fatal(err)
	}
}

func clearTable() {
	a.DB.Exec("DELETE FROM domains")
}

func executeRequest(req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	a.Router.ServeHTTP(rr, req)

	return rr
}

func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}

func addRecords(count int) {
	if count < 1 {
		count = 1
	}

	for i := 1; i <= count; i++ {
		a.DB.Exec("INSERT INTO domains(domain, cert_pem) VALUES($1, $2)", strconv.Itoa(i)+".com", testPEM)
	}
}

func TestEmptyTable(t *testing.T) {
	clearTable()

	req, _ := http.NewRequest("GET", "/domains", nil)
	response := executeRequest(req)

	checkResponseCode(t, http.StatusOK, response.Code)

	if body := response.Body.String(); body != "[]" {
		t.Errorf("Expected an empty array. Got %s", body)
	}
}

func TestGetNonExistentRecord(t *testing.T) {
	clearTable()

	req, _ := http.NewRequest("GET", "/domain/0x21.org", nil)
	response := executeRequest(req)

	checkResponseCode(t, http.StatusNotFound, response.Code)

	var m map[string]string
	json.Unmarshal(response.Body.Bytes(), &m)
	if m["error"] != "Domain not found" {
		t.Errorf("Expected the 'error' key of the response to be set to 'Domain not found'. Got '%s'", m["error"])
	}
}

func TestAddRecord(t *testing.T) {
	clearTable()

	payload := []byte(`{"domain":"test.com","server":"testtube"}`)

	req, _ := http.NewRequest("POST", "/domain", bytes.NewBuffer(payload))
	response := executeRequest(req)

	checkResponseCode(t, http.StatusCreated, response.Code)

	var m map[string]interface{}
	json.Unmarshal(response.Body.Bytes(), &m)

	if m["domain"] != "test.com" {
		t.Errorf("Expected domain to be 'test.com'. Got '%v'", m["domain"])
	}

	if m["server"] != "testtube" {
		t.Errorf("Expected product pem to be 'testtube'. Got '%v'", m["server"])
	}

}

func TestGetRecord(t *testing.T) {
	clearTable()
	addRecords(1)

	req, _ := http.NewRequest("GET", "/domain/1.com", nil)
	response := executeRequest(req)

	checkResponseCode(t, http.StatusOK, response.Code)
}

func TestDeleteRecord(t *testing.T) {
	clearTable()
	addRecords(1)

	req, _ := http.NewRequest("GET", "/domain/1.com", nil)
	response := executeRequest(req)
	checkResponseCode(t, http.StatusOK, response.Code)

	req, _ = http.NewRequest("DELETE", "/domain/1.com", nil)
	response = executeRequest(req)

	checkResponseCode(t, http.StatusOK, response.Code)

	req, _ = http.NewRequest("GET", "/domain/1.com", nil)
	response = executeRequest(req)
	checkResponseCode(t, http.StatusNotFound, response.Code)
}

func TestGetDomains(t *testing.T) {
	clearTable()
	addRecords(5)

	req, _ := http.NewRequest("GET", "/domains", nil)
	response := executeRequest(req)
	checkResponseCode(t, http.StatusOK, response.Code)

	//var m map[string]interface{}
	var m []string
	json.Unmarshal(response.Body.Bytes(), &m)

	//	data, ok := m["domains"].([]string)
	//	if !ok || data == nil {
	//		t.Errorf("Received garbled data: %v", m["domains"])
	//		return
	//	}
	//
	//	domains := []string(m["domains"].([]string))

	if len(m) != 5 {
		t.Errorf("Expected 5 values back, got %v", len(m))
	}
	for i := 0; i < 5; i++ {
		if m[i] != strconv.Itoa(i+1)+".com" {
			t.Errorf("Expected %v at index %d, got %v", strconv.Itoa(i+1)+".com", i, m[i])
		}
	}
}

func TestGetNewCerts(t *testing.T) {
	clearTable()
	addRecords(5)

	req, _ := http.NewRequest("GET", "/new_certificates", nil)
	response := executeRequest(req)
	checkResponseCode(t, http.StatusOK, response.Code)

	var m []map[string]interface{}
	json.Unmarshal(response.Body.Bytes(), &m)

	max := len(m)
	for i, record := range m {
		if record["domain"] != strconv.Itoa(max-i)+".com" {
			t.Errorf("Expected %v at index %d, got %v", strconv.Itoa(max-i)+".com", i, record["domain"])
		}

		if record["cert"] != testPEM {
			t.Errorf("Expected %v at index %d, got %v", testPEM, i, record["cert"])
		}
	}
}

func TestMain(m *testing.M) {

	a = main.Monitor{}
	a.Initialize(
		os.Getenv("TEST_DB_USERNAME"),
		os.Getenv("TEST_DB_PASSWORD"),
		os.Getenv("TEST_DB_NAME"))

	ensureTableExists()

	code := m.Run()

	clearTable()

	os.Exit(code)
}

const tableCreationQuery = `CREATE TABLE IF NOT EXISTS domains
(
	domain varchar (253) NOT NULL,
	cert_pem varchar NOT NULL,
	created_at timestamp NOT NULL DEFAULT(clock_timestamp())
)`

const testPEM = `-----BEGIN CERTIFICATE-----
MIIFWTCCBEGgAwIBAgIQKcAeK/vRrCIRYJslHLtUXjANBgkqhkiG9w0BAQsFADCB
kDELMAkGA1UEBhMCR0IxGzAZBgNVBAgTEkdyZWF0ZXIgTWFuY2hlc3RlcjEQMA4G
A1UEBxMHU2FsZm9yZDEaMBgGA1UEChMRQ09NT0RPIENBIExpbWl0ZWQxNjA0BgNV
BAMTLUNPTU9ETyBSU0EgRG9tYWluIFZhbGlkYXRpb24gU2VjdXJlIFNlcnZlciBD
QTAeFw0xNjA5MjEwMDAwMDBaFw0xNzA5MjEyMzU5NTlaMHUxITAfBgNVBAsTGERv
bWFpbiBDb250cm9sIFZhbGlkYXRlZDEpMCcGA1UECxMgSG9zdGVkIGJ5IEhPU1RJ
TkcgU0VSVklDRVMsIElOQy4xFDASBgNVBAsTC1Bvc2l0aXZlU1NMMQ8wDQYDVQQD
EwZ3d3cuc2IwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQChpclY5XTZ
dXUYUqL4nyztzYaLrsLGnUchfNtmP7y74hAVSTW2Vm755IjDvdw+4NK3JUlnWhh7
lzfapK+ABc97HQb1PNqWuai+p4vcimloqLJYYzIVGUwKnv+bmMudB5DilGWkhqca
k8v4Ro5Wsddd5RwxHl0A/sKRCiNnxhnQ+uHAzuahqw491mKruIDQc5t/lMAt1Sm/
OO2cb+EWolErsM9y41cnsG432geVNJh0IUlOzTSo6REO9b7YfAdEFSMdgim5oVRH
PqeVXnwHhTasYD0KLLaSO+QEMvXjFxk+JyvG8xkHCqeKGtoEW1yG1QKDFhfVDzRp
AOyZIDU02B8rAgMBAAGjggHHMIIBwzAfBgNVHSMEGDAWgBSQr2o6lFoL2JDqElZz
30O0Oija5zAdBgNVHQ4EFgQUW0j6ltmsCpXeKOISvLz6QToAhUQwDgYDVR0PAQH/
BAQDAgWgMAwGA1UdEwEB/wQCMAAwHQYDVR0lBBYwFAYIKwYBBQUHAwEGCCsGAQUF
BwMCME8GA1UdIARIMEYwOgYLKwYBBAGyMQECAgcwKzApBggrBgEFBQcCARYdaHR0
cHM6Ly9zZWN1cmUuY29tb2RvLmNvbS9DUFMwCAYGZ4EMAQIBMFQGA1UdHwRNMEsw
SaBHoEWGQ2h0dHA6Ly9jcmwuY29tb2RvY2EuY29tL0NPTU9ET1JTQURvbWFpblZh
bGlkYXRpb25TZWN1cmVTZXJ2ZXJDQS5jcmwwgYUGCCsGAQUFBwEBBHkwdzBPBggr
BgEFBQcwAoZDaHR0cDovL2NydC5jb21vZG9jYS5jb20vQ09NT0RPUlNBRG9tYWlu
VmFsaWRhdGlvblNlY3VyZVNlcnZlckNBLmNydDAkBggrBgEFBQcwAYYYaHR0cDov
L29jc3AuY29tb2RvY2EuY29tMBUGA1UdEQQOMAyCBnd3dy5zYoICc2IwDQYJKoZI
hvcNAQELBQADggEBACj7cCdFWw0Rg6fnTbRlnbyuoEAuT4YnFNTw5H2YEtyVCLLL
tZL2QlWPCsGkwyIrsX3PqYEXV95IMa9WQ0Vg+t5m+xFU9T3t9Uv4GWwyYGu9IvtS
5nwf8yfDQjqOMKGV7tHyd79VJq9QIy0UjpRxVNDv7fDEg99G9JB9IlGgWgUIOYXi
q6d428UL3qHF+uSABYR2sqajou0dJzUJD0UiP0vNLhbApwiMBFis/XTPtta/Ky+K
cXOEgc0BsY/gZw6TCiBLUP6DQcPGH6dRyzTY/1O676G0xr8171+ZAP+ZJb/W8xqZ
/YUAnq2guwMhSScugjWOG/9H2ITkiWhVs6iUFjU=
-----END CERTIFICATE-----`

const testPEM2 = `-----BEGIN CERTIFICATE-----
MIIE0TCCA7mgAwIBAgIQEYrlOdfgRh2aYUT6rx05zjANBgkqhkiG9w0BAQsFADBm
MQswCQYDVQQGEwJVUzEWMBQGA1UEChMNR2VvVHJ1c3QgSW5jLjEdMBsGA1UECxMU
RG9tYWluIFZhbGlkYXRlZCBTU0wxIDAeBgNVBAMTF0dlb1RydXN0IERWIFNTTCBD
QSAtIEczMB4XDTE2MDcxMjAwMDAwMFoXDTE3MDcxMjIzNTk1OVowGDEWMBQGA1UE
AwwNbWJlcm5oYXJkLmNvbTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEB
ALfc8xXkK62hC64Y96oYQ4e81RfwMi/CE7iRg+/01XolkhgSbyq1yAUKNnsREr57
L/OcTH4wQTR+1rZHBcCDfRSTj6zeL+Feq73hAojtXPdOTU6UjnnaEm4LM7PQaEK0
Rzm1qE+YAhzrySI0E8xMV8YK5ZDyrRqqx7xfZow9Q9WLL3VVklHkDZGI/hVgOWb4
zA0pVhS1R3aBIOZ8fd8+vcvK9+ZD1UsBSel+0I9wN/d3CcdXMtM7qdgkkf62bRd1
t2TXV4Yk+xnOT9+OkW8QlOoyAOEyvRJVHcarh48nff7PSb4cnZ62KSkOZY+2o024
Rkarp4jCvxwiwgHGFijcGscCAwEAAaOCAccwggHDMCsGA1UdEQQkMCKCDW1iZXJu
aGFyZC5jb22CEXd3dy5tYmVybmhhcmQuY29tMAkGA1UdEwQCMAAwKwYDVR0fBCQw
IjAgoB6gHIYaaHR0cDovL2d0LnN5bWNiLmNvbS9ndC5jcmwwgZ0GA1UdIASBlTCB
kjCBjwYGZ4EMAQIBMIGEMD8GCCsGAQUFBwIBFjNodHRwczovL3d3dy5nZW90cnVz
dC5jb20vcmVzb3VyY2VzL3JlcG9zaXRvcnkvbGVnYWwwQQYIKwYBBQUHAgIwNQwz
aHR0cHM6Ly93d3cuZ2VvdHJ1c3QuY29tL3Jlc291cmNlcy9yZXBvc2l0b3J5L2xl
Z2FsMB8GA1UdIwQYMBaAFK1lIoWQ0DvjoUmLN/nxCx1fF6B3MA4GA1UdDwEB/wQE
AwIFoDAdBgNVHSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwVwYIKwYBBQUHAQEE
SzBJMB8GCCsGAQUFBzABhhNodHRwOi8vZ3Quc3ltY2QuY29tMCYGCCsGAQUFBzAC
hhpodHRwOi8vZ3Quc3ltY2IuY29tL2d0LmNydDATBgorBgEEAdZ5AgQDAQH/BAIF
ADANBgkqhkiG9w0BAQsFAAOCAQEAahA3HgAChFEDpgWtLUVa2i/QXnplo4ngDoKH
RsLo/Muxf2Fiwt8QQpmOKgLGvwr2MbQsEIDxSPJbfa+jhKKPmZilZ/h1dDLv0uMh
a3CON+OcbbcdbSFaganIPhX/NDkSHV3CY+6knPr92cWKyctjzpIRo09tbgUafM1y
L3X9kYHPP5/fAADvOJxY0bKZgaMLlsj9jOxsulumCDWydEzQictDHrxSZfopwOGH
QkXH6oOUW91gpeJQe559HkKW/rl0NngVceGIwW2UKDLn6Mj7eHFVA1BJ1tmlBV0g
tdTar8cRF22+QI6u8SpWKaxYYDB7Ze5Q9sNCKVFvXdKjCYsGjg==
-----END CERTIFICATE-----`
