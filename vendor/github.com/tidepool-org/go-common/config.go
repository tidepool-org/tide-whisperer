package common

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

func LoadConfig(filenames []string, obj interface{}) error {
	for _, filename := range filenames {
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			continue
		}

		bytes, err := ioutil.ReadFile(filename)
		if err != nil {
			return err
		}

		if err := json.Unmarshal(bytes, obj); err != nil {
			return err
		}
	}
	return nil
}

func LoadEnvironmentConfig(envVars []string, obj interface{}) error {
	for _, envVar := range envVars {
		envValue := os.Getenv(envVar)
		if envValue == "" {
			return fmt.Errorf("%s not found", envVar)
		}

		if err := json.Unmarshal([]byte(envValue), obj); err != nil {
			return fmt.Errorf("%s errored: %s", envVar, err.Error())
		}
	}
	return nil
}

type ServiceConfig struct {
	Addr    string `json:"address"`
	Service string `json:"service"`
}
