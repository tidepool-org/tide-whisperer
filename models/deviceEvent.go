package models

import (
	"encoding/json"
	"github.com/mitchellh/mapstructure"
	"strings"
	"time"
	"fmt"
)

type DeviceEvent struct {
	Base                                         `mapstructure:",squash"`

	SubType      string                          `mapstructure:"subType" pg:"sub_type" json:"subType,omitempty"`
	Units        string                          `mapstructure:"units" pg:"units" json:"units,omitempty"`

	Value        float64                         `mapstructure:"value" pg:"value" json:"value,omitempty"`

	duration     int64                           `mapstructure:"duration" pg:"duration" json:"duration,omitempty"`

	Reason       map[string]interface{}          `mapstructure:"reason" pg:"reason" json:"reason,omitempty"`

	PrimeTarget  string                          `mapstructure:"primeTarget" pg:"prime_target" json:"primeTarget,omitempty"`
	Volume       float64                         `mapstructure:"volume" pg:"volume" json:"volume,omitempty"`
}

func DecodeDeviceEvent(data interface{}) (*DeviceEvent, error) {
	var deviceEvent = DeviceEvent{}

	if decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: StringToTimeHookFuncTimezoneOptional(time.RFC3339),
		Result: &deviceEvent,
	   } ); err == nil {
		if err := decoder.Decode(data); err != nil {
			//fmt.Println("Error decoding device event: ", err)
			return nil, err
		}

		return &deviceEvent, nil

	} else {
		fmt.Println("Can not create decoder: ", err)
		return nil, nil
	}
}


type DeviceEventAlias DeviceEvent

func (d DeviceEvent) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewJSONDeviceEvent(d))
}


func NewJSONDeviceEvent(deviceEvent DeviceEvent) JSONDeviceEvent {
	return JSONDeviceEvent{
		DeviceEventAlias(deviceEvent),
		strings.Trim(deviceEvent.DeviceTime.Format(time.RFC3339), "Z"),
	}
}

type JSONDeviceEvent struct {
	DeviceEventAlias
	DeviceTime string `json:"deviceTime"`
}
