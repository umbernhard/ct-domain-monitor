package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/pem"
	"time"

	"github.com/zmap/zgrab/ztools/zct"
	"github.com/zmap/zgrab/ztools/zct/scanner"
	"github.com/zmap/zgrab/ztools/zct/x509"
)

func processCert(entry *ct.LogEntry, cert *x509.Certificate, precert bool, server string) {
	// TODO Do we care about the server?
	serverName := ""
	domain := ""
	if len(cert.DNSNames) > 0 {
		domain = cert.DNSNames[0]
	} else if len(cert.PermittedDNSDomains) > 0 {
		domain = cert.PermittedDNSDomains[0]
	}

	flag := false

	for _, hostname := range hostnames[server] {
		if domain == hostname {
			flag = true
		}
	}

	// If we don't care about this cert, forget about it
	if !flag {
		return
	}

	intermediates := x509.NewCertPool()
	for _, interBytes := range entry.Chain {
		if len(interBytes) < 0 {
			continue
		}
		tmp, err := x509.ParseCertificate(interBytes)
		if err != nil {
			log.Noticef("Err parsing chain for %s:%d: %s\n", serverName, entry.Index, err)
			switch err.(type) {
			case x509.UnhandledCriticalExtension:
				block := pem.Block{"TRUSTED CERTIFICATE", nil, interBytes}
				log.Debug(string(pem.EncodeToMemory(&block)))
			}
			continue
		}
		intermediates.AddCert(tmp)
		fpArr := sha256.Sum256(tmp.Raw)
		log.Debugf("Added intermediate: %s\n", hex.EncodeToString(fpArr[:]))
	}
	opts := x509.VerifyOptions{domain, intermediates, roots, time.Now(), false, []x509.ExtKeyUsage{}}
	chains, err := cert.Verify(opts)
	valid := false
	if err == nil && len(chains) > 0 {
		valid = true
		log.Debugf("Valid leaf chain for %s:%d\n", serverName, entry.Index)
	} else {
		if err == nil {
			log.Debugf("Invalid leaf chain for %s:%d: No chains found\n", serverName, entry.Index)
		} else {
			log.Debugf("Invalid leaf chain for %s:%d: %s\n", serverName, entry.Index, err.Error())
		}
	}

	log.Criticalf("Cert! %s", domain)
	// XOR valid and precert, since we only want valid certs and also precerts
	if valid != precert {
		log.Debugf("Adding cert %v", domain)
		block := pem.Block{"TRUSTED CERTIFICATE", nil, cert.Raw}
		cert_pem := string(pem.EncodeToMemory(&block))
		err := monitor.DB.QueryRow(
			"INSERT INTO domains(domain, cert_pem) VALUES($1, $2)", domain, cert_pem).Scan()

		if err != nil {
			log.Debugf("Could not cert to database: %s", err)
		}

	}
}

func foundCert(entry *ct.LogEntry, server string) {
	processCert(entry, entry.X509Cert, false, server)
}

func foundPrecert(entry *ct.LogEntry, server string) {
	precert := entry.Precert.TBSCertificate
	processCert(entry, &precert, true, server)
}

func downloader(logConf LogConfig, logUpdater chan LogConfig, done chan bool, rootFile string, numFetch, numMatch int) {
	for {
		log.Debug("Downloading ", logConf.Name)
		logServerConnection := NewWithOffset(logConf.Url, logConf.BucketSize, logConf.LastIndex)
		if logServerConnection == nil {
			return
		}
		scanOpts := scanner.ScannerOptions{
			Matcher:       &scanner.MatchAll{},
			PrecertOnly:   false,
			BatchSize:     logConf.BucketSize,
			NumWorkers:    numMatch,
			ParallelFetch: numFetch,
			StartIndex:    logConf.LastIndex,
			Quiet:         false,
			Name:          logConf.Name,
			MaximumIndex:  logConf.MaximumIndex,
		}
		s := scanner.NewScanner(logServerConnection.logClient, scanOpts, log)
		updater := make(chan int64)
		// Update the log file
		go func() {
			for {
				update := <-updater
				logConf.LastIndex = update
				logUpdater <- logConf
			}
		}()

		delta, err := s.Scan(foundCert, foundPrecert, updater)

		if err != nil {
			log.Notice("Scan failed ", err)
		} else {
			logConf.LastIndex = delta
		}
		log.Noticef("%s now at index %d", logConf.Name, logConf.LastIndex)
		logUpdater <- logConf
		time.Sleep(time.Minute * 5)
	}
}
