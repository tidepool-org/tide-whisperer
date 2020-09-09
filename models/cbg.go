package models

import (
	"github.com/mitchellh/mapstructure"
	"time"
	"fmt"
)

type Cbg struct {
	Base                    `mapstructure:",squash"`

	Value          float64    `mapstructure:"value" pg:"value"`

	Units          string    `mapstructure:"units" pg:"units"`
}

func DecodeCbg(data interface{}) (*Cbg, error) {
	var cbg = Cbg{}

	if decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: StringToTimeHookFuncTimezoneOptional(time.RFC3339),
		Result: &cbg,
	   } ); err == nil {
		if err := decoder.Decode(data); err != nil {
			fmt.Println("Error decoding cbg: ", err)
			return nil, err
		}

		return &cbg, nil

	} else {
		fmt.Println("Can not create decoder: ", err)
		return nil, err
	}
}
