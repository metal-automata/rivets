//nolint:all
package kv

import (
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/metal-automata/rivets/events"
	kvTest "github.com/metal-automata/rivets/events/internal/test"

	"github.com/stretchr/testify/require"
)

func TestDefaultConfigAndOptions(t *testing.T) {
	t.Parallel()
	cfg := DefaultKVConfig("test")
	require.NotNil(t, cfg)
	require.Equal(t, "test", cfg.Bucket)
	require.Equal(t, 0, cfg.Replicas)
	require.Equal(t, time.Duration(0), cfg.TTL)
	require.Equal(t, "", cfg.Description)
	funcs := []Option{
		WithTTL(2 * time.Minute),
		WithStorageType(nats.MemoryStorage),
		WithReplicas(3),
		WithDescription("test"),
	}
	for _, f := range funcs {
		f(cfg)
	}
	require.Equal(t, "test", cfg.Description)
	require.Equal(t, 3, cfg.Replicas)
	require.Equal(t, nats.MemoryStorage, cfg.Storage)
	require.Equal(t, 2*time.Minute, cfg.TTL)
}

func TestCreateOrBind(t *testing.T) {
	srv := kvTest.StartJetStreamServer(t)
	defer kvTest.ShutdownJetStream(t, srv)
	nc, _ := kvTest.JetStreamContext(t, srv)

	evJS := events.NewJetstreamFromConn(nc)
	defer evJS.Close()

	kv, err := CreateOrBindKVBucket(evJS, "test-bucket")
	require.NoError(t, err)
	require.NotNil(t, kv)

	// now test the bind only-path
	kv2, err := CreateOrBindKVBucket(evJS, "test-bucket")
	require.NoError(t, err)
	require.NotNil(t, kv2)
}
