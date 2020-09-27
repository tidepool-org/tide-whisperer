package models

import (
	"github.com/mitchellh/mapstructure"
	"strings"
	"time"
	"fmt"
	"encoding/json"
)

type Cbg struct {
	Base                    `mapstructure:",squash"`

	Value          float64    `mapstructure:"value" pg:"value" json:"value,omitempty"`

	Units          string    `mapstructure:"units" pg:"units" json:"units,omitempty"`
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

type CbgAlias Cbg

func (c Cbg) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewJSONCbg(c))
}


func NewJSONCbg(cbg Cbg) JSONCbg {
	return JSONCbg{
		CbgAlias(cbg),
		strings.Trim(cbg.DeviceTime.Format(time.RFC3339), "Z"),
	}
}

type JSONCbg struct {
	CbgAlias
	DeviceTime string `json:"device_time"`
}

