package models

import (
	"encoding/json"
	"github.com/mitchellh/mapstructure"
	"time"
	"fmt"
)

type DeviceMeta struct {
	Base                                             `mapstructure:",squash"`

	Status             string                        `mapstructure:"status" pg:"status" json:"status,omitempty"`
	SubType            string                        `mapstructure:"subType" pg:"sub_type" json:"subType,omitempty"`
	Duration           int64                         `mapstructure:"duration" pg:"duration" json:"duration,omitempty"`

	ReasonMap    map[string]interface{}              `mapstructure:"reason" pg:"-"`
	ReasonJson   string                              `pg:"reason" json:"reason,omitempty"`

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

		reasonByteArray, err := json.Marshal(deviceMeta.ReasonMap)
		deviceMeta.ReasonJson = string(reasonByteArray)
		if err != nil {
			fmt.Println("Error encoding reason json: ", err)
			return nil, err
		}

		return &deviceMeta, nil

	} else {
		fmt.Println("Can not create decoder: ", err)
		return nil, nil
	}
}
