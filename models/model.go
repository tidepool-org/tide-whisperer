package models

import (
	"github.com/mitchellh/mapstructure"
	"fmt"
	"strings"
	"time"
	"reflect"
)

var Active = 0
var Inactive = 0
const DeviceDataCollection = "deviceData"
const UsersCollection = "users"
const ClinicsCollection = "clinic"
const ClinicsCliniciansCollection = "clinicsClinicians"
const ClinicsPatientsCollection = "clinicsPatients"
const PermsCollection = "perms"

type BaseDeviceModel struct {
	Type      string `mapstructure:"type"`
	Active    bool `mapstructure:"_active"`
}

func DecodeModel(data interface{}, topic string) (Model, error) {
	switch {
	case strings.HasSuffix(topic, DeviceDataCollection):
		return DecodeDeviceModel(data)
	default:
		return DecodeGeneralModel(data, topic)
	}
}

func DecodeGeneralModel(data interface{}, topic string) (Model, error) {
	switch {
	case strings.HasSuffix(topic, UsersCollection):
		user, err := DecodeUser(data)
		return user, err
	case strings.HasSuffix(topic, ClinicsCollection):
		user, err := DecodeClinics(data)
		return user, err
	case strings.HasSuffix(topic, ClinicsCliniciansCollection):
		user, err := DecodeClinicsClinicians(data)
		return user, err
	case strings.HasSuffix(topic, ClinicsPatientsCollection):
		user, err := DecodeClinicsPatients(data)
		return user, err
	case strings.HasSuffix(topic, PermsCollection):
		user, err := DecodeOldClinicsPatients(data)
		return user, err
	}
	fmt.Println("Could not decode.  Do not have a database for topic: ", topic)
	return nil, nil
}

func DecodeDeviceModel(data interface{}) (Model, error) {
	var baseDeviceModel BaseDeviceModel
	if err := mapstructure.Decode(data, &baseDeviceModel); err != nil {
		fmt.Println("Problem decoding base model", err)
		return nil, err
	}
	if baseDeviceModel.Active {
		Active += 1
		switch baseDeviceModel.Type {
		case "upload":
			upload, err := DecodeUpload(data)
			return upload, err
		case "basal":
			basal, err := DecodeBasal(data)
			return basal, err
		case "bolus":
			bolus, err := DecodeBolus(data)
			return bolus, err
		case "cbg":
			cbg, err := DecodeCbg(data)
			return cbg, err
		case "smbg":
			smbg, err := DecodeSmbg(data)
			return smbg, err
		case "wizard":
			wizard, err := DecodeWizard(data)
			return wizard, err
		case "food":
			food, err := DecodeFood(data)
			return food, err
		case "deviceEvent":
			deviceEvent, err := DecodeDeviceEvent(data)
			return deviceEvent, err
		case "pumpSettings":
			pumpSettings, err := DecodePumpSettings(data)
			return pumpSettings, err
		case "physicalActivity":
			physicalActivity, err := DecodePhysicalActivity(data)
			return physicalActivity, err
		case "cgmSettings":
			cgmSettings, err := DecodeCgmSettings(data)
			return cgmSettings, err
		case "deviceMeta":
			deviceMeta, err := DecodeDeviceMeta(data)
			return deviceMeta, err
		default:
			fmt.Println("Currently not handling type: ", baseDeviceModel.Type)
		}
	} else {
		Inactive += 1
	}
	return nil, nil
}

// StringToTimeHookFuncTimezoneOptional returns a DecodeHookFunc that converts
// strings to time.Time.  If time does not have a timezone - appends a Z for UTC timezone
func StringToTimeHookFuncTimezoneOptional(layout string) mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf(time.Time{}) {
			return data, nil
		}

		// Convert it by parsing
		s := data.(string)
		if !strings.Contains(s, "Z") && !strings.Contains(s, "+") {
			s += "Z"
		}
		return time.Parse(layout, s)
	}
}