package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"flag"
	"io/ioutil"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/op/go-logging"
	//	"github.com/research/censys-definitions/zsearch_definitions"
	"github.com/zmap/zgrab/ztools/zct"
	"github.com/zmap/zgrab/ztools/zct/scanner"
	"github.com/zmap/zgrab/ztools/zct/x509"
)

type LogConfig struct {
	Name         string   `json:"name"`
	Url          string   `json:"url"`
	LastIndex    int64    `json:"index"`
	BucketSize   int64    `json:"window"`
	UpdatePeriod int64    `json:"limit"`
	MaximumIndex int64    `json:"stop"`
	HostNames    []string `json:"hostnames`
}

type Configuration []LogConfig

var exit bool
var roots *x509.CertPool
var log = logging.MustGetLogger("")

var hostnames map[string][]string

// Example format string. Everything except the message has a custom color
// which is dependent on the log level. Many fields have a custom output
// formatting too, eg. the time returns the hour down to the milli second.
var format = logging.MustStringFormatter(
	`%{color}%{time:15:04:05} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
)

// TODO instrument for Prometheus
// TODO interface for adding, removing, and showing hostnames and certs (REST?)
// TODO postgres

// TODO Pull into own package?
func (logs Configuration) WriteConfig(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, log := range logs {
		bytes, err := json.Marshal(log)
		if err != nil {
			return err
		}
		f.Write(bytes)
		f.WriteString("\n")
	}
	return nil
}

func NewConfiguration(filename string) (Configuration, error) {
	res := Configuration{}
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if scanner.Text() == "" {
			break
		}
		parsed := LogConfig{}
		json.Unmarshal([]byte(scanner.Text()), &parsed)
		res = append(res, parsed)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return res, nil
}

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
		// TODO submit to postgres
		log.Fatalf("Valid Cert! %s", domain)
	}
}

func foundCert(entry *ct.LogEntry, server string) {
	processCert(entry, entry.X509Cert, false, server)
}

func foundPrecert(entry *ct.LogEntry, server string) {
	precert := entry.Precert.TBSCertificate
	processCert(entry, &precert, false, server)
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

func initialize(rootFile, configFile, output string, logLevel int) {
	var f *os.File
	if output == "-" {
		f = os.Stderr
	} else {
		var err error
		f, err = os.OpenFile(output, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("error opening file: %v", err)
		}
	}
	backend := logging.NewLogBackend(f, "", 0)
	backendFormat := logging.NewBackendFormatter(backend, format)
	backendLeveled := logging.AddModuleLevel(backendFormat)
	backendLeveled.SetLevel(logging.Level(logLevel), "")
	logging.SetBackend(backendLeveled)
	log.Debugf("Input Log level: %d %s\n", logging.Level(logLevel), logging.Level(logLevel).String())
	log.Debugf("Log level: %d %s\n", backendLeveled.GetLevel(""), backendLeveled.GetLevel("").String())
	infile, _ := os.Open(rootFile)
	defer infile.Close()
	bytes, _ := ioutil.ReadAll(infile)
	roots = x509.NewCertPool()
	_ = roots.AppendCertsFromPEM(bytes)
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGKILL)
		sig := <-c
		if sig == syscall.SIGTERM || sig == syscall.SIGINT || sig == syscall.SIGKILL {
			log.Fatal("Received a signal:", sig, ". Shutting down.")
			for {
				f, err := os.Open(configFile)
				if err != nil || f != nil {
					f.Close()
					break
				}
			}
			f.Close()
		} else {
			log.Notice("Received a signal:", sig, ". Ignoring.")
		}
		os.Exit(1)
	}()
}

func main() {
	configFile := flag.String("config", "./config.json", "the configuration file for log servers")
	output := flag.String("log", "-", "log file")
	rootFile := flag.String("root", "/etc/nss-root-store.pem", "an nss root store, defaults to etc/nss-root-store.pem")
	numProcs := flag.Int("proc", 0, "Number of processes to run on")
	numFetch := flag.Int("fetcher", 1, "Number of workers assigned to fetch certificates from each server")
	numMatch := flag.Int("matcher", 1, "Number of workers assigned to parse certs from each server")
	logLevel := flag.Int("log-level", 0, "log level")
	ex := flag.Bool("exit", false, "Tells the program to exit once it has gotten the most recent certificates")
	flag.Parse()

	// change this to allow multithreading
	runtime.GOMAXPROCS(*numProcs)
	initialize(*rootFile, *configFile, *output, *logLevel)
	exit = *ex

	config, err := NewConfiguration(*configFile)

	hostnames = make(map[string][]string)

	for _, conf := range config {
		hostnames[conf.Name] = conf.HostNames
	}
	if err != nil {
		log.Fatalf("Configuration error: %s", err)
	}
	logUpdater := make(chan LogConfig)
	done := make(chan bool)
	finished := make(chan bool)
	counter := 0
	for _, logC := range config {
		go downloader(logC, logUpdater, done, *rootFile, *numFetch, *numMatch)
		counter++
	}
	go func() {
		for i := 0; i < counter; i++ {
			<-done
		}
	}()
	for {
		select {
		case <-finished:
			if exit {
				os.Exit(0)
			}
		case update := <-logUpdater:
			for i, conf := range config {
				if conf.Url == update.Url {
					config[i] = update
					err = config.WriteConfig(*configFile)
					if err != nil {
						log.Fatal("Config write err: ", err)
					}
				}
			}
		}
	}
}
