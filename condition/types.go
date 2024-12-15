package condition

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/metal-automata/rivets/events"
	"golang.org/x/exp/slices"

	fleetdbapi "github.com/metal-automata/fleetdb/pkg/api/v1"
)

// Kind holds the value for the Condition Kind field.
type Kind string

type Server = fleetdbapi.Server
type Component = fleetdbapi.ServerComponent
type InstalledFirmware = fleetdbapi.InstalledFirmware
type ComponentSlugMap = fleetdbapi.ComponentSlugMap
type BMC = fleetdbapi.ServerBMC

const (
	ServerResourceType    string = "servers"
	ConditionResourceType string = "condition"

	ConditionCreateEvent events.EventType = "create"
	ConditionUpdateEvent events.EventType = "update"

	// ConditionStructVersion identifies the condition struct revision
	ConditionStructVersion string = "1.1"

	// StaleConditionThreshold is the period after which the Condition Orchestrator will reconcile the Condition.
	StaleThreshold = 2 * time.Hour
)

// State is the state value of a Condition
type State string

// Defines holds the value for the Condition State field.
const (
	Pending   State = "pending"
	Active    State = "active"
	Failed    State = "failed"
	Succeeded State = "succeeded"
)

// States returns available condition states.
func States() []State {
	return []State{
		Active,
		Pending,
		Failed,
		Succeeded,
	}
}

// Transition valid returns a bool value if the state transition is allowed.
func (current State) TransitionValid(next State) bool {
	switch {
	// Pending state can stay in Pending or transition to Active or Failed or Succeeded
	case current == Pending && slices.Contains([]State{Pending, Active, Failed, Succeeded}, next):
		return true
	// Active state can stay in Active or transition to Failed or Succeeded
	case current == Active && slices.Contains([]State{Active, Failed, Succeeded}, next):
		return true
	default:
		return false
	}
}

// StateValid validates the State.
func StateIsValid(s State) bool {
	return slices.Contains(States(), s)
}

// StateComplete returns true when the given state is considered to be final.
func StateIsComplete(s State) bool {
	return slices.Contains([]State{Failed, Succeeded}, s)
}

// Parameters is an interface for Condition Parameter types
type Parameters interface {
	Validate() error
}

// Definition holds the default parameters for a Condition.
type Definition struct {
	Kind                  Kind `mapstructure:"kind"`
	FailOnCheckpointError bool `mapstructure:"failOnCheckpointError"`
}

// Definitions is the list of conditions with helper methods.
type Definitions []*Definition

func (c Definitions) FindByKind(k Kind) *Definition {
	for _, e := range c {
		if e.Kind == k {
			return e
		}
	}

	return nil
}

// Condition defines model for Condition.
//
// nolint:govet // fieldalignment struct is easier to read in the current format
type Condition struct {
	// Version identifies the revision number for this struct.
	Version string `json:"version"`

	// Client is the user/jwt user that requested the condition.
	Client string `json:"client"`

	// TraceID enables tracking a Condition and any associated Conditions
	TraceID string `json:"traceID"`

	// SpanID enables tracking a Condition and any associated Conditions
	SpanID string `json:"spanID"`

	// ID is the identifier for this condition.
	ID uuid.UUID `json:"id"`

	// Target is the identifier for the target server this Condition is applicable for.
	Target uuid.UUID `json:"target"`

	// Kind is one of Kind.
	Kind Kind `json:"kind,omitempty"`

	// Parameters is a JSON object that is agreed upon by the controller
	// reconciling the condition and the client requesting the condition.
	Parameters json.RawMessage `json:"parameters,omitempty"`

	// State is one of State
	State State `json:"state,omitempty"`

	// Status is a JSON object that is agreed upon by the controller
	// reconciling the condition and the client requesting the condition.
	Status json.RawMessage `json:"status,omitempty"`

	// Should the worker executing this condition fail if its unable to checkpoint
	// the status of work on this condition.
	FailOnCheckpointError bool `json:"failOnCheckpointError,omitempty"`

	// Fault is used to introduce faults into the controller when executing on a condition.
	Fault *Fault `json:"fault,omitempty"`

	// UpdatedAt is when this object was last updated.
	UpdatedAt time.Time `json:"updatedAt,omitempty"`

	// CreatedAt is when this object was created.
	CreatedAt time.Time `json:"createdAt,omitempty"`
}

// Returns bool for when a Condition should be published to the Jetstream.
func (c *Condition) StreamPublishRequired() bool {
	switch c.Kind {
	// The FirmwareInstallInband condition is handled by inband controllers,
	// these inband controllers fetch pending Conditions through the Orchestrator API,
	// which gets it from the Active-Condition records KV.
	case FirmwareInstallInband:
		return false
	default:
		return true
	}
}

func (c *Condition) StreamPublishSubject(facilityCode string) string {
	return fmt.Sprintf("%s.servers.%s", facilityCode, c.Kind)
}

// Fault is used to introduce faults into the controller when executing on a condition.
//
// Note: this depends on controllers implementing support to honor the given fault.
//
// nolint:govet // fieldalignment struct is easier to read in the current format
type Fault struct {
	//  will cause the condition execution to panic on the controller.
	Panic bool `json:"panic"`

	// Introduce specified delay in execution of the condition on the controller.
	//
	// accepts the string format of time.Duration - 5s, 5m, 5h
	DelayDuration string `json:"delayDuration,omitempty"`

	// FailAt is a controller specific task/stage that the condition should fail in execution.
	//
	// for example, in the flasher controller, setting this field to `init` will cause the
	// condition task to fail at initialization.
	FailAt string `json:"failAt,omitempty"`
}

// MustBytes returns an encoded json representation of the condition or panics
func (c *Condition) MustBytes() []byte {
	byt, err := json.Marshal(c)
	if err != nil {
		panic("encoding condition failed: " + err.Error())
	}

	return byt
}

// StateValid validates the Condition State field.
func (c *Condition) StateValid() bool {
	return StateIsValid(c.State)
}

// IsComplete returns true if the condition has a state that is final.
func (c *Condition) IsComplete() bool {
	return StateIsComplete(c.State)
}

// ServerConditions is a type to hold a server ID and the conditions associated with it.
type ServerConditions struct {
	Conditions []*Condition
	ServerID   uuid.UUID
}
