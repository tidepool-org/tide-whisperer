package models

import (
	"github.com/mitchellh/mapstructure"
	"time"
	"fmt"
)

type Basal struct {
	Base                      `mapstructure:",squash"`

	DeliveryType      string   `mapstructure:"deliveryType,omitempty" pg:"delivery_type"`
	Duration          int64    `mapstructure:"duration,omitempty" pg:"duration"`
	ExpectedDuration  int64    `mapstructure:"expectedDuration,omitempty" pg:"expected_duration"`
	Rate              float64  `mapstructure:"rate,omitempty" pg:"rate"`
	Percent           float64  `mapstructure:"percent,omitempty" pg:"percent"`
	ScheduleName      string   `mapstructure:"scheduleName,omitempty" pg:"schedule_name"`
	Active            bool    `mapstructure:"_active" pg:"-"`
}

func DecodeBasal(data interface{}) (*Basal, error)  {
	var basal = Basal{}

	if decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: StringToTimeHookFuncTimezoneOptional(time.RFC3339),
		Result: &basal,
	   } ); err == nil {
		if err := decoder.Decode(data); err != nil {
			//fmt.Println("Error decoding basal: ", err)
			return nil, err
		}

		return &basal, nil

	} else {
		fmt.Println("Can not create decoder: ", err)
		return nil, err
	}
}

