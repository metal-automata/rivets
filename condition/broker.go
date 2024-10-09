package condition

import (
	"encoding/json"

	"github.com/google/uuid"
)

type (
	// BrokerAction identifies the kind of boot image action to be taken.
	BrokerAction string
	// BrokerActionPurpose hints the controller as to the reason for the action.
	BrokerActionPurpose string
)

const (
	Broker Kind = "broker"

	// These two Kinds are added just to enable the Condition API, Orchestrator to differentiate
	// between the two Kinds when applying an Update.
	//
	// https://github.com/metal-automata/conditionorc/blob/92f9b1619074b7c4ef4b35f39c3beab3c254966f/internal/store/nats.go#L263
	//
	BrokerAcquireServer Kind = "broker.acquireServer"
	BrokerReleaseServer Kind = "broker.releaseServer"

	// PurposeFirmwareInstall tells the target controller to take necessary steps
	// to ensure the server is reserved for a firmware install.
	PurposeFirmwareInstall BrokerActionPurpose = "firmwareInstall"

	// Acquire indicates the server is to be flagged into a state where its unavailable to other systems.
	AcquireServer BrokerAction = "acquireServer"
	// ReleaseServer indicates the server is to be reverted to its previous state.
	ReleaseServer BrokerAction = "releaseServer"
)

type BrokerTaskParameters struct {
	AssetID       uuid.UUID           `json:"asset_id"`
	Action        BrokerAction        `json:"action"`
	Purpose       BrokerActionPurpose `json:"action_purpose"`
	ServerAcquire *ServerAcquire      `json:"acquire,omitempty"`
	ServerRelease *ServerRelease      `json:"release,omitempty"`
}

func (p *BrokerTaskParameters) MapStringInterfaceToStruct(m map[string]interface{}) error {
	jsonData, err := json.Marshal(m)
	if err != nil {
		return err
	}

	return json.Unmarshal(jsonData, p)
}

func (p *BrokerTaskParameters) Unmarshal(r json.RawMessage) error {
	return json.Unmarshal(r, p)
}

func (p *BrokerTaskParameters) Marshal() (json.RawMessage, error) {
	return json.Marshal(p)
}

func (p *BrokerTaskParameters) MustMarshal() json.RawMessage {
	byt, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}

	return byt
}

type ServerAcquire struct {
	Info                 string `json:"info"`
	CustomIpxe           bool   `json:"custom_ipxe"`
	PersistentCustomIpxe bool   `json:"persistent_custom_ipxe"`
}

type ServerRelease struct {
	Info string `json:"info"`
}

func NewBrokerTaskParameters(assetID uuid.UUID, action BrokerAction, purpose BrokerActionPurpose, info string) *BrokerTaskParameters {
	switch action {
	case AcquireServer:
		return &BrokerTaskParameters{
			AssetID: assetID,
			Action:  AcquireServer,
			Purpose: purpose,
			ServerAcquire: &ServerAcquire{
				Info:                 info,
				CustomIpxe:           true,
				PersistentCustomIpxe: true,
			},
		}
	case ReleaseServer:
		return &BrokerTaskParameters{
			AssetID: assetID,
			Action:  ReleaseServer,
			Purpose: purpose,
			ServerRelease: &ServerRelease{
				Info: info,
			},
		}
	}

	return nil
}
