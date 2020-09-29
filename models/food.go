package models

import (
	"encoding/json"
	"github.com/mitchellh/mapstructure"
	"strings"
	"time"
	"fmt"
)

type Food struct {
	Base                                           `mapstructure:",squash"`

        Nutrition    map[string]interface{}         `mapstructure:"nutrition" pg:"nutrition" json:"nutrition"`
}

func DecodeFood(data interface{}) (*Food, error) {
	var food = Food{}

	if decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: StringToTimeHookFuncTimezoneOptional(time.RFC3339),
		Result: &food,
	   } ); err == nil {
		if err := decoder.Decode(data); err != nil {
			//fmt.Println("Error decoding food: ", err)
			return nil, err
		}

		return &food, nil

	} else {
		fmt.Println("Can not create decoder: ", err)
		return nil, nil
	}
}

type FoodAlias Food

func (f Food) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewJSONFood(f))
}


func NewJSONFood(food Food) JSONFood {
	return JSONFood{
		FoodAlias(food),
		strings.Trim(food.DeviceTime.Format(time.RFC3339), "Z"),
	}
}

type JSONFood struct {
	FoodAlias
	DeviceTime string `json:"deviceTime"`
}
