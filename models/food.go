package models

import (
	"encoding/json"
	"github.com/mitchellh/mapstructure"
	"time"
	"fmt"
)

type Food struct {
	Base                                           `mapstructure:",squash"`

	NutritionMap    map[string]interface{}         `mapstructure:"nutrition" pg:"-"`
	NutritionJson   string                         `pg:"nutrition"`

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

		nutritionByteArray, err := json.Marshal(food.NutritionMap)
		food.NutritionJson = string(nutritionByteArray)
		if err != nil {
			//fmt.Println("Error encoding nutrition json: ", err)
			return nil, err
		}

		return &food, nil

	} else {
		fmt.Println("Can not create decoder: ", err)
		return nil, nil
	}
}
