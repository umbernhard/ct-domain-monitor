package main

import (
	"testing"
)

func TestTest(t *testing.T) {
	if 1 == 0 {
		t.Errorf("Something has gone horribly wrong")
	}
	return
}
