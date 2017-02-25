package main

import (
	"bufio"
	"encoding/json"
	"os"
)

// LogConfig struct mirroring the config file schema
type LogConfig struct {
	Name         string   `json:"name"`
	Url          string   `json:"url"`
	LastIndex    int64    `json:"index"`
	BucketSize   int64    `json:"window"`
	UpdatePeriod int64    `json:"limit"`
	MaximumIndex int64    `json:"stop"`
	HostNames    []string `json:"hostnames"`
}

// Configuration "configuration", list of configs for each log we pull from
type Configuration []LogConfig

// WriteConfig dump the configuration objects to the relevant file
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

// NewConfiguration Create a new configuration object from a given file
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
