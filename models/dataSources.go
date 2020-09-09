package models

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
	"time"
)

type DataSources struct {
	Id                string   `mapstructure:"id,omitempty" pg:"id"`
	UserId            string   `mapstructure:"_userId,omitempty" pg:"user_id"`
	ProviderType      string   `mapstructure:"providerType,omitempty" pg:"provider_type"`
	ProviderName      string   `mapstructure:"providerName,omitempty" pg:"provider_name"`
	State             string   `mapstructure:"state,omitempty" pg:"state"`

	CreatedTime       time.Time  `mapstructure:"createdTime" pg:"created_time,type:timestamptz"`
	ModifiedTime      time.Time  `mapstructure:"modifiedTime" pg:"modified_time,type:timestamptz"`

	DataSetIds        []int64    `mapstructure:"dataSetIds" pg:"data_set_ids,array"`

	EarliestDateTime  time.Time  `mapstructure:"earliestDateTime" pg:"earliest_date_time,type:timestamptz"`
	LatestDateTime    time.Time  `mapstructure:"latestDateTime" pg:"latest_date_time,type:timestamptz"`
	LastImportTime    time.Time  `mapstructure:"lastImportTime" pg:"last_import_time,type:timestamptz"`

	Revision          int64   `mapstructure:"revision,omitempty" pg:"revision"`
}



func DecodeDataSources(data interface{}) (*DataSources, error) {
	var dataSources = DataSources{}

	if decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: StringToTimeHookFuncTimezoneOptional(time.RFC3339),
		Result: &dataSources,
	} ); err == nil {
		if err := decoder.Decode(data); err != nil {
			//fmt.Println("Error decoding dataSources: ", err)
			return nil, err
		}

		return &dataSources, nil

	} else {
		fmt.Println("Can not create decoder: ", err)
		return nil, nil
	}
}
