package models

import (
	"testing"

	stripe "github.com/stripe/stripe-go"
)

func TestDefaultSourceShouldNotCrash(t *testing.T) {
	card := DefaultSource(nil)
	if card != nil {
		t.Errorf("Expected nil card, got %+v", card)
	}
}

func TestDefaultSourceBlankUser(t *testing.T) {
	blankCustomer := stripe.Customer{}
	card := DefaultSource(&blankCustomer)
	if card != nil {
		t.Errorf("Expected nil card, got %+v", card)
	}
}
