package env

import (
	"testing"
	"time"
)

func TestGetDelaySuccess(t *testing.T) {
	d, err := GetDelaySuccess()
	if err != nil {
		t.Errorf("got error %v", err)
	}
	expected := 10 * time.Minute
	if d != expected {
		t.Errorf("Expected default of %v, got %v", expected, d)
	}
}

func TestGetDelayError(t *testing.T) {
	d, err := GetDelayError()
	if err != nil {
		t.Errorf("got error %v", err)
	}
	expected := 5 * time.Minute
	if d != expected {
		t.Errorf("Expected default of %v, got %v", expected, d)
	}
}
