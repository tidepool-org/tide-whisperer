package models

import (
	"encoding/json"
	"github.com/mitchellh/mapstructure"
	"strings"
	"time"
	"fmt"
)

type CgmSettings struct {
	Base                                             `mapstructure:",squash"`

	TransmitterId      string                        `mapstructure:"transmitterId" pg:"transmitter_id" json:"transmitterId,omitempty"`
	Units             string                         `mapstructure:"units" pg:"units" json:"units,omitempty"`

	LowAlerts    map[string]interface{}           `mapstructure:"lowAlerts" pg:"low_alerts" json:"lowAlerts,omitempty"`

	HighAlerts    map[string]interface{}           `mapstructure:"highAlerts" pg:"high_alerts" json:"highAlerts,omitempty"`

	RateOfChangeAlerts    map[string]interface{}   `mapstructure:"rateOfChangeAlerts" pg:"rate_of_change_alerts" json:"rateOfChangeAlerts,omitempty"`

	OutOfRangeAlerts    map[string]interface{}    `mapstructure:"outOfRangeAlerts" pg:"out_of_range_alerts" json:"outOfRangeAlerts,omitempty"`
}

func DecodeCgmSettings(data interface{}) (*CgmSettings, error) {
	var cgmSettings = CgmSettings{}

	if decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: StringToTimeHookFuncTimezoneOptional(time.RFC3339),
		Result: &cgmSettings,
	   } ); err == nil {
		if err := decoder.Decode(data); err != nil {
			//fmt.Println("Error decoding cgm settings: ", err)
			return nil, err
		}

		return &cgmSettings, nil

	} else {
		fmt.Println("Can not create decoder: ", err)
		return nil, nil
	}
}

type CgmSettingsAlias CgmSettings

func (c CgmSettings) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewJSONCgmSettings(c))
}


func NewJSONCgmSettings(cgbmSettings CgmSettings) JSONCgmSettings {
	return JSONCgmSettings{
		CgmSettingsAlias(cgbmSettings),
		strings.Trim(cgbmSettings.DeviceTime.Format(time.RFC3339), "Z"),
	}
}

type JSONCgmSettings struct {
	CgmSettingsAlias
	DeviceTime string `json:"deviceTime"`
}
