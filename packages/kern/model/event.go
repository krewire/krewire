package model

import "time"

// DomainEvent is a cross-module domain event.
type DomainEvent struct {
	Type    string    `json:"type"`
	Payload any       `json:"payload"`
	At      time.Time `json:"at"`
}

// NewDomainEvent creates a DomainEvent with the current time.
func NewDomainEvent(typ string, payload any) DomainEvent {
	return DomainEvent{Type: typ, Payload: payload, At: time.Now().UTC()}
}
