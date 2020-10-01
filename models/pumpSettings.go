package models

import (
	"encoding/json"
	"github.com/mitchellh/mapstructure"
	"strings"
	"time"
	"fmt"
)

type PumpSettings struct {
	Base                                             `mapstructure:",squash"`

	ActiveSchedule          string                      `mapstructure:"activeSchedule" pg:"active_schedule" json:"activeSchedule,omitempty"`

	BasalSchedules       interface{}      `mapstructure:"basalSchedules" pg:"basal_schedules" json:"basalSchedules,omitempty"`

	BgTargets             []interface{}      `mapstructure:"bgTargets" pg:"bg_targets" json:"bgTargets,omitempty"`

	CarbRatios            []interface{}      `mapstructure:"carbRatio" pg:"carb_ratios" json:"carbRatios,omitempty"`

	InsulinSensitivities   []interface{}      `mapstructure:"insulinSensitivities" pg:"insulin_sensitivities" json:"insulinSensitivities,omitempty"`

	units                interface{}      `mapstructure:"units" pg:"units" json:"units,omitempty"`
}

func DecodePumpSettings(data interface{}) (*PumpSettings, error) {
	var pumpSettings = PumpSettings{}

	if decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: StringToTimeHookFuncTimezoneOptional(time.RFC3339),
		Result: &pumpSettings,
	   } ); err == nil {
		if err := decoder.Decode(data); err != nil {
			//fmt.Println("Error decoding pump settings: ", err)
			return nil, err
		}

		return &pumpSettings, nil

	} else {
		fmt.Println("Can not create decoder: ", err)
		return nil, err
	}
}

type PumpSettingsAlias PumpSettings

func (p PumpSettings) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewJSONPumpSettings(p))
}


func NewJSONPumpSettings(pumpSettings PumpSettings) JSONPumpSettings {
	return JSONPumpSettings{
		PumpSettingsAlias(pumpSettings),
		strings.Trim(pumpSettings.DeviceTime.Format(time.RFC3339), "Z"),
	}
}

type JSONPumpSettings struct {
	PumpSettingsAlias
	DeviceTime string `json:"deviceTime"`
}
