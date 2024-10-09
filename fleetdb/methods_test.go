package fleetdb

import (
	"encoding/json"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	ss "github.com/metal-automata/fleetdb/pkg/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertComponents(t *testing.T) {
	// Test case 1: Empty input
	cs := []ss.ServerComponent{}
	result := ConvertComponents(cs)
	assert.Empty(t, result, "ConvertComponents should return an empty slice for empty input")

	// Test case 2: Conversion with valid input
	cs = []ss.ServerComponent{
		{
			UUID:                uuid.New(),
			ServerUUID:          uuid.New(),
			Name:                "CPU",
			Vendor:              "Intel",
			Model:               "Xeon",
			Serial:              "ABC123",
			Attributes:          []ss.Attributes{},
			VersionedAttributes: []ss.VersionedAttributes{},
			ComponentTypeID:     "cpu-type",
			ComponentTypeName:   "cpu",
			ComponentTypeSlug:   "cpu",
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		},
		{
			UUID:                uuid.New(),
			ServerUUID:          uuid.New(),
			Name:                "Memory",
			Vendor:              "Samsung",
			Model:               "DDR4",
			Serial:              "XYZ456",
			Attributes:          []ss.Attributes{},
			VersionedAttributes: []ss.VersionedAttributes{},
			ComponentTypeID:     "memory-type",
			ComponentTypeName:   "memory",
			ComponentTypeSlug:   "memory",
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		},
	}

	result = ConvertComponents(cs)

	assert.Len(t, result, 2, "ConvertComponents should return a slice with two elements")

	// Sort the result for consistent comparison
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	// Check the converted components
	assert.Equal(t, cs[0].UUID.String(), result[0].ID)
	assert.Equal(t, cs[0].ComponentTypeSlug, result[0].Name)
	assert.Equal(t, cs[0].Vendor, result[0].Vendor)
	assert.Equal(t, cs[0].Model, result[0].Model)
	assert.Equal(t, cs[0].Serial, result[0].Serial)

	assert.Equal(t, cs[1].UUID.String(), result[1].ID)
	assert.Equal(t, cs[1].ComponentTypeSlug, result[1].Name)
	assert.Equal(t, cs[1].Vendor, result[1].Vendor)
	assert.Equal(t, cs[1].Model, result[1].Model)
	assert.Equal(t, cs[1].Serial, result[1].Serial)

	// Test case 3: Conversion with versioned attributes
	cs = []ss.ServerComponent{
		{
			UUID:       uuid.New(),
			ServerUUID: uuid.New(),
			Name:       "BIOS",
			Vendor:     "Dell",
			Model:      "Version X",
			Serial:     "DEF789",
			Attributes: []ss.Attributes{},
			VersionedAttributes: []ss.VersionedAttributes{
				{
					Namespace: FirmwareVersionInbandNS,
					Data:      json.RawMessage(`{"firmware": {"installed": "1.2.3"}}`),
				},
				{
					Namespace: StatusInbandNS,
					Data:      json.RawMessage(`{"status": {"state": "ok"}}`),
				},
			},
			ComponentTypeID:   "bios-type",
			ComponentTypeName: "BIOS",
			ComponentTypeSlug: "BIOS",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
	}

	result = ConvertComponents(cs)

	assert.Len(t, result, 1, "ConvertComponents should return a slice with one element")

	// Check the converted component with versioned attributes
	assert.Equal(t, cs[0].UUID.String(), result[0].ID)
	assert.Equal(t, cs[0].Name, result[0].Name)
	assert.Equal(t, cs[0].Vendor, result[0].Vendor)
	assert.Equal(t, cs[0].Model, result[0].Model)
	assert.Equal(t, cs[0].Serial, result[0].Serial)

	assert.NotNil(t, result[0].Firmware)
	assert.Equal(t, "1.2.3", result[0].Firmware.Installed)

	assert.NotNil(t, result[0].Status)
	assert.Equal(t, "ok", result[0].Status.State)
}

func TestRecordToComponent(t *testing.T) {
	t.Parallel()
	// test that the base transformations work properly
	t.Run("no variable attributes", func(t *testing.T) {
		t.Parallel()

		record := &ss.ServerComponent{
			UUID:       uuid.New(),
			ServerUUID: uuid.New(),
			Name:       "BIOS",
			Vendor:     "Dell",
			Model:      "Version X",
			Serial:     "DEF789",
			Attributes: []ss.Attributes{
				{
					Namespace: "sh.hollow.alloy.inband.metadata",
					Data:      json.RawMessage(`{"ID": "my-id"}`),
				},
			},
			VersionedAttributes: []ss.VersionedAttributes{},
			ComponentTypeName:   "BIOS",
			ComponentTypeSlug:   "bios",
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}

		got, err := RecordToComponent(record)
		require.NoError(t, err)
		require.Equal(t, record.UUID.String(), got.ID)
		require.Equal(t, record.ComponentTypeSlug, got.Name)
		require.Equal(t, record.Vendor, got.Vendor)
		require.Equal(t, record.Model, got.Model)
		require.Equal(t, record.Serial, got.Serial)
		require.Equal(t, record.UpdatedAt, got.UpdatedAt)
		require.NotNil(t, got.Attributes)
		require.Equal(t, "my-id", got.Attributes.ID)
	})
	t.Run("populate empty fields", func(t *testing.T) {
		t.Parallel()
		record := &ss.ServerComponent{
			UUID:       uuid.New(),
			ServerUUID: uuid.New(),
			Name:       "BIOS",
			Vendor:     "Dell",
			Model:      "Version X",
			Serial:     "DEF789",
			Attributes: []ss.Attributes{},
			VersionedAttributes: []ss.VersionedAttributes{
				{
					Namespace: FirmwareVersionInbandNS,
					Data:      json.RawMessage(`{"firmware": {"installed": "1.2.3"}}`),
				},
				{
					Namespace: StatusOutofbandNS,
					Data:      json.RawMessage(`{"status": {"state": "ok"}}`),
				},
			},
			ComponentTypeID:   "bios-type",
			ComponentTypeName: "BIOS",
			ComponentTypeSlug: "bios",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		got, err := RecordToComponent(record)
		require.NoError(t, err)
		require.Equal(t, "1.2.3", got.Firmware.Installed)
		require.Equal(t, "ok", got.Status.State)
	})
	t.Run("specific overrides", func(t *testing.T) {
		t.Parallel()
		record := &ss.ServerComponent{
			UUID:       uuid.New(),
			ServerUUID: uuid.New(),
			Name:       "BIOS",
			Vendor:     "Dell",
			Model:      "Version X",
			Serial:     "DEF789",
			Attributes: []ss.Attributes{},
			VersionedAttributes: []ss.VersionedAttributes{
				{
					Namespace: FirmwareVersionOutofbandNS,
					Data:      json.RawMessage(`{"firmware": {"installed": "1.2.3"}}`),
				},
				{
					Namespace: StatusOutofbandNS,
					Data:      json.RawMessage(`{"status": {"state": "ok"}}`),
				},
				{
					Namespace: FirmwareVersionInbandNS,
					Data:      json.RawMessage(`{"firmware": {"installed": "1.2.3-45"}}`),
				},
				{
					Namespace: StatusInbandNS,
					Data:      json.RawMessage(`{"status": {"state": "ok, but could be better"}}`),
				},
			},
			ComponentTypeID:   "bios-type",
			ComponentTypeName: "BIOS",
			ComponentTypeSlug: "bios",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		got, err := RecordToComponent(record)
		require.NoError(t, err)
		require.Equal(t, "1.2.3", got.Firmware.Installed, "firmware version does not favor out-of-band")
		require.Equal(t, "ok, but could be better", got.Status.State, "status does not favor in-band")
	})
	t.Run("malformed variable attribute returns an error", func(t *testing.T) {
		t.Parallel()
		record := &ss.ServerComponent{
			UUID:       uuid.New(),
			ServerUUID: uuid.New(),
			Name:       "BIOS",
			Vendor:     "Dell",
			Model:      "Version X",
			Serial:     "DEF789",
			Attributes: []ss.Attributes{},
			VersionedAttributes: []ss.VersionedAttributes{
				{
					Namespace: FirmwareVersionOutofbandNS,
					Data:      json.RawMessage(`{"firmware": {"installed": 1.2.3}}`),
				},
			},
		}
		_, err := RecordToComponent(record)
		require.ErrorIs(t, err, errBadVAttr)
	})
}
