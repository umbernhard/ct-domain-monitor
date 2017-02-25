package postgres

import (
	"testing"
)

func TestOpen(t *testing.T) {
	err := Open("test", "test")

	if err != nil {
		t.Errorf("Couldn't open DB: %s", err)
	}

	defer Close()

}

func TestRentrantClose(t *testing.T) {
	Close()
}

func TestSubmit(t *testing.T) {
	//	t.Errorf("Not implemented")
}
