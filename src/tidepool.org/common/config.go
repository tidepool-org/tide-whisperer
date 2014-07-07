package common

import (
	"encoding/json"
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

type ServiceConfig struct {
	Addr    string `json:"address"`
	Service string `json:"service"`
}
