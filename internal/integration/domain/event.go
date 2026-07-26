// Package domain defines the versioned integration event envelope.
package domain

import (
	"errors"
	"time"

	foundation "github.com/opencorex-org/openrevenue/pkg/domain"
	"github.com/opencorex-org/openrevenue/pkg/id"
)

type eventTag struct{}

type Actor struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type Event struct {
	EventID       id.ID[eventTag]   `json:"eventId"`
	EventType     string            `json:"eventType"`
	EventVersion  int               `json:"eventVersion"`
	OccurredAt    time.Time         `json:"occurredAt"`
	CorrelationID string            `json:"correlationId"`
	CausationID   string            `json:"causationId"`
	Actor         Actor             `json:"actor"`
	Tenant        string            `json:"tenant"`
	Jurisdiction  string            `json:"jurisdiction"`
	Data          map[string]string `json:"data"`
	Metadata      map[string]string `json:"metadata"`
}

func New(scope foundation.Context, eventType, aggregateType, aggregateID string, now time.Time, data map[string]string) (Event, error) {
	if err := scope.Validate(); err != nil {
		return Event{}, err
	}
	if eventType == "" || aggregateType == "" || aggregateID == "" || now.IsZero() {
		return Event{}, errors.New("event type, aggregate, id, and occurrence time are required")
	}
	return Event{
		EventID: id.New[eventTag](), EventType: eventType, EventVersion: 1,
		Tenant: scope.Tenant().String(), Jurisdiction: scope.Jurisdiction().String(),
		Actor:         Actor{ID: scope.Actor().ID(), Kind: string(scope.Actor().Kind())},
		CorrelationID: scope.CorrelationID().String(), CausationID: scope.CausationID().String(),
		OccurredAt: now.UTC(), Data: data,
		Metadata: map[string]string{"aggregateType": aggregateType, "aggregateId": aggregateID},
	}, nil
}
