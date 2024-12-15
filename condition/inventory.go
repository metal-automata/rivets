package condition

import (
	"encoding/json"

	"github.com/google/uuid"
)

type InventoryMethod string

const (
	// Condition kind
	Inventory Kind = "inventory"

	// Inventory methods
	InbandInventory    InventoryMethod = "inband"
	OutofbandInventory InventoryMethod = "outofband"
)

// InventoryTaskParameters are the parameters set for an inventory collection condition
//
// nolint:govet // prefer readability over fieldalignment for this case
type InventoryTaskParameters struct {
	// CollectBiosCfg defaults to true
	CollectBiosCfg bool `json:"collect_bios_cfg"`

	// CollectFirmwareStatus defaults to true
	CollectFirwmareStatus bool `json:"collect_firmware_status"`

	// Method defaults to Outofband
	Method InventoryMethod `json:"inventory_method"`

	// Asset identifier.
	AssetID uuid.UUID `json:"asset_id"`
}

func NewInventoryTaskParameters(assetID uuid.UUID, method InventoryMethod, collectFirmwareStatus, collectBiosCfg bool) *InventoryTaskParameters {
	return &InventoryTaskParameters{
		AssetID:               assetID,
		CollectBiosCfg:        collectBiosCfg,
		CollectFirwmareStatus: collectFirmwareStatus,
		Method:                method,
	}
}

func MustInventoryJSON(assetID uuid.UUID, method InventoryMethod, collectFirmwareStatus, collectBiosCfg bool) []byte {
	p := &InventoryTaskParameters{
		AssetID:               assetID,
		CollectBiosCfg:        collectBiosCfg,
		CollectFirwmareStatus: collectFirmwareStatus,
		Method:                method,
	}
	byt, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	return byt
}

func MustDefaultInventoryJSON(assetID uuid.UUID) []byte {
	return MustInventoryJSON(assetID, OutofbandInventory, true, true)
}

func (p *InventoryTaskParameters) MapStringInterfaceToStruct(m map[string]interface{}) error {
	jsonData, err := json.Marshal(m)
	if err != nil {
		return err
	}

	return json.Unmarshal(jsonData, p)
}

func (p *InventoryTaskParameters) Unmarshal(r json.RawMessage) error {
	return json.Unmarshal(r, p)
}

func (p *InventoryTaskParameters) Marshal() (json.RawMessage, error) {
	return json.Marshal(p)
}
