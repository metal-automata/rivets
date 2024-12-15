package condition

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestNewTaskFromCondition(t *testing.T) {
	cond := &Condition{
		ID:         uuid.New(),
		Kind:       FirmwareInstall,
		Status:     []byte(`{"some":"status"}`),
		Target:     uuid.New(),
		Parameters: json.RawMessage(`{"param": "value"}`),
		Fault:      &Fault{Panic: true},
	}

	task := NewTaskFromCondition(cond)

	sr, err := StatusRecordFromMessage(cond.Status)
	assert.Nil(t, err)

	assert.NotNil(t, task, "task should not be nil")
	assert.Equal(t, TaskVersion1, task.StructVersion, "expected StructVersion to be %s, but got %s", TaskVersion1, task.StructVersion)
	assert.Equal(t, cond.ID, task.ID, "expected ID to be %s, but got %s", cond.ID, task.ID)
	assert.Equal(t, cond.Kind, task.Kind, "expected Kind to be %s, but got %s", cond.Kind, task.Kind)
	assert.Equal(t, Pending, task.State, "expected State to be %s, but got %s", Pending, task.State)
	assert.Equal(t, &Server{UUID: cond.Target}, task.Server, "expected Server to be %v, but got %v", &Server{UUID: cond.Target}, task.Server)
	assert.Equal(t, cond.Parameters, task.Parameters, "expected Parameters to be %v, but got %v", cond.Parameters, task.Parameters)
	assert.Equal(t, json.RawMessage(`{"empty": true}`), task.Data, "expected Data to be %v, but got %v", json.RawMessage(`{"empty": true}`), task.Data)
	assert.Equal(t, cond.Fault, task.Fault, "expected Fault to be %v, but got %v", cond.Fault, task.Fault)
	assert.Equal(t, sr, &task.Status, "expected Status to be %v, but got %v", sr, task.Status)
}

func TestStatusRecordBytesEqual(t *testing.T) {
	tests := []struct {
		name     string
		curSV    json.RawMessage
		newSV    json.RawMessage
		expected bool
		err      bool
	}{
		{
			name:     "equal JSON objects",
			curSV:    json.RawMessage(`{"key": "value"}`),
			newSV:    json.RawMessage(`{"key": "value"}`),
			expected: true,
			err:      false,
		},
		{
			name:     "unequal JSON objects",
			curSV:    json.RawMessage(`{"key": "value"}`),
			newSV:    json.RawMessage(`{"key": "differentValue"}`),
			expected: false,
			err:      false,
		},
		{
			name:     "malformed current JSON",
			curSV:    json.RawMessage(`{"key": "value"`),
			newSV:    json.RawMessage(`{"key": "value"}`),
			expected: false,
			err:      true,
		},
		{
			name:     "malformed new JSON",
			curSV:    json.RawMessage(`{"key": "value"}`),
			newSV:    json.RawMessage(`{"key": "value"`),
			expected: false,
			err:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			equal, err := statusRecordBytesEqual(tt.curSV, tt.newSV)
			if tt.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expected, equal)
		})
	}
}

func TestTaskUpdate(t *testing.T) {
	taskID := uuid.New()
	tests := []struct {
		name         string
		initialTask  Task[any, any]
		updateTask   Task[any, any]
		expectedTask Task[any, any]
		expectedErr  error
	}{
		{
			name: "Successful update",
			initialTask: Task[any, any]{
				ID:     taskID,
				Kind:   FirmwareInstall,
				State:  Pending,
				Status: StatusRecord{},
			},
			updateTask: Task[any, any]{
				ID:       taskID,
				Kind:     FirmwareInstall,
				State:    Active,
				WorkerID: "worker-123",
				Status:   StatusRecord{[]StatusMsg{{Msg: "update"}}},
			},
			expectedTask: Task[any, any]{
				ID:       taskID,
				Kind:     FirmwareInstall,
				State:    Active,
				WorkerID: "worker-123",
				Status:   StatusRecord{[]StatusMsg{{Msg: "update"}}},
			},
			expectedErr: nil,
		},
		{
			name: "Update fails with ID mismatch",
			initialTask: Task[any, any]{
				ID:     uuid.New(),
				Kind:   FirmwareInstall,
				State:  Pending,
				Status: StatusRecord{},
			},
			updateTask: Task[any, any]{
				ID:     uuid.New(),
				Kind:   FirmwareInstall,
				State:  Active,
				Status: StatusRecord{},
			},
			expectedErr: errors.Wrap(errTaskUpdate, "Task.ID mismatch"),
		},
		{
			name: "Update fails with Kind mismatch",
			initialTask: Task[any, any]{
				ID:     taskID,
				Kind:   FirmwareInstall,
				State:  Pending,
				Status: StatusRecord{},
			},
			updateTask: Task[any, any]{
				ID:     taskID,
				Kind:   Inventory,
				State:  Active,
				Status: StatusRecord{},
			},
			expectedErr: errors.Wrap(errTaskUpdate, "Task.Kind mismatch"),
		},
		{
			name: "Update fails with final state",
			initialTask: Task[any, any]{
				ID:     taskID,
				Kind:   FirmwareInstall,
				State:  Succeeded,
				Status: StatusRecord{},
			},
			updateTask: Task[any, any]{
				ID:     taskID,
				Kind:   FirmwareInstall,
				State:  Active,
				Status: StatusRecord{},
			},
			expectedErr: errors.Wrap(errTaskUpdate, "current state is final: "+string(Succeeded)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := tt.initialTask
			taskUpdate := tt.updateTask

			err := task.Update(&taskUpdate)
			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedTask.State, task.State)
			assert.Equal(t, tt.expectedTask.WorkerID, task.WorkerID)
			assert.Equal(t, tt.expectedTask.Status, task.Status)
			assert.WithinDuration(t, time.Now(), task.UpdatedAt, time.Second)
		})
	}
}

func TestTaskFromMessage(t *testing.T) {
	tests := []struct {
		name         string
		msg          json.RawMessage
		expectedTask *Task[any, any]
		expectedErr  error
	}{
		{
			name: "Valid Task message",
			msg: json.RawMessage(`{
				"task_version": "1.0",
				"worker_id": "worker-123",
				"id": "910e03b0-f84d-4c88-9eeb-e07f35710d03",
				"kind": "firmwareInstall",
				"state": "pending",
				"status": {"records":[{"ts":"0001-01-01T00:00:00Z","msg":"initialized"}]},
				"data": {"empty": true},
				"parameters": {"param": "value"},
				"fault": {"panic": true},
				"facility_code": "123",
				"server": {"UUID": "ed483ef9-098e-4892-bfcf-1696c44fd7a9"},
				"traceID": "123",
				"spanID": "123"}`),
			expectedTask: &Task[any, any]{
				StructVersion: "1.0",
				ID:            uuid.MustParse("910e03b0-f84d-4c88-9eeb-e07f35710d03"),
				Kind:          FirmwareInstall,
				State:         Pending,
				Status:        StatusRecord{[]StatusMsg{{Msg: "initialized"}}},
				Data: map[string]interface{}{
					"empty": true,
				},
				Parameters: map[string]interface{}{
					"param": "value",
				},
				FacilityCode: "123",
				WorkerID:     "worker-123",
				Fault:        &Fault{Panic: true},
				Server:       &Server{UUID: uuid.MustParse("ed483ef9-098e-4892-bfcf-1696c44fd7a9")},
				TraceID:      "123",
				SpanID:       "123",
			},
			expectedErr: nil,
		},
		{
			name:         "Invalid Task message",
			msg:          json.RawMessage(`{"task_version": "1.0"`),
			expectedTask: nil,
			expectedErr:  errors.Wrap(errInvalidTaskJSON, "unexpected end of JSON input"),
		},
		{
			name:         "Empty Task message",
			msg:          json.RawMessage(`{}`),
			expectedTask: &Task[any, any]{},
			expectedErr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := TaskFromMessage(tt.msg)

			if tt.expectedErr != nil {
				assert.Nil(t, task)
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedTask, task)
			}
		})
	}
}
