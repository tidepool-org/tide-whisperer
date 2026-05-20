package transform

import (
	"bytes"
	_ "embed"
	"encoding/csv"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
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

	deviceModelIndex := slices.IndexFunc(records[0], func(c string) bool {
		return strings.EqualFold(c, "deviceModel")
	})
	if deviceModelIndex == -1 {
		return nil, fmt.Errorf("unable to find deviceModel header in csv")
	}
	friendlyNameIndex := slices.IndexFunc(records[0], func(c string) bool {
		return strings.EqualFold(c, "friendlyName")
	})
	if friendlyNameIndex == -1 {
		return nil, fmt.Errorf("unable to find friendlyName header in csv")
	}

	deviceModelToName := map[string]string{}
	for i := 1; i < len(records); i++ {
		record := records[i]
		if deviceModelIndex >= len(record) || friendlyNameIndex >= len(record) {
			return nil, fmt.Errorf("unable to find device mapping at row %v", i+1)
		}

		deviceModel := strings.TrimSpace(record[deviceModelIndex])
		deviceNiceName := strings.TrimSpace(record[friendlyNameIndex])
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

	name, exists := findPrefixedDeviceName(deviceModel, d.deviceModelToName)
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

// findPrefixedDeviceName searches the device name to friendly name mapping
// of deviceNames for any key that is a (not necessarily proper) prefix of the
// input deviceModel. This is to handle cases where device models contain
// miscellaneous suffixes such as hashes.
func findPrefixedDeviceName(deviceModel string, deviceNames map[string]string) (friendlyName string, found bool) {
	deviceModel = strings.ToLower(strings.TrimSpace(deviceModel))
	// We could also just keep a sorted slice by deviceModel slice and do a
	// binary search to find a device model key that is a prefix of the input
	// deviceModel name but since there are only a couple hundred entries, not
	// going to bother.

	var longestDeviceName string
	for modelName := range deviceNames {
		// We have to loop through all names because there may exist names in
		// deviceNames that are prefixes of each other. For example, given
		// deviceNames of {"FreeStyle Libre 3": "Xyz", "FreeStyle Libre 3 Plus":
		// "Abc"} and a deviceModel of "FreeStyle Libre 3 Plus some-suffix-here" we
		// do not want to prematurely return the value mapped for "FreeStyle Libre
		// 3" (Of couse now the binary search looks more appealing as it can handle
		// this).
		if strings.HasPrefix(deviceModel, strings.ToLower(modelName)) && (!found || utf8.RuneCountInString(modelName) > utf8.RuneCountInString(longestDeviceName)) {
			found = true
			longestDeviceName = modelName
		}
	}
	if found {
		return deviceNames[longestDeviceName], found
	}
	return "", false
}
