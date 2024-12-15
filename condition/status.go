package condition

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/metal-automata/rivets/events/registry"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
)

type ErrQueryStatus struct {
	err       error
	kvBucket  string
	lookupKey string
	workerID  string
}

func (e *ErrQueryStatus) Error() string {
	s := fmt.Sprintf("Condition status lookup error, bucket=%s", e.kvBucket)

	if e.lookupKey != "" {
		s += fmt.Sprintf(" lookupKey=%s", e.lookupKey)
	}

	if e.workerID != "" {
		s += fmt.Sprintf(" workerID=%s", e.workerID)
	}

	if e.err != nil {
		s += fmt.Sprintf(" err=%s", e.err.Error())
	}

	return s
}

func newErrQueryStatus(err error, kvBucket, lookupKey, workerID string) error {
	return &ErrQueryStatus{
		err:       err,
		kvBucket:  kvBucket,
		lookupKey: lookupKey,
		workerID:  workerID,
	}
}

const (
	StatusValueVersion int32 = 1
)

// StatusValue is the canonical structure for reporting status of an ongoing task
type StatusValue struct {
	CreatedAt  time.Time       `json:"created"`
	UpdatedAt  time.Time       `json:"updated"`
	WorkerID   string          `json:"worker"`
	Target     string          `json:"target"`
	TraceID    string          `json:"traceID"`
	SpanID     string          `json:"spanID"`
	State      string          `json:"state"`
	Status     json.RawMessage `json:"status"`
	MsgVersion int32           `json:"msgVersion"`
	// WorkSpec json.RawMessage `json:"spec"` XXX: for re-publish use-cases
}

// MustBytes sets the version field of the StatusValue so any callers don't have
// to deal with it. It will panic if we cannot serialize to JSON for some reason.
func (v *StatusValue) MustBytes() []byte {
	v.MsgVersion = StatusValueVersion
	byt, err := json.Marshal(v)
	if err != nil {
		panic("unable to serialize status value: " + err.Error())
	}
	return byt
}

func UnmarshalStatusValue(b []byte) (*StatusValue, error) {
	sv := &StatusValue{}
	if err := json.Unmarshal(b, &sv); err != nil {
		return nil, err
	}

	return sv, nil
}

// TaskState holds the value for the task state in the KV.
//
// note: this differs from the condition.State, can these be merged?
type TaskState int

const (
	NotStarted    TaskState = iota
	InProgress              // another worker has started it, is still around and updated recently
	Complete                // task is done
	Orphaned                // the worker that started this task doesn't exist anymore
	Indeterminate           // we got an error in the process of making the check
)

// CheckConditionInProgress returns the status of the task from the KV store
//
//nolint:gocyclo // status checks are cyclomatic
func CheckConditionInProgress(conditionID, facilityCode, kvBucket string, js nats.JetStreamContext) (TaskState, error) {
	handle, err := js.KeyValue(kvBucket)
	if err != nil {
		errKV := errors.Wrap(err, "bind to status KV bucket for condition lookup failed")
		return Indeterminate, newErrQueryStatus(errKV, kvBucket, "", "")
	}

	lookupKey := fmt.Sprintf("%s.%s", facilityCode, conditionID)
	entry, err := handle.Get(lookupKey)
	switch {
	case errors.Is(err, nats.ErrKeyNotFound):
		// This should be by far the most common path through this code.
		return NotStarted, nil

	case err == nil:
		break // we'll handle this outside the switch

	default:
		return Indeterminate, newErrQueryStatus(err, kvBucket, lookupKey, "")
	}

	// we have an status entry for this condition, is is complete?
	sv := StatusValue{}
	if errJSON := json.Unmarshal(entry.Value(), &sv); errJSON != nil {
		errJSON = errors.Wrap(errJSON, "unable to construct a sane status for condition")
		return Indeterminate, newErrQueryStatus(errJSON, kvBucket, lookupKey, "")
	}

	if StateIsComplete(State(sv.State)) {
		return Complete, nil
	}

	// is the worker handling this condition alive?
	controllerID, err := registry.ControllerIDFromString(sv.WorkerID)
	if err != nil {
		errWorker := errors.Wrap(err, "bad worker ID")
		return Indeterminate, newErrQueryStatus(errWorker, kvBucket, lookupKey, sv.WorkerID)
	}

	_, err = registry.LastContact(controllerID)
	switch {
	case errors.Is(err, nats.ErrKeyNotFound):
		// the data for this worker aged-out, it's no longer active
		// XXX: the most conservative thing to do here is to return
		// indeterminate but most times this will indicate that the
		// worker crashed/restarted and this task should be restarted.

		// We're going to restart this condition when we return from
		// this function. Use the KV handle we have to delete the
		// existing task key.
		if err = handle.Delete(lookupKey); err != nil {
			errWorker := errors.Wrap(err, "unable to delete existing condition status")

			return Indeterminate, newErrQueryStatus(errWorker, kvBucket, lookupKey, sv.WorkerID)
		}

		return Orphaned, nil

	case err == nil:
		return InProgress, nil

	default:
		errWorker := errors.Wrap(err, "error looking up worker last contact")
		return Indeterminate, newErrQueryStatus(errWorker, kvBucket, lookupKey, sv.WorkerID)
	}
}

func NewTaskStatusRecord(s string) StatusRecord {
	sr := StatusRecord{}
	if s == "" {
		return sr
	}

	sr.Append(s)

	return sr
}

var (
	ErrInvalidStatusRecord = errors.New("invalid status record")
)

// StatusRecord holds status information for a Condition
type StatusRecord struct {
	StatusMsgs []StatusMsg `json:"records"`
}

// StatusMsg is a single record within the StatusRecord
type StatusMsg struct {
	Timestamp time.Time `json:"ts,omitempty"`
	Msg       string    `json:"msg,omitempty"`
}

func (sr *StatusRecord) Append(s string) {
	if s == "" {
		return
	}

	for _, r := range sr.StatusMsgs {
		if r.Msg == s {
			return
		}
	}

	if len(sr.StatusMsgs) > 4 {
		sr.StatusMsgs = sr.StatusMsgs[1:]
	}

	n := StatusMsg{Timestamp: time.Now(), Msg: s}

	sr.StatusMsgs = append(sr.StatusMsgs, n)
}

func (sr *StatusRecord) Last() string {
	if len(sr.StatusMsgs) == 0 {
		return ""
	}

	return sr.StatusMsgs[len(sr.StatusMsgs)-1].Msg
}

func (sr *StatusRecord) Update(currentMsg, newMsg string) {
	for idx, r := range sr.StatusMsgs {
		if r.Msg == currentMsg {
			sr.StatusMsgs[idx].Msg = newMsg
		}
	}
}

func (sr *StatusRecord) MustMarshal() json.RawMessage {
	b, err := json.Marshal(sr)
	if err != nil {
		panic(err)
	}

	return b
}

func (sr *StatusRecord) Marshal() (json.RawMessage, error) {
	return json.Marshal(sr)
}

func StatusRecordFromMessage(m json.RawMessage) (*StatusRecord, error) {
	sr := &StatusRecord{}
	if err := json.Unmarshal(m, sr); err != nil {
		return nil, errors.Wrap(ErrInvalidStatusRecord, err.Error())
	}

	return sr, nil
}
