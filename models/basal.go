package models

import (
	"github.com/mitchellh/mapstructure"
	"strings"
	"time"
	"fmt"
	"encoding/json"
)

type Basal struct {
	Base                      `mapstructure:",squash"`

	DeliveryType      string   `mapstructure:"deliveryType,omitempty" pg:"delivery_type" json:"deliveryType,omitempty"`
	Duration          int64    `mapstructure:"duration,omitempty" pg:"duration" json:"duration,omitempty"`
	ExpectedDuration  int64    `mapstructure:"expectedDuration,omitempty" pg:"expected_duration" json:"expectedDuration,omitempty"`
	Rate              float64  `mapstructure:"rate,omitempty" pg:"rate" json:"rate,omitempty"`
	Percent           float64  `mapstructure:"percent,omitempty" pg:"percent" json:"percent,omitempty"`
	ScheduleName      string   `mapstructure:"scheduleName,omitempty" pg:"schedule_name" json:"scheduleName,omitempty"`
}

func DecodeBasal(data interface{}) (*Basal, error)  {
	var basal = Basal{}

	if decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: StringToTimeHookFuncTimezoneOptional(time.RFC3339),
		Result: &basal,
	   } ); err == nil {
		if err := decoder.Decode(data); err != nil {
			//fmt.Println("Error decoding basal: ", err)
			return nil, err
		}

		return &basal, nil

	} else {
		fmt.Println("Can not create decoder: ", err)
		return nil, err
	}
}

type BasalAlias Basal

func (b Basal) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewJSONBasal(b))
}


func NewJSONBasal(basal Basal) JSONBasal {
	return JSONBasal{
		BasalAlias(basal),
		strings.Trim(basal.DeviceTime.Format(time.RFC3339), "Z"),
	}
}

type JSONBasal struct {
	BasalAlias
	DeviceTime string `json:"deviceTime"`
}
