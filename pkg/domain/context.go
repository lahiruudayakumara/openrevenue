package domain

import (
	"regexp"
	"strings"
)

var (
	tenantPattern       = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)
	jurisdictionPattern = regexp.MustCompile(`^[A-Z]{2}(?:-[A-Z0-9]{1,3})?$`)
	actorIDPattern      = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9._:@/-]{0,126}[A-Za-z0-9])?$`)
	correlationPattern  = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9._:-]{0,126}[A-Za-z0-9])?$`)
)

type TenantID struct{ value string }

func NewTenantID(value string) (TenantID, error) {
	if value != strings.TrimSpace(value) || len(value) < 2 || len(value) > 63 || !tenantPattern.MatchString(value) {
		return TenantID{}, invalid("tenant", "must be a 2-63 character lowercase identifier")
	}
	return TenantID{value: value}, nil
}

func (v TenantID) String() string { return v.value }
func (v TenantID) Validate() error {
	_, err := NewTenantID(v.value)
	return err
}

type Jurisdiction struct{ code string }

func NewJurisdiction(code string) (Jurisdiction, error) {
	if code != strings.TrimSpace(code) || !jurisdictionPattern.MatchString(code) {
		return Jurisdiction{}, invalid("jurisdiction", "must be an uppercase country or country-subdivision code")
	}
	return Jurisdiction{code: code}, nil
}

func (v Jurisdiction) String() string { return v.code }
func (v Jurisdiction) Validate() error {
	_, err := NewJurisdiction(v.code)
	return err
}

type ActorKind string

const (
	ActorUser    ActorKind = "USER"
	ActorService ActorKind = "SERVICE"
	ActorSystem  ActorKind = "SYSTEM"
)

type Actor struct {
	kind ActorKind
	id   string
}

func NewActor(kind ActorKind, id string) (Actor, error) {
	switch kind {
	case ActorUser, ActorService, ActorSystem:
	default:
		return Actor{}, invalid("actor kind", "is unsupported")
	}
	if id != strings.TrimSpace(id) || !actorIDPattern.MatchString(id) {
		return Actor{}, invalid("actor id", "must be a safe 1-128 character identifier")
	}
	return Actor{kind: kind, id: id}, nil
}

func (a Actor) Kind() ActorKind { return a.kind }
func (a Actor) ID() string      { return a.id }
func (a Actor) Validate() error {
	_, err := NewActor(a.kind, a.id)
	return err
}

type CorrelationID struct{ value string }

func NewCorrelationID(value string) (CorrelationID, error) {
	if value != strings.TrimSpace(value) || !correlationPattern.MatchString(value) {
		return CorrelationID{}, invalid("correlation id", "must be a safe 1-128 character identifier")
	}
	return CorrelationID{value: value}, nil
}

func (v CorrelationID) String() string { return v.value }
func (v CorrelationID) Validate() error {
	_, err := NewCorrelationID(v.value)
	return err
}

// Context is mandatory for application operations and scopes all state access.
type Context struct {
	tenant       TenantID
	jurisdiction Jurisdiction
	actor        Actor
	correlation  CorrelationID
}

func NewContext(tenant TenantID, jurisdiction Jurisdiction, actor Actor, correlation CorrelationID) (Context, error) {
	c := Context{tenant: tenant, jurisdiction: jurisdiction, actor: actor, correlation: correlation}
	if err := c.Validate(); err != nil {
		return Context{}, err
	}
	return c, nil
}

func (c Context) Tenant() TenantID             { return c.tenant }
func (c Context) Jurisdiction() Jurisdiction   { return c.jurisdiction }
func (c Context) Actor() Actor                 { return c.actor }
func (c Context) CorrelationID() CorrelationID { return c.correlation }
func (c Context) Validate() error {
	if err := c.tenant.Validate(); err != nil {
		return err
	}
	if err := c.jurisdiction.Validate(); err != nil {
		return err
	}
	if err := c.actor.Validate(); err != nil {
		return err
	}
	return c.correlation.Validate()
}

// IsolationKey returns a process-local composite key. It must not be logged.
func (c Context) IsolationKey(resourceID string) string {
	return c.tenant.String() + "\x00" + c.jurisdiction.String() + "\x00" + resourceID
}
