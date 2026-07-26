package domain

import (
	"errors"
	"time"

	foundation "github.com/opencorex-org/openrevenue/pkg/domain"
	"github.com/opencorex-org/openrevenue/pkg/id"
)

type EventTag struct{}
type EventID = id.ID[EventTag]
type Event struct {
	ID            EventID           `json:"id"`
	TenantID      string            `json:"tenantId"`
	Jurisdiction  string            `json:"jurisdiction"`
	Action        string            `json:"action"`
	Actor         string            `json:"actor"`
	ActorType     string            `json:"actorType"`
	ResourceType  string            `json:"resourceType"`
	ResourceID    string            `json:"resourceId"`
	OccurredAt    time.Time         `json:"occurredAt"`
	CorrelationID string            `json:"correlationId"`
	CausationID   string            `json:"causationId"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

func New(scope foundation.Context, action, resourceType, resourceID string, now time.Time) (Event, error) {
	if err := scope.Validate(); err != nil {
		return Event{}, err
	}
	if action == "" || resourceType == "" || resourceID == "" {
		return Event{}, errors.New("audit action, resource type, and resource id are required")
	}
	if now.IsZero() {
		return Event{}, errors.New("audit occurrence time is required")
	}
	return Event{
		ID: id.New[EventTag](), TenantID: scope.Tenant().String(),
		Jurisdiction: scope.Jurisdiction().String(), Action: action,
		Actor: scope.Actor().ID(), ActorType: string(scope.Actor().Kind()),
		ResourceType: resourceType, ResourceID: resourceID,
		CorrelationID: scope.CorrelationID().String(),
		CausationID:   scope.CausationID().String(),
		OccurredAt:    now.UTC(),
	}, nil
}
