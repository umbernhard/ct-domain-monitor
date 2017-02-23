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
	"strings"
	"syscall"
	"time"

	"github.com/op/go-logging"
	"github.com/research/censys-ctsync/censys-queuer/protobufs/zsearch_definitions"
	"github.com/zmap/zgrab/ztools/zct"
	"github.com/zmap/zgrab/ztools/zct/scanner"
	"github.com/zmap/zgrab/ztools/zct/x509"
)

type LogConfig struct {
	Name         string `json:"name"`
	Url          string `json:"url"`
	LastIndex    int64  `json:"index"`
	BucketSize   int64  `json:"window"`
	UpdatePeriod int64  `json:"limit"`
	MaximumIndex int64  `json:"stop"`
}

type Configuration []LogConfig

var exit bool
var roots *x509.CertPool
var log = logging.MustGetLogger("")

// Example format string. Everything except the message has a custom color
// which is dependent on the log level. Many fields have a custom output
// formatting too, eg. the time returns the hour down to the milli second.
var format = logging.MustStringFormatter(
	`%{color}%{time:15:04:05} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
)

// TODO instrument for Prometheus
// TODO interface for adding, removing, and showing hostnames and certs (REST?)
// TODO postgres

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

// TODO merge with foundPrecert
func foundCert(entry *ct.LogEntry, server string) {
	var parent_fingerprint [32]byte
	if len(entry.Chain) > 0 {
		parent_fingerprint = sha256.Sum256(entry.Chain[0])
		log.Debug("----------------------------------")
		fpArr := sha256.Sum256(entry.X509Cert.Raw)
		log.Debug(hex.EncodeToString(fpArr[:]))
		for i, _ := range entry.Chain {
			fpArr = sha256.Sum256(entry.Chain[i])
			log.Debug(hex.EncodeToString(fpArr[:]))
		}
		log.Debug("----------------------------------")
	}
	serverName := zsearch.CTServer(zsearch.CTServer_value[server])
	if zsearch.CTServer_value[server] == zsearch.CTServer_value["CT_SERVER_RESERVED"] {
		log.Warning("Unkown CT server: ", server)
		return
	}
	domain := ""
	if len(entry.X509Cert.DNSNames) > 0 {
		domain = entry.X509Cert.DNSNames[0]
	} else if len(entry.X509Cert.PermittedDNSDomains) > 0 {
		domain = entry.X509Cert.PermittedDNSDomains[0]
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
	chains, err := entry.X509Cert.Verify(opts)
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
	// TODO replace with postgres
	//	err = censys.ImportCertificateCT(nil, entry.X509Cert.Raw, serverName, entry.Index, entry.Leaf.TimestampedEntry.Timestamp, parent_fingerprint[:], valid)
	//	if err != nil {
	//		log.Noticef("Err importing certificate for %s:%d: %s\n", serverName, entry.Index, err)
	//	} else {
	//		log.Infof("Success importing certificate for %s:%d\n", serverName, entry.Index)
	//	}

	// Submit all chain entries
	for i, interBytes := range entry.Chain {
		tmp, err := x509.ParseCertificate(interBytes)
		if err != nil {
			switch err.(type) {
			case x509.NonFatalErrors:
				log.Noticef("Non-Fatal Err parsing chain for %s:%d: %s\n", serverName, entry.Index, err)
			default:
				log.Noticef("Fatal Err parsing chain for %s:%d: %s\n", serverName, entry.Index, err)
				continue
			}
		}
		var parent_print [32]byte
		if i < len(entry.Chain)-1 {
			parent_print = sha256.Sum256(entry.Chain[i+1])
		} else {
			parent_print = sha256.Sum256(entry.Chain[i])
		}
		domain := ""
		if len(tmp.DNSNames) > 0 {
			domain = tmp.DNSNames[0]
		} else if len(tmp.PermittedDNSDomains) > 0 {
			domain = tmp.PermittedDNSDomains[0]
		}
		opts := x509.VerifyOptions{domain, intermediates, roots, time.Now(), false, []x509.ExtKeyUsage{}}
		chains, err := tmp.Verify(opts)
		chain_valid := false
		if err == nil && len(chains) > 0 {
			chain_valid = true
			log.Debugf("Valid chain for %s:%d\n", serverName, entry.Index)
		} else {
			log.Debugf("Invalid chain for %s:%d\n", serverName, entry.Index)
		}
		// TODO replace with postgres?
		//		err = censys.ImportCertificateCTChain(nil, tmp.Raw, serverName, entry.Index, entry.Leaf.TimestampedEntry.Timestamp, parent_print[:], chain_valid)
		//		if err != nil {
		//			log.Noticef("Err submitting chain for %s:%d: %s\n", serverName, entry.Index, err)
		//		} else {
		//			log.Infof("Success importing certificate for %s:%d\n", serverName, entry.Index)
		//		}
	}
}

// TODO Merge with foundCert
func foundPrecert(entry *ct.LogEntry, server string) {
	var parent_fingerprint [32]byte
	if len(entry.Chain) > 1 {
		parent_fingerprint = sha256.Sum256(entry.Chain[1])
	}
	serverName := zsearch.CTServer(zsearch.CTServer_value[server])
	if zsearch.CTServer_value[server] == zsearch.CTServer_value["CT_SERVER_RESERVED"] {
		log.Warning("Unkown CT server: ", server)
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
	}
	// TODO run against hostname list, output to postgres if found
	//	err := censys.ImportCertificateCT(nil, entry.Precert.Raw, serverName, entry.Index, entry.Leaf.TimestampedEntry.Timestamp, parent_fingerprint[:], false)
	//	if err != nil {
	//		log.Noticef("Err submitting chain for %s:%d: %s\n", serverName, entry.Index, err)
	//	} else {
	//		log.Infof("Success importing certificate for %s:%d\n", serverName, entry.Index)
	//	}

	// Submit all chain entries
	// TODO Do we care?
	for i, interBytes := range entry.Chain {
		tmp, err := x509.ParseCertificate(interBytes)
		if err != nil {
			switch err.(type) {
			case x509.NonFatalErrors:
				log.Noticef("Non-Fatal Err parsing chain for %s:%d: %s\n", serverName, entry.Index, err)
			default:
				log.Noticef("Fatal Err parsing chain for %s:%d: %s\n", serverName, entry.Index, err)
				continue
			}
		}
		var parent_print [32]byte
		if i < len(entry.Chain)-1 {
			parent_print = sha256.Sum256(entry.Chain[i+1])
		} else {
			parent_print = sha256.Sum256(entry.Chain[i])
		}
		domain := ""
		if len(tmp.DNSNames) > 0 {
			domain = tmp.DNSNames[0]
		} else if len(tmp.PermittedDNSDomains) > 0 {
			domain = tmp.PermittedDNSDomains[0]
		}
		opts := x509.VerifyOptions{domain, intermediates, roots, time.Now(), false, []x509.ExtKeyUsage{}}
		chains, err := tmp.Verify(opts)
		chain_valid := false
		if err == nil && len(chains) > 0 {
			chain_valid = true
		}
		err = censys.ImportCertificateCTChain(nil, tmp.Raw, serverName, entry.Index, entry.Leaf.TimestampedEntry.Timestamp, parent_print[:], chain_valid)
		if err != nil {
			log.Noticef("Err submitting chain for %s:%d: %s\n", serverName, entry.Index, err)
		} else {
			log.Infof("Success importing certificate for %s:%d\n", serverName, entry.Index)
		}
	}
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
	brokerString := flag.String("brokers", "127.0.0.1:9092", "a comma separated list of the kafka broker locations")
	queue := flag.String("topic", "ct_to_zdb", "the kafka topic to place certs in")
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
	brokers := strings.Split(*brokerString, ",")

	// TODO Ripout kafka, replace with postgres
	//	err := censys.ConnectToKafka(brokers, *queue)
	//	if err != nil {
	//		log.Fatalf("Kafka connection error: %s", err)
	//	}
	config, err := NewConfiguration(*configFile)

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
