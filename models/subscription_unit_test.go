package models

import (
	"testing"
)

func TestDefaultSourceShouldNotCrash(t *testing.T) {
	card := DefaultSource(nil)
	if card != nil {
		t.Errorf("Expected nil card, got %+v", card)
	}
}
