package models

import (
	"encoding/json"
	"github.com/mitchellh/mapstructure"
	"strings"
	"time"
	"fmt"
)

type Wizard struct {
	Base                                             `mapstructure:",squash"`

	Bolus             string                         `mapstructure:"bolus" pg:"bolus" json:"bolus,omitempty"`
	Units             string                         `mapstructure:"units" pg:"units" json:"units,omitempty"`

	Recommended    map[string]interface{}         `mapstructure:"recommended" pg:"recommended" json:"recommended"`

	BgInput           float64                         `mapstructure:"bgInput" pg:"bg_input,notnull" json:"bgInput,omitempty"`
	BgTarget          map[string]interface{}         `mapstructure:"bgTarget" pg:"bg_target" json:"bgTarget"`

	CarbInput         int64                          `mapstructure:"carbInput" pg:"carb_input,notnull" json:"carbInput,omitempty"`
	InsulinCarbRatio  int64                          `mapstructure:"insulinCarbRatio" pg:"insulin_carb_ratio,notnull" json:"insulinCarbRatio,omitempty"`

	InsulinOnBoard    float64                        `mapstructure:"insulinOnBoard" pg:"insulin_on_board,notnull" json:"insulinOnBoard,omitempty"`
	InsulinSensitivity    float64                    `mapstructure:"insulinSensitivity" pg:"insulin_sensitivity,notnull" json:"insulinSensitivity,omitempty"`


}

func DecodeWizard(data interface{}) (*Wizard, error) {
	var wizard = Wizard{}

	if decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: StringToTimeHookFuncTimezoneOptional(time.RFC3339),
		Result: &wizard,
	   } ); err == nil {
		if err := decoder.Decode(data); err != nil {
			//fmt.Println("Error decoding wizard: ", err)
			return nil, err
		}

		return &wizard, nil

	} else {
		fmt.Println("Can not create decoder: ", err)
		return nil, nil
	}
}

type WizardAlias Wizard

func (w Wizard) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewJSONWizard(w))
}


func NewJSONWizard(wizard Wizard) JSONWizard {
	return JSONWizard{
		WizardAlias(wizard),
		strings.Trim(wizard.DeviceTime.Format(time.RFC3339), "Z"),
	}
}

type JSONWizard struct {
	WizardAlias
	DeviceTime string `json:"deviceTime"`
}
