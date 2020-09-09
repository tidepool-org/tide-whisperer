package models

import (
	"encoding/json"
	"github.com/mitchellh/mapstructure"
	"time"
	"fmt"
)

type PhysicalActivity struct {
	Base                                           `mapstructure:",squash"`

	DurationMap    map[string]interface{}         `mapstructure:"duration" pg:"-"`
	DurationJson   string                         `pg:"duration"`

	DistanceMap    map[string]interface{}         `mapstructure:"distance" pg:"-"`
	DistanceJson   string                         `pg:"distance"`

	EnergyMap      map[string]interface{}         `mapstructure:"energy" pg:"-"`
	EnergyJson     string                         `pg:"energy"`

	Name           string                         `mapstructure:"name" pg:"name"`
}

func DecodePhysicalActivity(data interface{}) (*PhysicalActivity, error) {
	var physicalActivity = PhysicalActivity{}

	if decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: StringToTimeHookFuncTimezoneOptional(time.RFC3339),
		Result: &physicalActivity,
	   } ); err == nil {
		if err := decoder.Decode(data); err != nil {
			//fmt.Println("Error decoding physical activity: ", err)
			return nil, err
		}

		durationByteArray, err := json.Marshal(physicalActivity.DurationMap)
		physicalActivity.DurationJson = string(durationByteArray)
		if err != nil {
			fmt.Println("Error encoding duration json: ", err)
			return nil, err
		}

		distanceByteArray, err := json.Marshal(physicalActivity.DistanceMap)
		physicalActivity.DistanceJson = string(distanceByteArray)
		if err != nil {
			fmt.Println("Error encoding Distance json: ", err)
			return nil, err
		}

		energyByteArray, err := json.Marshal(physicalActivity.EnergyMap)
		physicalActivity.EnergyJson = string(energyByteArray)
		if err != nil {
			fmt.Println("Error encoding Energy json: ", err)
			return nil, err
		}

		return &physicalActivity, nil

	} else {
		fmt.Println("Can not create decoder: ", err)
		return nil, nil
	}
}
