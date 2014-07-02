package jepson

import "encoding/json"

func JSONString(data []byte) (string, error) {
	var retVal string
	err := json.Unmarshal(data, &retVal)
	return retVal, err
}
