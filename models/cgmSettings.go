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

	LowAlertsMap    map[string]interface{}           `mapstructure:"lowAlerts" pg:"-"`
	LowAlertsJson   string                           `pg:"low_alerts"`

	HighAlertsMap    map[string]interface{}           `mapstructure:"highAlerts" pg:"-"`
	HighAlertsJson   string                           `pg:"high_alerts"`

	RateOfChangeAlertsMap    map[string]interface{}   `mapstructure:"rateOfChangeAlerts" pg:"-"`
	RateOfChangeAlertsJson   string                   `pg:"rate_of_change_alerts"`

	OutOfRangeAlertsMap    map[string]interface{}    `mapstructure:"outOfRangeAlerts" pg:"-"`
	OutOfRangeAlertsJson   string                    `pg:"out_of_range_alerts"`
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

		lowAlertsByteArray, err := json.Marshal(cgmSettings.LowAlertsMap)
		cgmSettings.LowAlertsJson = string(lowAlertsByteArray)

		highAlertsByteArray, err := json.Marshal(cgmSettings.HighAlertsMap)
		cgmSettings.HighAlertsJson = string(highAlertsByteArray)

		rateOfChangeAlertsByteArray, err := json.Marshal(cgmSettings.RateOfChangeAlertsMap)
		cgmSettings.RateOfChangeAlertsJson = string(rateOfChangeAlertsByteArray)

		outOfRangeAlertsByteArray, err := json.Marshal(cgmSettings.OutOfRangeAlertsMap)
		cgmSettings.OutOfRangeAlertsJson = string(outOfRangeAlertsByteArray)

		if err != nil {
			fmt.Println("Error encoding reason json: ", err)
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
