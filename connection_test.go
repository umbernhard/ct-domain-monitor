package main

import (
	"testing"
)

var (
	testtube = "https://ct.googleapis.com/testtube"
)

func testLSC(lSC *LogServerConnection, bucketSize int64, offset int64, t *testing.T) {

	if lSC != nil {
		if lSC.start != offset || lSC.end != bucketSize+offset {
			t.Errorf("Bucket indices are incorrect")
			return
		}

	} else {
		t.Errorf("Couldn't get a new server connection")
		return
	}

	return

}

func TestNew(t *testing.T) {
	uri := testtube
	bucketSize := int64(1)

	lSC := New(uri, bucketSize)

	testLSC(lSC, bucketSize, int64(0), t)

}

func TestNewWithOffset0(t *testing.T) {
	uri := testtube
	bucketSize := int64(1)
	offset := int64(0)

	lSC := NewWithOffset(uri, bucketSize, offset)

	testLSC(lSC, bucketSize, offset, t)

}

func TestNewWithOffset1(t *testing.T) {
	bucketSize := int64(1)
	offset := int64(1)

	lSC := NewWithOffset(testtube, bucketSize, offset)

	testLSC(lSC, bucketSize, offset, t)

}

func TestSlideBucket(t *testing.T) {
	bucketSize := int64(10)

	lSC := New(testtube, bucketSize)

	lSC.slideBucket()

	if lSC.start != bucketSize+1 || lSC.end != bucketSize*2 {
		t.Errorf("Couldn't slide bucket %d, %d", lSC.start, lSC.end)
		return
	}
}

func TestGetLogEntries(t *testing.T) {
	bucketSize := int64(10)

	lSC := New(testtube, bucketSize)

	entries, err := lSC.GetLogEntries()

	if err != nil {
		t.Errorf("Couldn't get log entries!")
		return
	}

	if int64(len(entries)) != bucketSize+1 {
		t.Errorf("Did not get the right number of entries: %d", len(entries))
		return
	}
}
