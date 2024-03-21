package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.hollow.sh/toolbox/events/registry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/metal-toolbox/rivets/condition"
	"github.com/metal-toolbox/rivets/events"
	"github.com/metal-toolbox/rivets/events/pkg/kv"
)

var (
	kvTTL = 10 * 24 * time.Hour
)

// ConditionStatusPublisher defines an interface for publishing status updates for conditions.
type ConditionStatusPublisher interface {
	Publish(ctx context.Context, serverID string, state condition.State, status json.RawMessage)
}

// NatsConditionStatusPublisher implements the StatusPublisher interface to publish condition status updates using NATS.
type NatsConditionStatusPublisher struct {
	kv           nats.KeyValue
	log          *logrus.Logger
	facilityCode string
	conditionID  string
	controllerID string
	lastRev      uint64
}

// NewNatsConditionStatusPublisher creates a new NatsConditionStatusPublisher for a given condition ID.
//
// It initializes a NATS KeyValue store for tracking condition statuses.
func (n *NatsController) NewNatsConditionStatusPublisher(conditionID string) (*NatsConditionStatusPublisher, error) {
	kvOpts := []kv.Option{
		kv.WithDescription(fmt.Sprintf("%s condition status tracking", n.natsConfig.AppName)),
		kv.WithTTL(kvTTL),
		kv.WithReplicas(n.natsConfig.KVReplicationFactor),
	}

	errKV := errors.New("unable to bind to status KV bucket")
	statusKV, err := kv.CreateOrBindKVBucket(n.stream.(*events.NatsJetstream), string(n.conditionKind), kvOpts...)
	if err != nil {
		return nil, errors.Wrap(errKV, err.Error())
	}

	return &NatsConditionStatusPublisher{
		facilityCode: n.facilityCode,
		conditionID:  conditionID,
		controllerID: n.controllerID.String(),
		kv:           statusKV,
		log:          n.logger,
	}, nil
}

// Publish implements the StatusPublisher interface. It serializes and publishes the current status of a condition to NATS.
func (s *NatsConditionStatusPublisher) Publish(ctx context.Context, serverID string, state condition.State, status json.RawMessage) {
	_, span := otel.Tracer(pkgName).Start(
		ctx,
		"controller.Publish.KV",
		trace.WithSpanKind(trace.SpanKindConsumer),
	)
	defer span.End()

	key := fmt.Sprintf("%s.%s", s.facilityCode, s.conditionID)
	sv := &condition.StatusValue{
		WorkerID:  s.controllerID,
		Target:    serverID,
		TraceID:   trace.SpanFromContext(ctx).SpanContext().TraceID().String(),
		SpanID:    trace.SpanFromContext(ctx).SpanContext().SpanID().String(),
		State:     string(state),
		Status:    status,
		UpdatedAt: time.Now(),
	}

	payload := sv.MustBytes()

	var err error
	var rev uint64
	if s.lastRev == 0 {
		rev, err = s.kv.Create(key, payload)
	} else {
		rev, err = s.kv.Update(key, payload, s.lastRev)
	}

	if err != nil {
		metricsNATSError("publish-condition-status")
		span.AddEvent("status publish failure",
			trace.WithAttributes(
				attribute.String("controllerID", s.controllerID),
				attribute.String("serverID", serverID),
				attribute.String("conditionID", s.conditionID),
				attribute.String("error", err.Error()),
			),
		)

		s.log.WithError(err).WithFields(logrus.Fields{
			"serverID":          serverID,
			"assetFacilityCode": s.facilityCode,
			"conditionID":       s.conditionID,
			"lastRev":           s.lastRev,
			"key":               key,
		}).Warn("unable to write task status")
		return
	}

	s.lastRev = rev
	s.log.WithFields(logrus.Fields{
		"serverID":          serverID,
		"assetFacilityCode": s.facilityCode,
		"taskID":            s.conditionID,
		"lastRev":           s.lastRev,
		"key":               key,
	}).Trace("published task status")
}

// conditionState represents the various states a condition can be in during its lifecycle.
type conditionState int

const (
	notStarted    conditionState = iota
	inProgress                   // another controller has started it, is still around and updated recently
	complete                     // condition is done
	orphaned                     // the controller that started this task doesn't exist anymore
	indeterminate                // we got an error in the process of making the check
)

// ConditionStatusQueryor defines an interface for querying the status of a condition.
type ConditionStatusQueryor interface {
	// ConditionState returns the current state of a condition based on its ID.
	ConditionState(conditionID string) conditionState
}

// NatsConditionStatusQueryor implements ConditionStatusQueryor to query condition states using NATS.
type NatsConditionStatusQueryor struct {
	kv           nats.KeyValue
	logger       *logrus.Logger
	facilityCode string
	controllerID string
}

// NewNatsConditionStatusQueryor creates a new NatsConditionStatusQueryor instance, initializing a NATS KeyValue store for condition status queries.
func (n *NatsController) NewNatsConditionStatusQueryor() (*NatsConditionStatusQueryor, error) {
	errKV := errors.New("unable to connect to status KV for condition progress lookup")
	kvHandle, err := events.AsNatsJetStreamContext(n.stream.(*events.NatsJetstream)).KeyValue(string(n.conditionKind))
	if err != nil {
		n.logger.WithError(err).Error(errKV.Error())
		return nil, errors.Wrap(errKV, err.Error())
	}

	return &NatsConditionStatusQueryor{
		kv:           kvHandle,
		logger:       n.logger,
		facilityCode: n.facilityCode,
		controllerID: n.controllerID.String(),
	}, nil
}

// ConditionState queries the NATS KeyValue store to determine the current state of a condition.
func (p *NatsConditionStatusQueryor) ConditionState(conditionID string) conditionState {
	lookupKey := fmt.Sprintf("%s.%s", p.facilityCode, conditionID)
	entry, err := p.kv.Get(lookupKey)
	switch err {
	case nats.ErrKeyNotFound:
		// This should be by far the most common path through this code.
		return notStarted
	case nil:
		break // we'll handle this outside the switch
	default:
		p.logger.WithError(err).WithFields(logrus.Fields{
			"conditionID": conditionID,
			"lookupKey":   lookupKey,
		}).Warn("error reading condition status")

		return indeterminate
	}

	// we have an status entry for this condition, is is complete?
	sv := condition.StatusValue{}
	if errJSON := json.Unmarshal(entry.Value(), &sv); errJSON != nil {
		p.logger.WithError(errJSON).WithFields(logrus.Fields{
			"conditionID": conditionID,
			"lookupKey":   lookupKey,
		}).Warn("unable to construct a sane status for this condition")

		return indeterminate
	}

	if condition.State(sv.State) == condition.Failed ||
		condition.State(sv.State) == condition.Succeeded {
		p.logger.WithFields(logrus.Fields{
			"conditionID":    conditionID,
			"conditionState": sv.State,
			"lookupKey":      lookupKey,
		}).Info("this condition is already complete")

		return complete
	}

	// is the worker handling this condition alive?
	worker, err := registry.ControllerIDFromString(sv.WorkerID)
	if err != nil {
		p.logger.WithError(err).WithFields(logrus.Fields{
			"conditionID":           conditionID,
			"original controllerID": sv.WorkerID,
		}).Warn("bad controller identifier")

		return indeterminate
	}

	activeAt, err := registry.LastContact(worker)
	switch err {
	case nats.ErrKeyNotFound:
		// the data for this worker aged-out, it's no longer active
		// XXX: the most conservative thing to do here is to return
		// indeterminate but most times this will indicate that the
		// worker crashed/restarted and this task should be restarted.
		p.logger.WithFields(logrus.Fields{
			"conditionID":           conditionID,
			"original controllerID": sv.WorkerID,
		}).Info("original controller not found")

		// We're going to restart this condition when we return from
		// this function. Use the KV handle we have to delete the
		// existing task key.
		if err = p.kv.Delete(lookupKey); err != nil {
			p.logger.WithError(err).WithFields(logrus.Fields{
				"conditionID":           conditionID,
				"original controllerID": sv.WorkerID,
				"lookupKey":             lookupKey,
			}).Warn("unable to delete existing condition status")

			return indeterminate
		}

		return orphaned
	case nil:
		timeStr, _ := activeAt.MarshalText()
		p.logger.WithError(err).WithFields(logrus.Fields{
			"conditionID":           conditionID,
			"original controllerID": sv.WorkerID,
			"lastActive":            timeStr,
		}).Warn("error looking up controller last contact")

		return inProgress
	default:
		p.logger.WithError(err).WithFields(logrus.Fields{
			"conditionID":           conditionID,
			"original controllerID": sv.WorkerID,
		}).Warn("error looking up controller last contact")

		return indeterminate
	}
}

// eventStatusAcknowleger provides an interface for acknowledging the status of events in a NATS JetStream.
type eventStatusAcknowleger interface {
	inProgress()
	complete()
	nak()
}

// natsEventStatusAcknowleger implements eventStatusAcknowleger to interact with NATS JetStream events.
type natsEventStatusAcknowleger struct {
	event  events.Message
	logger *logrus.Logger
}

func (n *NatsController) newNatsEventStatusAcknowleger(event events.Message) *natsEventStatusAcknowleger {
	return &natsEventStatusAcknowleger{event, n.logger}
}

// inProgress marks the event as being in progress in the NATS JetStream.
func (p *natsEventStatusAcknowleger) inProgress() {
	if err := p.event.InProgress(); err != nil {
		metricsNATSError("ack-in-progress")
		p.logger.WithError(err).Warn("event Ack as Inprogress returned error")
		return
	}

	p.logger.Trace("event ack as InProgress successful")
}

// complete marks the event as complete in the NATS JetStream.
func (p *natsEventStatusAcknowleger) complete() {
	if err := p.event.Ack(); err != nil {
		metricsNATSError("ack")
		p.logger.WithError(err).Warn("event Ack as complete returned error")
		return
	}

	p.logger.Trace("event ack as Complete successful")
}

// nak sends a negative acknowledgment for the event in the NATS JetStream, indicating it requires further handling.
func (p *natsEventStatusAcknowleger) nak() {
	if err := p.event.Nak(); err != nil {
		metricsNATSError("nak")
		p.logger.WithError(err).Warn("event Nak error")
		return
	}

	p.logger.Trace("event nak successful")
}