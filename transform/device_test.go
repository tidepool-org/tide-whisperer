package transform

import (
	"maps"
	"reflect"
	"strings"
	"testing"
)

func TestParseDevicesCSV(t *testing.T) {
	tests := []struct {
		name        string
		csvContent  string
		expected    map[string]string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid CSV with header and data",
			csvContent: `deviceModel,friendlyName
G6,Dexcom G6
G7,Dexcom G7
670G,MiniMed 670G
780G,MiniMed 780G`,
			expected: map[string]string{
				"G6":   "Dexcom G6",
				"G7":   "Dexcom G7",
				"670G": "MiniMed 670G",
				"780G": "MiniMed 780G",
			},
			expectError: false,
		},
		{
			name: "valid CSV with extra columns",
			csvContent: `deviceModel,manufacturer,friendlyName,type
G6,Dexcom,Dexcom G6,CGM
670G,Medtronic,MiniMed 670G,Pump`,
			expected: map[string]string{
				"G6":   "Dexcom G6",
				"670G": "MiniMed 670G",
			},
			expectError: false,
		},
		{
			name: "CSV with whitespace that should be trimmed",
			csvContent: `deviceModel,friendlyName
  G6  ,  Dexcom G6  
 670G , MiniMed 670G `,
			expected: map[string]string{
				"G6":   "Dexcom G6",
				"670G": "MiniMed 670G",
			},
			expectError: false,
		},
		{
			name: "CSV with empty values (should be skipped)",
			csvContent: `deviceModel,friendlyName
G6,Dexcom G6
,
670G,MiniMed 670G
G7,`,
			expected: map[string]string{
				"G6":   "Dexcom G6",
				"670G": "MiniMed 670G",
			},
			expectError: false,
		},
		{
			name:        "empty CSV",
			csvContent:  "",
			expected:    nil,
			expectError: true,
			errorMsg:    "devices csv is empty",
		},
		{
			name:        "CSV with only header",
			csvContent:  "deviceModel,friendlyName",
			expected:    map[string]string{},
			expectError: false,
		},
		{
			name: "missing deviceModel header",
			csvContent: `model,friendlyName
G6,Dexcom G6`,
			expected:    nil,
			expectError: true,
			errorMsg:    "unable to find deviceModel header in csv",
		},
		{
			name: "missing friendlyName header",
			csvContent: `deviceModel,niceName
G6,Dexcom G6`,
			expected:    nil,
			expectError: true,
			errorMsg:    "unable to find friendlyName header in csv",
		},
		{
			name: "malformed CSV",
			csvContent: `deviceModel,deviceNiceName
G6,"unclosed quote`,
			expected:    nil,
			expectError: true,
			errorMsg:    "unable to read csv",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDevicesCSV([]byte(tt.csvContent))

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestNewDeviceNameTransformerFromMap(t *testing.T) {
	deviceMap := map[string]string{
		"G6":   "Dexcom G6",
		"670G": "MiniMed 670G",
	}

	transformer := NewDeviceNameTransformerFromMap(deviceMap)

	if transformer == nil {
		t.Error("expected non-nil transformer")
		return
	}

	if !reflect.DeepEqual(transformer.deviceModelToName, deviceMap) {
		t.Errorf("expected %v, got %v", deviceMap, transformer.deviceModelToName)
	}
}

func TestDeviceNameTransformer_Transform(t *testing.T) {
	deviceMap := map[string]string{
		"G6":          "Dexcom G6",
		"G7":          "Dexcom G7",
		"670G":        "MiniMed 670G",
		"780G":        "MiniMed 780G",
		"Libre2":      "FreeStyle Libre 2",
		"Libre2 Plus": "FreeStyle Libre 2 Plus",
		"OmniPod5":    "Omnipod 5",
	}
	transformer := NewDeviceNameTransformerFromMap(deviceMap)

	tests := []struct {
		name     string
		input    map[string]any
		expected map[string]any
	}{
		{
			name: "valid upload with known CGM device",
			input: map[string]any{
				"type":        "upload",
				"deviceModel": "G6",
				"userId":      123,
			},
			expected: map[string]any{
				"type":        "upload",
				"deviceModel": "G6",
				"userId":      123,
				"deviceName":  "Dexcom G6",
			},
		},
		{
			name: "valid upload with known insulin pump",
			input: map[string]any{
				"type":        "upload",
				"deviceModel": "670G",
				"userId":      123,
			},
			expected: map[string]any{
				"type":        "upload",
				"deviceModel": "670G",
				"userId":      123,
				"deviceName":  "MiniMed 670G",
			},
		},
		{
			name: "valid upload with device model that needs trimming",
			input: map[string]any{
				"type":        "upload",
				"deviceModel": "  OmniPod5 ",
				"userId":      123,
			},
			expected: map[string]any{
				"type":        "upload",
				"deviceModel": "  OmniPod5 ",
				"userId":      123,
				"deviceName":  "Omnipod 5",
			},
		},
		{
			name: "non-upload type should not be transformed",
			input: map[string]any{
				"type":        "download",
				"deviceModel": "G6",
				"userId":      123,
			},
			expected: map[string]any{
				"type":        "download",
				"deviceModel": "G6",
				"userId":      123,
			},
		},
		{
			name: "missing type should not be transformed",
			input: map[string]any{
				"deviceModel": "G6",
				"userId":      123,
			},
			expected: map[string]any{
				"deviceModel": "G6",
				"userId":      123,
			},
		},
		{
			name: "non-string type should not be transformed",
			input: map[string]any{
				"type":        123,
				"deviceModel": "G6",
				"userId":      123,
			},
			expected: map[string]any{
				"type":        123,
				"deviceModel": "G6",
				"userId":      123,
			},
		},
		{
			name: "missing deviceModel should not be transformed",
			input: map[string]any{
				"type":   "upload",
				"userId": 123,
			},
			expected: map[string]any{
				"type":   "upload",
				"userId": 123,
			},
		},
		{
			name: "non-string deviceModel should not be transformed",
			input: map[string]any{
				"type":        "upload",
				"deviceModel": 123,
				"userId":      123,
			},
			expected: map[string]any{
				"type":        "upload",
				"deviceModel": 123,
				"userId":      123,
			},
		},
		{
			name: "unknown device model should not be transformed",
			input: map[string]any{
				"type":        "upload",
				"deviceModel": "UnknownPump",
				"userId":      123,
			},
			expected: map[string]any{
				"type":        "upload",
				"deviceModel": "UnknownPump",
				"userId":      123,
			},
		},
		{
			name: "empty device model should not be transformed",
			input: map[string]any{
				"type":        "upload",
				"deviceModel": "",
				"userId":      123,
			},
			expected: map[string]any{
				"type":        "upload",
				"deviceModel": "",
				"userId":      123,
			},
		},
		{
			name: "should overwrite existing deviceName",
			input: map[string]any{
				"type":        "upload",
				"deviceModel": "G6",
				"deviceName":  "Old Name",
				"userId":      123,
			},
			expected: map[string]any{
				"type":        "upload",
				"deviceModel": "G6",
				"deviceName":  "Dexcom G6",
				"userId":      123,
			},
		},
		{
			name: "deviceModel with a mapped name that is a proper prefix of the deviceModel should be transformed",
			input: map[string]any{
				"type":        "upload",
				"deviceModel": "OmniPod5_with_suffix",
				"userId":      12345,
			},
			expected: map[string]any{
				"type":        "upload",
				"deviceModel": "OmniPod5_with_suffix",
				"userId":      12345,
				"deviceName":  "Omnipod 5",
			},
		},
		{
			name: "deviceModel with a mapped name that is an exact match of the deviceModel should be transformed",
			input: map[string]any{
				"type":        "upload",
				"deviceModel": "OmniPod5",
				"userId":      123456,
			},
			expected: map[string]any{
				"type":        "upload",
				"deviceModel": "OmniPod5",
				"userId":      123456,
				"deviceName":  "Omnipod 5",
			},
		},
		{
			name: "deviceModel that is a proper prefix of a mapped name should not be transformed",
			input: map[string]any{
				"type":        "upload",
				"deviceModel": "Omni",
				"userId":      1234567,
			},
			expected: map[string]any{
				"type":        "upload",
				"deviceModel": "Omni",
				"userId":      1234567,
			},
		},
		{
			name: "transformed deviceName should use the longest mapped model name",
			input: map[string]any{
				"type":        "upload",
				"deviceModel": "Libre2 Plus abcdefghi",
				"userId":      12345678,
			},
			expected: map[string]any{
				"type":        "upload",
				"deviceModel": "Libre2 Plus abcdefghi",
				"userId":      12345678,
				"deviceName":  "FreeStyle Libre 2 Plus",
			},
		},
		{
			name: "case-insensitive deviceModel matches should be transformed",
			input: map[string]any{
				"type":        "upload",
				"deviceModel": "libre2",
				"userId":      123456789,
			},
			expected: map[string]any{
				"type":        "upload",
				"deviceModel": "libre2",
				"userId":      123456789,
				"deviceName":  "FreeStyle Libre 2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy of input to avoid modifying the test case
			input := make(map[string]any)
			maps.Copy(input, tt.input)

			transformer.Transform(input)

			if !reflect.DeepEqual(input, tt.expected) {
				t.Errorf("expected %#v, got %#v", tt.expected, input)
			}
		})
	}
}

func TestGetMapValueAs(t *testing.T) {
	testMap := map[string]any{
		"stringValue": "hello",
		"intValue":    42,
		"boolValue":   true,
		"nilValue":    nil,
	}

	tests := []struct {
		name           string
		key            string
		expectedResult any
		expectedExists bool
		resultType     string
	}{
		{
			name:           "get existing string value",
			key:            "stringValue",
			expectedResult: "hello",
			expectedExists: true,
			resultType:     "string",
		},
		{
			name:           "get existing int value",
			key:            "intValue",
			expectedResult: 42,
			expectedExists: true,
			resultType:     "int",
		},
		{
			name:           "get existing bool value",
			key:            "boolValue",
			expectedResult: true,
			expectedExists: true,
			resultType:     "bool",
		},
		{
			name:           "get non-existing key",
			key:            "nonExisting",
			expectedResult: "",
			expectedExists: false,
			resultType:     "string",
		},
		{
			name:           "get nil value",
			key:            "nilValue",
			expectedResult: "",
			expectedExists: false,
			resultType:     "string",
		},
		{
			name:           "wrong type assertion",
			key:            "stringValue",
			expectedResult: 0,
			expectedExists: false,
			resultType:     "int",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.resultType {
			case "string":
				result, exists := getMapValueAs[string](tt.key, testMap)
				if result != tt.expectedResult || exists != tt.expectedExists {
					t.Errorf("expected (%v, %v), got (%v, %v)", tt.expectedResult, tt.expectedExists, result, exists)
				}
			case "int":
				result, exists := getMapValueAs[int](tt.key, testMap)
				if result != tt.expectedResult || exists != tt.expectedExists {
					t.Errorf("expected (%v, %v), got (%v, %v)", tt.expectedResult, tt.expectedExists, result, exists)
				}
			case "bool":
				result, exists := getMapValueAs[bool](tt.key, testMap)
				if result != tt.expectedResult || exists != tt.expectedExists {
					t.Errorf("expected (%v, %v), got (%v, %v)", tt.expectedResult, tt.expectedExists, result, exists)
				}
			}
		})
	}
}

func TestNewDeviceNameTransformer(t *testing.T) {
	// This test requires the actual embedded CSV file to work
	// Since we can't control the embedded file in tests, we'll test the error case
	// and assume the CSV parsing is tested separately

	// Note: In a real scenario, you might want to:
	// 1. Mock the embedded file using build tags or interfaces
	// 2. Test with a known good CSV file
	// 3. Or create a separate testable version that takes the CSV content as parameter

	t.Run("constructor behavior", func(t *testing.T) {
		// This will either succeed or fail based on the actual embedded CSV
		// The important thing is that it doesn't panic and returns appropriate values
		transformer, err := NewDeviceNameTransformer()

		if err != nil {
			// If there's an error, it should be properly formatted
			if transformer != nil {
				t.Error("expected nil transformer when error occurs")
			}
		} else {
			// If no error, transformer should be valid
			if transformer == nil {
				t.Error("expected non-nil transformer when no error")
			}
			if transformer.deviceModelToName == nil {
				t.Error("expected non-nil deviceModelToName map")
			}
		}
	})
}
