package main

import (
	"flag"
	"io/ioutil"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	_ "github.com/lib/pq"
	"github.com/op/go-logging"
	"github.com/zmap/zgrab/ztools/zct/x509"
)

var exit bool
var roots *x509.CertPool
var log = logging.MustGetLogger("")

var hostnames map[string][]string

// Buffer for newly added domains so we can catch up
var newHostNames map[string][]string

var monitor Monitor

// Example format string. Everything except the message has a custom color
// which is dependent on the log level. Many fields have a custom output
// formatting too, eg. the time returns the hour down to the milli second.
var format = logging.MustStringFormatter(
	`%{color}%{time:15:04:05} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
)

// TODO instrument for Prometheus

func initialize(rootFile, configFile, output string, logLevel int) {

	hostnames = make(map[string][]string)
	newHostNames = make(map[string][]string)
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

	user := flag.String("user", "monitor", "User for postgres DB")
	password := flag.String("password", "", "Password for pq")
	dbname := flag.String("dbname", "ctdomainmonitor", "Name of pq database")

	monitor = Monitor{}
	monitor.Initialize(*user, *password, *dbname)
	log.Debugf("Initialized monitor, db %v", monitor.DB)

	go monitor.Run(":8080")
	// change this to allow multithreading
	runtime.GOMAXPROCS(*numProcs)
	initialize(*rootFile, *configFile, *output, *logLevel)
	exit = *ex

	config, err := NewConfiguration(*configFile)

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
