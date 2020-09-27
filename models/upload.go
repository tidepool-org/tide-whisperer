package models

import (
	"github.com/mitchellh/mapstructure"
	"strings"
	"time"
	"fmt"
	"encoding/json"
)

type Upload struct {
	Base                    `mapstructure:",squash"`

	DataSetType    string    `mapstructure:"dataSetType" pg:"data_set_type" json:"dataSetType,omitempty"`
	DataState      string    `mapstructure:"_dataState" pg:"data_state" json:"_dataState,omitempty"`

	DeviceSerialNumber string `mapstructure:"deviceSerialNumber" pg:"device_serial_number" json:"deviceSerialNumber,omitempty"`
	State          string    `mapstructure:"_state" pg:"state" json:"_state,omitempty"`
	Version        string    `mapstructure:"version" pg:"version" json:"version,omitempty"`
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

type UploadAlias Upload

func (u Upload) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewJSONUpload(u))
}


func NewJSONUpload(smbg Upload) JSONUpload {
	return JSONUpload{
		UploadAlias(smbg),
		strings.Trim(smbg.DeviceTime.Format(time.RFC3339), "Z"),
	}
}

type JSONUpload struct {
	UploadAlias
	DeviceTime string `json:"deviceTime"`
}