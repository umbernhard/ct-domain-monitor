package main

import (
	"bufio"
	"encoding/json"
	"os"
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
