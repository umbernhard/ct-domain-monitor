package main

import (
	"encoding/json"
	"testing"
)

func TestGetDomainsEndpoint(t *testing.T) {

}

func TestGetDomainEndpoint(t *testing.T) {

}

func TestAddDomainEndpoint(t *testing.T) {

}
func TestGetNewCeritificatesEndpoint(t *testing.T) {

}
func TestRemoveDomainEndpoint(t *testing.T) {

}

func TestMain(m *testing.M) {

	RestAPIRun()

	code := m.Run()

	os.Exit(code)
}
