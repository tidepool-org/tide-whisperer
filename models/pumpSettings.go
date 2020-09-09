package models

import (
	"encoding/json"
	"github.com/mitchellh/mapstructure"
	"time"
	"fmt"
)

type PumpSettings struct {
	Base                                             `mapstructure:",squash"`

	ActiveSchedule          string                      `mapstructure:"activeSchedule" pg:"active_schedule"`

	BasalSchedulesMap       map[string]interface{}      `mapstructure:"basalSchedules" pg:"-"`
	BasalSchedulesJson      string                      `pg:"basal_schedules"`

	BgTargetMap             []interface{}      `mapstructure:"bgTarget" pg:"-"`
	BgTargetJson            string                      `pg:"bg_target"`

	CarbRatioMap            []interface{}      `mapstructure:"carbRatio" pg:"-"`
	CarbRatioJson           string                      `pg:"carb_ratio"`

	InsulinSensitivityMap   []interface{}      `mapstructure:"insulinSensitivity" pg:"-"`
	InsulinSensitivityJson  string                      `pg:"insulin_sensitivity"`

	unitsMap                map[string]interface{}      `mapstructure:"units" pg:"-"`
	unitsJson               string                      `pg:"units"`
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

		basalSchedulesByteArray, err := json.Marshal(pumpSettings.BasalSchedulesMap)
		pumpSettings.BasalSchedulesJson = string(basalSchedulesByteArray)
		if err != nil {
			fmt.Println("Error encoding Basal Schedules json: ", err)
			return nil, err
		}

		bgTargetMapByteArray, err := json.Marshal(pumpSettings.BgTargetMap)
		pumpSettings.BgTargetJson = string(bgTargetMapByteArray)
		if err != nil {
			fmt.Println("Error encoding Bg Target json: ", err)
			return nil, err
		}

		carbRatioMapByteArray, err := json.Marshal(pumpSettings.CarbRatioMap)
		pumpSettings.CarbRatioJson = string(carbRatioMapByteArray)
		if err != nil {
			fmt.Println("Error encoding carb ration json: ", err)
			return nil, err
		}

		insulinSensitivityByteArray, err := json.Marshal(pumpSettings.InsulinSensitivityMap)
		pumpSettings.InsulinSensitivityJson = string(insulinSensitivityByteArray)
		if err != nil {
			fmt.Println("Error encoding insulin sensitivity json: ", err)
			return nil, err
		}

		unitsMapByteArray, err := json.Marshal(pumpSettings.unitsMap)
		pumpSettings.unitsJson = string(unitsMapByteArray)
		if err != nil {
			fmt.Println("Error encoding units json: ", err)
			return nil, err
		}

		return &pumpSettings, nil

	} else {
		fmt.Println("Can not create decoder: ", err)
		return nil, err
	}
}
