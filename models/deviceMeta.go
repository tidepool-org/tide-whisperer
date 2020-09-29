package models

import (
	"encoding/json"
	"github.com/mitchellh/mapstructure"
	"strings"
	"time"
	"fmt"
)

type DeviceMeta struct {
	Base                                             `mapstructure:",squash"`

	Status             string                        `mapstructure:"status" pg:"status" json:"status,omitempty"`
	SubType            string                        `mapstructure:"subType" pg:"sub_type" json:"subType,omitempty"`
	Duration           int64                         `mapstructure:"duration" pg:"duration" json:"duration,omitempty"`

	Reason    map[string]interface{}              `mapstructure:"reason" pg:"reason" json:"reason"`
}

func DecodeDeviceMeta(data interface{}) (*DeviceMeta, error) {
	var deviceMeta = DeviceMeta{}

	if decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: StringToTimeHookFuncTimezoneOptional(time.RFC3339),
		Result: &deviceMeta,
	   } ); err == nil {
		if err := decoder.Decode(data); err != nil {
			//fmt.Println("Error decoding device meta: ", err)
			return nil, err
		}

		return &deviceMeta, nil

	} else {
		fmt.Println("Can not create decoder: ", err)
		return nil, nil
	}
}


type DeviceMetaAlias DeviceMeta

func (d DeviceMeta) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewJSONDeviceMeta(d))
}


func NewJSONDeviceMeta(deviceMeta DeviceMeta) JSONDeviceMeta {
	return JSONDeviceMeta{
		DeviceMetaAlias(deviceMeta),
		strings.Trim(deviceMeta.DeviceTime.Format(time.RFC3339), "Z"),
	}
}

type JSONDeviceMeta struct {
	DeviceMetaAlias
	DeviceTime string `json:"deviceTime"`
}
