package models

import (
	"errors"
	"fmt"
	"github.com/mitchellh/mapstructure"
)

type ClinicsPatients struct {
	ClinicId          string    `mapstructure:"clinicId" pg:"clinic_id"`
	PatientId          string    `mapstructure:"patientId" pg:"patient_id"`

	active   bool    `mapstructure:"active" pg:"active"`
}

func DecodeClinicsPatients(data interface{}) (*ClinicsPatients, error) {
	var patients = ClinicsPatients{}

	if decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result: &patients,
	   } ); err == nil {
		if err := decoder.Decode(data); err != nil {
			//fmt.Println("Error decoding clinis: ", err)
			return nil, err
		}

		if patients.ClinicId == "" || patients.PatientId == "" {
			//fmt.Println("clinicID or patientID is null ")
			return nil, errors.New("clinicid or patientid is null")

		}

		return &patients, nil

	} else {
		fmt.Println("Can not create decoder: ", err)
		return nil, err
	}
}

func (u *ClinicsPatients) GetType() string {
	return "clinicsPatients"
}

