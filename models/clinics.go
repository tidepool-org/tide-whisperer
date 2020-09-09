package models

import (
	"errors"
	"fmt"
	"github.com/mitchellh/mapstructure"
)

type Clinics struct {
	IdMap  map[string]string            `mapstructure:"_id" pg:"-"`
	ClinicId    string        `pg:"clinic_id"`

	Name         string    `mapstructure:"name" pg:"name"`
	Address         string    `mapstructure:"address" pg:"address"`

	active   bool    `mapstructure:"active" pg:"active"`
}

func DecodeClinics(data interface{}) (*Clinics, error) {
	var clinics = Clinics{}

	if decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result: &clinics,
	   } ); err == nil {
		if err := decoder.Decode(data); err != nil {
			//fmt.Println("Error decoding clinis: ", err)
			return nil, err
		}

		if client_id, ok := clinics.IdMap["$oid"] ; ok {
			clinics.ClinicId = client_id
		}
		if clinics.ClinicId == ""  {
			//fmt.Println("clinicid is null ")
			return nil, errors.New("clinicid is null")

		}

		return &clinics, nil

	} else {
		fmt.Println("Can not create decoder: ", err)
		return nil, err
	}
}

func (u *Clinics) GetType() string {
	return "clinics"
}

