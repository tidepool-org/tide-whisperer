package models

import (
	"github.com/mitchellh/mapstructure"
	"time"
	"fmt"
)

type Bolus struct {
	Base                    `mapstructure:",squash"`

	Normal         float64   `mapstructure:"normal" pg:"normal"`

	SubType        string    `mapstructure:"subType" pg:"sub_type"`
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
