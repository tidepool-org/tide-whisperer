package models

import (
	"github.com/mitchellh/mapstructure"
	"strings"
	"time"
	"fmt"
	"encoding/json"
)

type Smbg struct {
	Base                    `mapstructure:",squash"`

	Units          string    `mapstructure:"units" pg:"units" json:"units"`

	Value          float64    `mapstructure:"value" pg:"value" json:"value"`
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

type SmbgAlias Smbg

func (s Smbg) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewJSONSmbg(s))
}


func NewJSONSmbg(smbg Smbg) JSONSmbg {
	return JSONSmbg{
		SmbgAlias(smbg),
		strings.Trim(smbg.DeviceTime.Format(time.RFC3339), "Z"),
	}
}

type JSONSmbg struct {
	SmbgAlias
	DeviceTime string `json:"deviceTime"`
}
