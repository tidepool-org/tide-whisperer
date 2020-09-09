package models

import (
	"github.com/mitchellh/mapstructure"
	"time"
	"fmt"
)

type Upload struct {
	Base                    `mapstructure:",squash"`

	DataSetType    string    `mapstructure:"dataSetType" pg:"data_set_type"`
	DataState      string    `mapstructure:"_dataState" pg:"data_state"`

	DeviceSerialNumber string `mapstructure:"deviceSerialNumber" pg:"device_serial_number"`
	State          string    `mapstructure:"_state" pg:"state"`
	Version        string    `mapstructure:"version" pg:"version"`
}

func DecodeUpload(data interface{}) (*Upload, error) {
	var upload = Upload{}

	if decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: StringToTimeHookFuncTimezoneOptional(time.RFC3339),
		Result: &upload,
	   } ); err == nil {
		if err := decoder.Decode(data); err != nil {
			//fmt.Println("Error decoding upload: ", err)
			return nil, err
		}

		return &upload, nil

	} else {
		fmt.Println("Can not create decoder: ", err)
		return nil, nil
	}
}
