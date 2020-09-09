package models

import (
	"errors"
	"fmt"
	"github.com/mitchellh/mapstructure"
)

type OldClinicsPatients struct {
	OldClinicId          string    `mapstructure:"userId" pg:"old_clinic_id"`
	PatientId          string      `mapstructure:"groupId" pg:"patient_id"`

}

func DecodeOldClinicsPatients(data interface{}) (*OldClinicsPatients, error) {
	var patients = OldClinicsPatients{}

	if decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result: &patients,
	   } ); err == nil {
		if err := decoder.Decode(data); err != nil {
			//fmt.Println("Error decoding clinis: ", err)
			return nil, err
		}

		if patients.OldClinicId == "" || patients.PatientId == "" {
			//fmt.Println("clinicID or patientID is null ")
			return nil, errors.New("clinicid or patientid is null")

		}

		return &patients, nil

	} else {
		fmt.Println("Can not create decoder: ", err)
		return nil, err
	}
}

func (u *OldClinicsPatients) GetType() string {
	return "oldClinicsPatients"
}

