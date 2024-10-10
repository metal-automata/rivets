package condition

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/google/uuid"
	rtypes "github.com/metal-automata/rivets/types"
	"github.com/pkg/errors"
)

var (
	errInvalidTaskJSON = errors.New("invalid Task JSON")
	errTaskUpdate      = errors.New("invalid Task update")
)

const (
	TaskVersion1 = "1.0"
	// Task KV bucket name
	TaskKVRepositoryBucket = "tasks"
)

func TaskKVRepositoryKey(facilityCode string, conditionKind Kind, serverID string) string {
	return fmt.Sprintf("%s.%s.%s", facilityCode, conditionKind, serverID)
}

type Task[P, D any] struct {
	// StructVersion indicates the Task object version and is used to determine Task  compatibility.
	StructVersion string `json:"task_version"`

	// Task unique identifier, this is set to the Condition identifier.
	ID uuid.UUID `json:"id"`

	// Kind is the type of Condition this Task is derived from
	Kind Kind `json:"kind"`

	// state is the state of the install
	State State `json:"state"`

	// status holds informational data on the state
	Status StatusRecord `json:"status"`

	// Data holds Condition Task specific data
	Data D `json:"data,omitempty"`

	// Parameters holds Condition specific parameters for this task
	Parameters P `json:"parameters,omitempty"`

	// Fault is a field to inject failures into a flasher task execution,
	// this is set from the Condition only when the worker is run with fault-injection enabled.
	Fault *Fault `json:"fault,omitempty"`

	// FacilityCode identifies the facility this task is to be executed in.
	FacilityCode string `json:"facility_code"`

	// Server holds attributes about target server this task is for.
	Server *rtypes.Server `json:"server,omitempty"`

	// WorkerID is the identifier for the worker executing this task.
	WorkerID string `json:"worker_id,omitempty"`

	TraceID string `json:"traceID"`
	SpanID  string `json:"spanID"`

	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
}

// SetState implements the Task runner interface
func (t *Task[P, D]) SetState(state State) {
	t.State = state
}

func (t *Task[P, D]) Marshal() (json.RawMessage, error) {
	return json.Marshal(t)
}

// TaskFromMessage converts a Task json.RawMessage object into its Task equivalent
func TaskFromMessage(msg json.RawMessage) (*Task[any, any], error) {
	t := &Task[any, any]{}
	if err := json.Unmarshal(msg, t); err != nil {
		return nil, errors.Wrap(errInvalidTaskJSON, err.Error())
	}

	return t, nil
}

// Update validates and applies the given changes to the Task object
//
// nolint:gocyclo // field validation is cyclomatic
func (t *Task[P, D]) Update(update *Task[P, D]) error {
	if t.ID != uuid.Nil && t.ID != update.ID {
		return errors.Wrap(errTaskUpdate, "Task.ID mismatch")
	}

	if t.Kind != "" && t.Kind != update.Kind {
		return errors.Wrap(errTaskUpdate, "Task.Kind mismatch")
	}

	// condition is already in a completed state, no further updates to be published
	if StateIsComplete(t.State) {
		return errors.Wrap(errTaskUpdate, "current state is final: "+string(t.State))
	}

	// worker ID update
	if update.WorkerID != "" {
		t.WorkerID = update.WorkerID
	}

	// ts update
	t.UpdatedAt = time.Now()

	// update State
	if update.State != "" && update.State != t.State {
		t.State = update.State
	}

	// update data
	t.Data = update.Data

	// update status
	currTaskStatus, err := t.Status.Marshal()
	if err != nil {
		return errors.Wrap(errTaskUpdate, "current Task.Status marshal error: "+err.Error())
	}

	updateTaskStatus, err := update.Status.Marshal()
	if err != nil {
		return errors.Wrap(errTaskUpdate, "new Task.Status marshal error: "+err.Error())
	}

	svEqual, err := statusRecordBytesEqual(currTaskStatus, updateTaskStatus)
	if err != nil {
		return errors.Wrap(errTaskUpdate, err.Error())
	}

	// At minimum a valid Status JSON has 9 chars - `{"a": "b"}`
	if currTaskStatus != nil && !svEqual && len(updateTaskStatus) >= 9 {
		t.Status = update.Status
	}

	return nil
}

// svBytesEqual compares the JSON in the two json.RawMessage
//
// source: https://stackoverflow.com/questions/32408890/how-to-compare-two-json-requests
func statusRecordBytesEqual(curSV, newSV json.RawMessage) (bool, error) {
	var j, j2 interface{}
	if err := json.Unmarshal(curSV, &j); err != nil {
		return false, errors.Wrap(err, "current StatusValue unmarshal error")
	}

	if err := json.Unmarshal(newSV, &j2); err != nil {
		return false, errors.Wrap(err, "new StatusValue unmarshal error")
	}

	return reflect.DeepEqual(j2, j), nil
}

func NewTaskFromCondition(cond *Condition) *Task[any, any] {
	sr, err := StatusRecordFromMessage(cond.Status)
	if err != nil {
		srp := NewTaskStatusRecord("Task initialized")
		sr = &srp
	}

	return &Task[any, any]{
		StructVersion: TaskVersion1,
		ID:            cond.ID,
		Kind:          cond.Kind,
		State:         Pending,
		Server:        &rtypes.Server{ID: cond.Target.String()},
		Parameters:    cond.Parameters,
		Data:          json.RawMessage(`{"empty": true}`), // placeholder value
		Fault:         cond.Fault,
		Status:        *sr,
		TraceID:       cond.TraceID,
		SpanID:        cond.SpanID,
	}
}
