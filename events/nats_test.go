//nolint:all
package events

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	traceSDK "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	natsTest "github.com/metal-automata/rivets/events/internal/test"
)

func TestJetstreamFromConn(t *testing.T) {
	t.Parallel()
	core := natsTest.StartCoreServer(t)
	defer func() {
		core.Shutdown()
		core.WaitForShutdown()
	}()

	jsSrv := natsTest.StartJetStreamServer(t)
	defer natsTest.ShutdownJetStream(t, jsSrv)

	jsConn, _ := natsTest.JetStreamContext(t, jsSrv)

	conn, err := nats.Connect(core.ClientURL())
	require.NoError(t, err, "error on initial connection to core test server")
	defer conn.Close()

	// XXX: you can successfuilly get a JetStreamContext from a non-JetStream-enabled server but
	// it's an error to actually use it. That's super annoying. An error is returned from
	// conn.JetStream() only when incompatible options are provided to the function.
	njs := NewJetstreamFromConn(conn)

	_, err = AsNatsJetStreamContext(njs).AccountInfo()
	require.Error(t, err, "expected an error trying to use JetStream functions on a core server")
	require.Equal(t, nats.ErrJetStreamNotEnabled, err, "unexpected error touching JetStream context from core server")

	njs = NewJetstreamFromConn(jsConn)
	_, err = AsNatsJetStreamContext(njs).AccountInfo()
	require.NoError(t, err, "unexpected error using JetStream")
	njs.Close()
}

func TestPublishAndSubscribe(t *testing.T) {
	jsSrv := natsTest.StartJetStreamServer(t)
	defer natsTest.ShutdownJetStream(t, jsSrv)

	jsConn, _ := natsTest.JetStreamContext(t, jsSrv)
	njs := NewJetstreamFromConn(jsConn)
	defer njs.Close()

	subject := "pre.test"
	njs.parameters = &NatsOptions{
		AppName: "TestPublishAndSubscribe",
		Stream: &NatsStreamOptions{
			Name: "test_stream",
			Subjects: []string{
				subject,
			},
			Retention: "workQueue",
		},
		Consumer: &NatsConsumerOptions{
			Name: "test_consumer",
			Pull: true,
			SubscribeSubjects: []string{
				subject,
			},
			FilterSubject: subject,
		},
		PublisherSubjectPrefix: "pre",
	}
	require.NoError(t, njs.addStream())
	require.NoError(t, njs.addConsumer())

	_, err := njs.Subscribe(context.TODO())
	require.NoError(t, err)

	payload := []byte("test data")
	require.NoError(t, njs.Publish(context.TODO(), "test", payload))

	msg, err := njs.PullOneMsg(context.TODO(), subject)
	require.NoError(t, err)
	require.Equal(t, payload, msg.Data())

	_, err = njs.PullOneMsg(context.TODO(), subject)
	require.Error(t, err)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestPublishAndSubscribe_WithRollup(t *testing.T) {
	jsSrv := natsTest.StartJetStreamServer(t)
	defer natsTest.ShutdownJetStream(t, jsSrv)

	jsConn, _ := natsTest.JetStreamContext(t, jsSrv)
	njs := NewJetstreamFromConn(jsConn)
	defer njs.Close()

	subject := "pre.test"
	njs.parameters = &NatsOptions{
		AppName: "TestPublishAndSubscribe",
		Stream: &NatsStreamOptions{
			Name: "test_stream",
			Subjects: []string{
				subject,
			},
			Retention: "workQueue",
		},
		Consumer: &NatsConsumerOptions{
			Name: "test_consumer",
			Pull: true,
			SubscribeSubjects: []string{
				subject,
			},
			FilterSubject: subject,
		},
		PublisherSubjectPrefix: "pre",
	}
	require.NoError(t, njs.addStream())
	require.NoError(t, njs.addConsumer())

	_, err := njs.Subscribe(context.TODO())
	require.NoError(t, err)

	payload := []byte("test data")
	require.NoError(t, njs.PublishOverwrite(context.TODO(), "test", payload))
	payload2 := []byte("rollup")
	require.NoError(t, njs.PublishOverwrite(context.TODO(), "test", payload2))

	msg, err := njs.PullOneMsg(context.TODO(), subject)
	require.NoError(t, err)
	require.Equal(t, payload2, msg.Data())

	_, err = njs.PullOneMsg(context.TODO(), subject)
	require.Error(t, err)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func Test_addConsumer(t *testing.T) {
	jsSrv := natsTest.StartJetStreamServer(t)
	defer natsTest.ShutdownJetStream(t, jsSrv)

	jsConn, _ := natsTest.JetStreamContext(t, jsSrv)
	njs := NewJetstreamFromConn(jsConn)
	defer njs.Close()

	subjects := []string{"pre.test", "pre.bar"}
	consumerCfg := &NatsConsumerOptions{
		Name:              "test_consumer",
		QueueGroup:        "test",
		Pull:              true,
		SubscribeSubjects: subjects,
		MaxAckPending:     10,
		AckWait:           600 * time.Second,
	}

	njs.parameters = &NatsOptions{
		AppName: "TestPublishAndSubscribe",
		Stream: &NatsStreamOptions{
			Name:      "test_stream",
			Subjects:  subjects,
			Retention: "workQueue",
		},
		Consumer:               consumerCfg,
		PublisherSubjectPrefix: "pre",
	}

	require.NoError(t, njs.addStream())

	// add config
	require.NoError(t, njs.addConsumer())

	consumerInfo, err := njs.jsctx.ConsumerInfo("test_stream", consumerCfg.Name)
	require.NoError(t, err)

	assert.Equal(t, consumerCfg.Name, consumerInfo.Name)
	assert.Equal(t, false, consumerInfo.PushBound)
	assert.Equal(t, consumerCfg.MaxAckPending, consumerInfo.Config.MaxAckPending)
	assert.Equal(t, consumerMaxDeliver, consumerInfo.Config.MaxDeliver)
	assert.Equal(t, consumerAckPolicy, consumerInfo.Config.AckPolicy)
	assert.Equal(t, consumerCfg.AckWait, consumerInfo.Config.AckWait)
	assert.Equal(t, consumerDeliverPolicy, consumerInfo.Config.DeliverPolicy)
	assert.Equal(t, consumerCfg.QueueGroup, consumerInfo.Config.DeliverGroup)
	// TODO: for some reason the stream does not indicate it has multiple filter subjects
	// nats server bug?
	//assert.Equal(t, consumerCfg.SubscribeSubjects, consumerInfo.Config.FilterSubjects)

	// update config
	consumerCfg.MaxAckPending = 30
	require.NoError(t, njs.addConsumer())

	consumerInfo, err = njs.jsctx.ConsumerInfo("test_stream", consumerCfg.Name)
	require.NoError(t, err)

	assert.Equal(t, consumerCfg.MaxAckPending, consumerInfo.Config.MaxAckPending)
}

func TestInjectOtelTraceContext(t *testing.T) {
	// set the tracing propagator so its available for injection
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}),
	)

	// setup a new trace provider
	ctx, span := traceSDK.NewTracerProvider().Tracer("testing").Start(context.Background(), "foo.bar")
	defer span.End()

	msg := nats.NewMsg("foo.bar")
	msg.Data = []byte(`hello`)

	injectOtelTraceContext(ctx, msg)

	assert.NotEmpty(t, msg.Header.Get("Traceparent"))
}

func TestExtractOtelTraceContext(t *testing.T) {
	// set the tracing propagator so its available for injection
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}),
	)

	// setup a new trace provider
	ctx, span := traceSDK.NewTracerProvider().Tracer("testing").Start(context.Background(), "foo.bar")
	defer span.End()

	msg := nats.NewMsg("foo.bar")
	msg.Data = []byte(`hello`)

	// inject
	injectOtelTraceContext(ctx, msg)

	// msg header gets a trace parent added
	traceParent := msg.Header.Get("Traceparent")

	// wrap natsMsg to pass to extract method
	nm := &natsMsg{msg}

	ctxWithTrace := nm.ExtractOtelTraceContext(context.Background())
	got := trace.SpanFromContext(ctxWithTrace).SpanContext().TraceID().String()

	assert.Contains(t, traceParent, got)
}

func Test_addStream(t *testing.T) {
	jsSrv := natsTest.StartJetStreamServer(t)
	defer natsTest.ShutdownJetStream(t, jsSrv)

	jsConn, _ := natsTest.JetStreamContext(t, jsSrv)
	njs := NewJetstreamFromConn(jsConn)
	defer njs.Close()

	streamName := "foo"
	njs.parameters = &NatsOptions{
		Stream: &NatsStreamOptions{
			Name:             streamName,
			Subjects:         []string{"foo.bar"},
			Retention:        "workQueue",
			Acknowledgements: true,
			DuplicateWindow:  1 * time.Minute,
		},
	}

	// test add
	assert.Nil(t, njs.addStream())

	streamInfo, err := AsNatsJetStreamContext(njs).StreamInfo(streamName)
	assert.Nil(t, err)

	// validate defaults
	assert.Equal(t, njs.parameters.Stream.Name, streamInfo.Config.Name)
	assert.Equal(t, njs.parameters.Stream.Subjects, streamInfo.Config.Subjects)
	assert.Equal(t, nats.WorkQueuePolicy, streamInfo.Config.Retention)
	assert.True(t, streamInfo.Config.AllowRollup)
	assert.Equal(t, conditionJetstreamTTL, streamInfo.Config.MaxAge)

	assert.Equal(t, njs.parameters.Stream.DuplicateWindow, streamInfo.Config.Duplicates)
	assert.False(t, streamInfo.Config.NoAck)

	// test update
	njs.parameters.Stream.Acknowledgements = false
	njs.parameters.Stream.DuplicateWindow = 5 * time.Minute
	njs.parameters.Stream.Subjects = []string{"bar.bar"}

	assert.Nil(t, njs.addStream())
	streamInfo2, err := AsNatsJetStreamContext(njs).StreamInfo(streamName)
	assert.Nil(t, err)

	assert.Equal(t, 5*time.Minute, streamInfo2.Config.Duplicates)
	assert.Equal(t, njs.parameters.Stream.Subjects, streamInfo2.Config.Subjects)
}
