package models

import (
	"encoding/json"
	"github.com/mitchellh/mapstructure"
	"strings"
	"time"
	"fmt"
)

type PhysicalActivity struct {
	Base                                           `mapstructure:",squash"`

	Duration    map[string]interface{}         `mapstructure:"duration" pg:"duration" json:"duration"`

	Distance    map[string]interface{}         `mapstructure:"distance" pg:"distance" json:"distance"`

	Energy      map[string]interface{}         `mapstructure:"energy" pg:"energy" json:"energy"`

	Name           string                         `mapstructure:"name" pg:"name" json:"name"`
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

		return &physicalActivity, nil

	} else {
		fmt.Println("Can not create decoder: ", err)
		return nil, nil
	}
}


type PhysicalActivityAlias PhysicalActivity

func (p PhysicalActivity) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewJSONPhysicalActivity(p))
}


func NewJSONPhysicalActivity(physicalActivity PhysicalActivity) JSONPhysicalActivity {
	return JSONPhysicalActivity{
		PhysicalActivityAlias(physicalActivity),
		strings.Trim(physicalActivity.DeviceTime.Format(time.RFC3339), "Z"),
	}
}

type JSONPhysicalActivity struct {
	PhysicalActivityAlias
	DeviceTime string `json:"deviceTime"`
}
