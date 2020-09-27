package models

import (
	"github.com/mitchellh/mapstructure"
	"strings"
	"time"
	"fmt"
	"encoding/json"
)

type Bolus struct {
	Base                    `mapstructure:",squash"`

	Normal         float64   `mapstructure:"normal" pg:"normal" json:"normal,omitempty"`

	SubType        string    `mapstructure:"subType" pg:"sub_type" json:"subType,omitempty"`
}

func DecodeBolus(data interface{}) (*Bolus, error) {
	var bolus = Bolus{}

	if decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: StringToTimeHookFuncTimezoneOptional(time.RFC3339),
		Result: &bolus,
	   } ); err == nil {
		if err := decoder.Decode(data); err != nil {
			//fmt.Println("Error decoding bolus: ", err)
			return nil, err
		}

		return &bolus, nil

	} else {
		fmt.Println("Can not create decoder: ", err)
		return nil, nil
	}
}

type BolusAlias Bolus

func (b Bolus) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewJSONBolus(b))
}


func NewJSONBolus(bolus Bolus) JSONBolus {
	return JSONBolus{
		BolusAlias(bolus),
		strings.Trim(bolus.DeviceTime.Format(time.RFC3339), "Z"),
	}
}

type JSONBolus struct {
	BolusAlias
	DeviceTime string `json:"deviceTime"`
}