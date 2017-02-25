package main

import (
	"errors"
	"os"

	"github.com/zmap/zgrab/ztools/zct"
	"github.com/zmap/zgrab/ztools/zct/client"
)

var (
	// ErrTreeHead if we can't get the tree head
	ErrTreeHead = errors.New("Error to get tree head")
	// ErrLogEntries if there's an issue getting log entries
	ErrLogEntries = errors.New("Error ...")
	// ErrCertificateNotFound if we cannot find a certificate
	ErrCertificateNotFound = errors.New("Error certificate not found")
)

// LogServerConnection Struct containing the CT log connection and relevant data
type LogServerConnection struct {
	logClient  *client.LogClient
	outputFile *os.File
	treeSize   int64
	bucketSize int64
	start      int64
	end        int64
}

func merkleTreeSize(logClient *client.LogClient) (uint64, error) {
	treeHead, err := logClient.GetSTH()
	if err != nil {
		return 0, err
	}
	return treeHead.TreeSize, nil
}

func leafCertificate(logEntry ct.LogEntry) ([]byte, error) {

	if logEntry.Leaf.LeafType != ct.TimestampedEntryLeafType {
		return nil, ErrCertificateNotFound
	}
	if logEntry.Leaf.TimestampedEntry.EntryType != ct.X509LogEntryType && logEntry.Leaf.TimestampedEntry.EntryType != ct.PrecertLogEntryType {
		return nil, ErrCertificateNotFound
	}

	if logEntry.Leaf.TimestampedEntry.EntryType == ct.X509LogEntryType {
		return logEntry.Leaf.TimestampedEntry.X509Entry, nil
	}
	return logEntry.Leaf.TimestampedEntry.PrecertEntry.TBSCertificate, nil
}

// New Create a new connection to server <uri>, downloading <bucketSize> entries at a time
func New(uri string, bucketSize int64) *LogServerConnection {
	var c LogServerConnection
	var err error
	c.logClient = client.New(uri)
	if err != nil {
		log.Error(err)
		return nil
	}
	treeSize, err := merkleTreeSize(c.logClient)
	if err != nil {
		log.Error(err)
		return nil
	}
	c.treeSize = int64(treeSize)
	//fmt.Fprintf(os.Stderr, "Connection with %s has size %d\n", uri, c.treeSize)
	if bucketSize >= c.treeSize {
		c.bucketSize = c.treeSize / 2
	} else {
		c.bucketSize = bucketSize
	}
	c.start = 0
	c.end = c.bucketSize
	return &c
}

// NewWithOffset Same as New, but starts at entry <offset> in the log
func NewWithOffset(uri string, bucketSize int64, start int64) *LogServerConnection {
	c := New(uri, bucketSize)

	if c == nil {
		return nil
	}
	c.start = start
	c.end = start + bucketSize
	return c
}

func (c *LogServerConnection) slideBucket() {
	if c.start == 0 {
		c.start = 1
	}
	c.start += c.bucketSize
	c.end += c.bucketSize
}

// GetLogEntries get one window's worth of entries, slide window
func (c *LogServerConnection) GetLogEntries() ([]ct.LogEntry, error) {
	if c.end >= c.treeSize {
		c.treeSize -= 1
		c.end = c.treeSize
	}

	log.Info("Requesting Tree Range: %d-%d/%d\n", c.start, c.end, c.treeSize)
	entries, err := c.logClient.GetEntries(c.start, c.end)

	log.Info("Entries length: %d", len(entries))

	if err != nil {
		return nil, err
	}

	if len(entries) < int(c.bucketSize) && c.end != c.treeSize {
		c.end = c.start + int64(len(entries))
		c.bucketSize = int64(len(entries))
	}

	return entries, nil
}
