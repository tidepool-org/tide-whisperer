package models

import (
	"github.com/mitchellh/mapstructure"
	"time"
	"fmt"
)

type Smbg struct {
	Base                    `mapstructure:",squash"`

	Units          string    `mapstructure:"units" pg:"units"`

	value          float64    `mapstructure:"value" pg:"value"`
}

func DecodeSmbg(data interface{}) (*Smbg, error) {
	var smbg = Smbg{}

	if decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: StringToTimeHookFuncTimezoneOptional(time.RFC3339),
		Result: &smbg,
	   } ); err == nil {
		if err := decoder.Decode(data); err != nil {
			//fmt.Println("Error decoding smbg: ", err)
			return nil, err
		}

		return &smbg, nil

	} else {
		fmt.Println("Can not create decoder: ", err)
		return nil, nil
	}
}
