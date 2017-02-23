package main

import (
	"testing"
)

var (
	test_tube = "https://ct.googleapis.com/testtube"
)

func TestNew(t *testing.T) {
	uri := test_tube
	bucketSize := int64(1)

	lSC := New(uri, bucketSize)

	if lSC != nil {
		if lSC.start != 0 || lSC.end != bucketSize {
			t.Errorf("Bucket indices are incorrect")
			return
		}

	} else {
		t.Errorf("Couldn't get a new server connection")
		return
	}

	return
}
