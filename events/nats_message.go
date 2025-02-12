//nolint:wsl // useless
package events

import (
	"context"

	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// here we implement the Message interface for nats.Msg

// AsNatsMsg exposes the underlying nats.Msg to a sophisticated consumer.
func AsNatsMsg(m Message) (*nats.Msg, error) {
	nm, ok := m.(*natsMsg)
	if !ok {
		return nil, errors.New("Message is not a NATS message type")
	}
	return nm.msg, nil
}

// MustNatsMsg will panic if the type assertion fails
func MustNatsMsg(m Message) *nats.Msg {
	nm := m.(*natsMsg)
	return nm.msg
}

type natsMsg struct {
	msg *nats.Msg
}

func (nm *natsMsg) Ack() error {
	return nm.msg.Ack()
}
func (nm *natsMsg) Nak() error {
	return nm.msg.Nak()
}

func (nm *natsMsg) Term() error {
	return nm.msg.Term()
}

func (nm *natsMsg) InProgress() error {
	return nm.msg.InProgress()
}

func (nm *natsMsg) Subject() string {
	return nm.msg.Subject
}

func (nm *natsMsg) Data() []byte {
	return nm.msg.Data
}

func (nm *natsMsg) ExtractOtelTraceContext(ctx context.Context) context.Context {
	if nm == nil || nm.msg.Header == nil {
		return ctx
	}

	return otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(nm.msg.Header))
}
