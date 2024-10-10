package fleetdb

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	bmclibcomm "github.com/bmc-toolbox/common"
	fleetdbapi "github.com/metal-automata/fleetdb/pkg/api/v1"
)

var (
	ErrAttributeFromLabel = errors.New("error creating Attribute from Label")
	ErrLabelFromAttribute = errors.New("error creating Label from Attribute")
	ErrFwSetByVendorModel = errors.New("error identifying firmware set by server vendor, model")
)

func AttributeFromLabels(ns string, labels map[string]string) (*fleetdbapi.Attributes, error) {
	data, err := json.Marshal(labels)
	if err != nil {
		return nil, errors.Wrap(ErrAttributeFromLabel, err.Error())
	}

	return &fleetdbapi.Attributes{
		Namespace: ns,
		Data:      data,
	}, nil
}

// AttributeByNamespace returns the fleetdb attribute in the slice that matches the namespace
func AttributeByNamespace(ns string, attributes []fleetdbapi.Attributes) *fleetdbapi.Attributes {
	for _, attribute := range attributes {
		if attribute.Namespace == ns {
			return &attribute
		}
	}

	return nil
}

// VendorModelFromAttrs unpacks the attributes payload to return the vendor, model attributes for a server
func VendorModelFromAttrs(attrs []fleetdbapi.Attributes) (vendor, model string) {
	attr := AttributeByNamespace(ServerVendorAttributeNS, attrs)
	if attr == nil {
		return "", ""
	}

	data := map[string]string{}
	if err := json.Unmarshal(attr.Data, &data); err != nil {
		return "", ""
	}

	return bmclibcomm.FormatVendorName(data["vendor"]), bmclibcomm.FormatProductName(data["model"])
}

// FirmwareSetIDByVendorModel returns the firmware set ID matched by the vendor, model attributes
func FirmwareSetIDByVendorModel(ctx context.Context, vendor, model string, client *fleetdbapi.Client) (uuid.UUID, error) {
	fwSet, err := FirmwareSetByVendorModel(ctx, vendor, model, client)
	if err != nil {
		return uuid.Nil, err
	}

	log.Printf(
		"fw sets identified for vendor: %s, model: %s, fwset: %s\n",
		vendor,
		model,
		fwSet[0].UUID.String(),
	)

	return fwSet[0].UUID, nil
}

// FirmwareSetByVendorModel returns the firmware set matched by the vendor, model attributes
func FirmwareSetByVendorModel(ctx context.Context, vendor, model string, client *fleetdbapi.Client) ([]fleetdbapi.ComponentFirmwareSet, error) {
	vendor = strings.TrimSpace(vendor)
	if vendor == "" {
		return []fleetdbapi.ComponentFirmwareSet{}, errors.Wrap(
			ErrFwSetByVendorModel,
			"got empty vendor attribute",
		)
	}

	model = strings.TrimSpace(model)
	if model == "" {
		return []fleetdbapi.ComponentFirmwareSet{}, errors.Wrap(
			ErrFwSetByVendorModel,
			"got empty model attribute",
		)
	}

	// ?attr=sh.hollow.firmware_set.labels~vendor~eq~dell&attr=sh.hollow.firmware_set.labels~model~eq~r750&attr=sh.hollow.firmware_set.labels~latest~eq~false
	// list latest, default firmware sets by vendor, model attributes
	fwSetListparams := &fleetdbapi.ComponentFirmwareSetListParams{
		AttributeListParams: []fleetdbapi.AttributeListParams{
			{
				Namespace: FirmwareSetAttributeNS,
				Keys:      []string{"vendor"},
				Operator:  "eq",
				Value:     strings.ToLower(vendor),
			},
			{
				Namespace: FirmwareSetAttributeNS,
				Keys:      []string{"model"},
				Operator:  "like",
				Value:     strings.ToLower(model),
			},
			{
				Namespace: FirmwareSetAttributeNS,
				Keys:      []string{"latest"}, // latest indicates the most current revision of the firmware set.
				Operator:  "eq",
				Value:     "true",
			},
			{
				Namespace: FirmwareSetAttributeNS,
				Keys:      []string{"default"}, // default indicates the firmware set does not belong to an org/project.
				Operator:  "eq",
				Value:     "true",
			},
		},
	}

	fwSet, _, err := client.ListServerComponentFirmwareSet(ctx, fwSetListparams)
	if err != nil {
		return []fleetdbapi.ComponentFirmwareSet{}, errors.Wrap(ErrFwSetByVendorModel, err.Error())
	}

	if len(fwSet) == 0 {
		return []fleetdbapi.ComponentFirmwareSet{}, errors.Wrap(
			ErrFwSetByVendorModel,
			fmt.Sprintf("no fw sets identified for vendor: %s, model: %s", vendor, model),
		)
	}

	return fwSet, nil
}

// FirmwareSetByVendorModel returns the firmware set matched by the vendor, model and labels attributes
func FirmwareSetByLabels(ctx context.Context, vendor, model string, labels map[string]string, client *fleetdbapi.Client) ([]fleetdbapi.ComponentFirmwareSet, error) {
	var fwSetListparams *fleetdbapi.ComponentFirmwareSetListParams
	labelParts := make([]string, 0)
	for key, value := range labels {
		labelParts = append(labelParts, fmt.Sprintf("%s=%s", key, value))
	}
	labelsParams := strings.Join(labelParts, ",")

	// labels select requires len(AttributeListParams) = 0
	fwSetListparams = &fleetdbapi.ComponentFirmwareSetListParams{
		Model:  model,
		Vendor: vendor,
		Labels: labelsParams,
	}

	fwSet, _, err := client.ListServerComponentFirmwareSet(ctx, fwSetListparams)
	if err != nil {
		return []fleetdbapi.ComponentFirmwareSet{}, errors.Wrap(ErrFwSetByVendorModel, err.Error())
	}

	if len(fwSet) == 0 {
		return []fleetdbapi.ComponentFirmwareSet{}, errors.Wrap(
			ErrFwSetByVendorModel,
			fmt.Sprintf("no fw sets identified for vendor: %s, model: %s", vendor, model),
		)
	}

	return fwSet, nil
}
