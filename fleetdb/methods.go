package fleetdb

import (
	"encoding/json"
	"sort"

	ss "github.com/metal-automata/fleetdb/pkg/api/v1"
	rt "github.com/metal-automata/rivets/types"
	"github.com/pkg/errors"
)

var (
	errBadVAttr = errors.New("unable to deserialize versioned attribute")
	errBadAttr  = errors.New("unable to deserialize attribute")
)

func UnpackVersionedAttribute(attr *ss.VersionedAttributes, dst any) error {
	err := json.Unmarshal(attr.Data, dst)
	if err != nil {
		return errors.Wrap(errBadVAttr, err.Error())
	}

	return nil
}

func UnpackAttribute(attr *ss.Attributes, dst any) error {
	err := json.Unmarshal(attr.Data, dst)
	if err != nil {
		return errors.Wrap(errBadAttr, err.Error())
	}

	return nil
}

// RecordToComponent takes a single incoming FleetDB component record and creates
// a rivets Component from it. An important difference from ConvertComponents (cf.
// below) is that this sets a given attribute/versioned-attribute preferentially
// based on some domain-specific factors. For most things, we prefer in-band data,
// as we find that it is more complete, depending on the vendor. Dell is generally
// equivalent information, but SMC is far more detailed in-band. The exception is
// firmware versions, where Dell and SMC provide equivalent information in-band
// vs. not, and biasing in-band inventory would actually hide updates from firmware
// installs because the new versions are discovered via the BMC.
//
//nolint:gocyclo,gocritic
func RecordToComponent(rec *ss.ServerComponent) (*rt.Component, error) {
	component := &rt.Component{
		ID:        rec.UUID.String(),
		Name:      rec.ComponentTypeSlug,
		Vendor:    rec.Vendor,
		Model:     rec.Model,
		Serial:    rec.Serial,
		UpdatedAt: rec.UpdatedAt,
	}

	for _, va := range rec.VersionedAttributes {
		va := va
		// the following relies on the knowledge that we'll get only the latest
		// versioned attributes from FleetDB.
		switch va.Namespace {
		case FirmwareVersionInbandNS:
			if component.Firmware == nil {
				fwva := &FirmwareVersionedAttribute{}
				if err := UnpackVersionedAttribute(&va, fwva); err != nil {
					return nil, err
				}
				component.Firmware = fwva.Firmware
			}
		case FirmwareVersionOutofbandNS:
			fwva := &FirmwareVersionedAttribute{}
			if err := UnpackVersionedAttribute(&va, fwva); err != nil {
				return nil, err
			}
			component.Firmware = fwva.Firmware
		case StatusInbandNS:
			st := &StatusVersionedAttribute{}
			if err := UnpackVersionedAttribute(&va, st); err != nil {
				return nil, err
			}
			component.Status = st.Status
		case StatusOutofbandNS:
			if component.Status == nil {
				st := &StatusVersionedAttribute{}
				if err := UnpackVersionedAttribute(&va, st); err != nil {
					return nil, err
				}
				component.Status = st.Status
			}
		}
	}

	for _, a := range rec.Attributes {
		a := a // this doesn't need to be here after go 1.22
		if a.Namespace == ComponentAttributeInbandNS {
			attrs := &rt.ComponentAttributes{}
			if err := UnpackAttribute(&a, attrs); err != nil {
				return nil, err
			}
			component.Attributes = attrs
			break
		}
	}

	return component, nil
}

// ConvertComponents converts Serverservice components to the rivets component type
// nolint:gocyclo // component conversion is cyclomatic
func ConvertComponents(cs []ss.ServerComponent) []*rt.Component {
	if cs == nil {
		return nil
	}

	set := make([]*rt.Component, 0, len(cs))
	for idx := range cs {
		c := cs[idx]
		component := &rt.Component{
			ID:        c.UUID.String(),
			Name:      c.ComponentTypeSlug,
			Vendor:    c.Vendor,
			Model:     c.Model,
			Serial:    c.Serial,
			UpdatedAt: c.UpdatedAt,
		}

		for _, vattr := range c.VersionedAttributes {
			vattr := vattr
			// TODO: set the most current data from either the inband/outofband NS
			switch vattr.Namespace {
			case FirmwareVersionInbandNS, FirmwareVersionOutofbandNS:
				fw := &FirmwareVersionedAttribute{}
				if err := UnpackVersionedAttribute(&vattr, fw); err == nil {
					component.Firmware = fw.Firmware
				}

			case StatusOutofbandNS, StatusInbandNS:
				st := &StatusVersionedAttribute{}
				if err := UnpackVersionedAttribute(&vattr, st); err == nil {
					component.Status = st.Status
				}
			}
		}

		for _, attr := range c.Attributes {
			attr := attr
			switch attr.Namespace {
			case ComponentAttributeInbandNS, ComponentAttributeOutofbandNS:
				a := &rt.ComponentAttributes{}
				if err := UnpackAttribute(&attr, a); err == nil {
					component.Attributes = a
				}
			}
		}

		set = append(set, component)
	}

	sort.Slice(set, func(i, j int) bool {
		return set[i].Name < set[j].Name
	})

	return set
}

// ConvertServer converts the serverservice server to the rivets Server type
func ConvertServer(s *ss.Server) *rt.Server {
	if s == nil {
		return nil
	}

	server := &rt.Server{
		ID:        s.UUID.String(),
		Facility:  s.FacilityCode,
		Name:      s.Name,
		UpdatedAt: s.UpdatedAt,
	}

	// TODO: set the most current data from either the inband/outofband NS
	for _, attr := range s.Attributes {
		attr := attr

		switch attr.Namespace {
		case ServerAttributeNSBmcAddress:
			bmcAttr := BMCAttribute{}
			if err := UnpackAttribute(&attr, &bmcAttr); err == nil {
				server.BMCAddress = bmcAttr.Address
			}

		case ServerVendorAttributeNS:
			svmAttr := ServerVendorAttribute{}
			if err := UnpackAttribute(&attr, &svmAttr); err == nil {
				server.Vendor = svmAttr.Vendor
				server.Model = svmAttr.Model
				server.Serial = svmAttr.Serial
			}
		case BiosConfigInbandNS, BiosConfigOutofbandNS:
			bioscfg := map[string]string{}
			if err := UnpackAttribute(&attr, &bioscfg); err == nil {
				server.BIOSCfg = bioscfg
			}
		}
	}

	return server
}
