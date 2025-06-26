package transform

import (
	"bytes"
	_ "embed"
	"encoding/csv"
	"fmt"
	"slices"
	"strings"
)

//go:embed device_names.csv
var deviceNamesCSV []byte

type DeviceNameTransformer struct {
	deviceModelToName map[string]string
}

func NewDeviceNameTransformer() (*DeviceNameTransformer, error) {
	devices, err := ParseDevicesCSV(deviceNamesCSV)
	if err != nil {
		return nil, fmt.Errorf("unable to parse devices csv: %w", err)
	}

	return &DeviceNameTransformer{
		deviceModelToName: devices,
	}, nil
}

func ParseDevicesCSV(content []byte) (map[string]string, error) {
	csvReader := csv.NewReader(bytes.NewReader(content))
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("unable to read csv: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("devices csv is empty")
	}

	deviceModelIndex := slices.Index(records[0], "deviceModel")
	if deviceModelIndex == -1 {
		return nil, fmt.Errorf("unable to find deviceModel header in csv")
	}
	deviceNiceNameIndex := slices.Index(records[0], "deviceNiceName")
	if deviceNiceNameIndex == -1 {
		return nil, fmt.Errorf("unable to find deviceNiceName header in csv")
	}

	deviceModelToName := map[string]string{}
	for i := 1; i < len(records); i++ {
		record := records[i]
		if deviceModelIndex >= len(record) || deviceNiceNameIndex >= len(record) {
			return nil, fmt.Errorf("unable to find device mapping at row %v", i)
		}

		deviceModel := strings.TrimSpace(record[deviceModelIndex])
		deviceNiceName := strings.TrimSpace(record[deviceNiceNameIndex])
		if deviceModel == "" || deviceNiceName == "" {
			continue
		}

		deviceModelToName[deviceModel] = deviceNiceName
	}

	return deviceModelToName, nil
}

func NewDeviceNameTransformerFromMap(deviceModelToName map[string]string) *DeviceNameTransformer {
	return &DeviceNameTransformer{deviceModelToName: deviceModelToName}
}

func (d *DeviceNameTransformer) Transform(datum map[string]any) {
	datumType, exists := getMapValueAs[string]("type", datum)
	if !exists || datumType != "upload" {
		return
	}

	deviceModel, exists := getMapValueAs[string]("deviceModel", datum)
	if !exists {
		return
	}

	name, exists := d.deviceModelToName[strings.TrimSpace(deviceModel)]
	if !exists || name == "" {
		return
	}

	datum["deviceName"] = name
}

func getMapValueAs[T any](key string, datum map[string]any) (result T, exists bool) {
	var value any

	value, exists = datum[key]
	if !exists {
		return
	}

	result, exists = value.(T)
	return
}
