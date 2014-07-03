package disc

import (
	"encoding/json"
	"net/url"
	"tidepool.org/common/jepson"
)

type ServiceListing struct {
	url.URL
	Service    string
	properties map[string]string
}

func (sl *ServiceListing) UnmarshalJSON(data []byte) error {
	asMap := make(map[string]json.RawMessage)
	err := json.Unmarshal(data, &asMap)
	if err != nil {
		return err
	}

	properties := make(map[string]string)
	for k, v := range asMap {
		switch k {
		case "host":
			sl.Host, _ = jepson.JSONString(([]byte)(v))
		case "protocol":
			sl.Scheme, _ = jepson.JSONString(([]byte)(v))
		case "service":
			sl.Service, _ = jepson.JSONString(([]byte)(v))
		default:
			properties[k], _ = jepson.JSONString(([]byte)(v))
		}
	}
	sl.properties = properties

	return nil
}

func (sl *ServiceListing) MarshalJSON() ([]byte, error) {
	objs := make(map[string]string)

	objs["service"] = sl.Service
	if sl.Host != "" {
		objs["host"] = sl.Host
	}
	if sl.Scheme != "" {
		objs["protocol"] = sl.Scheme
	}

	for k, v := range sl.properties {
		objs[k] = v
	}

	return json.Marshal(objs)
}

func (sl *ServiceListing) getProperty(property string) string {
	return sl.properties[property]
}

func (lhs *ServiceListing) Equals(rhs ServiceListing) bool {
	retVal := lhs.Service == rhs.Service && lhs.URL == rhs.URL && len(lhs.properties) == len(rhs.properties)

	if retVal {
		for k, v := range lhs.properties {
			if rhs.properties[k] != v {
				return false
			}
		}
	}

	return retVal
}
