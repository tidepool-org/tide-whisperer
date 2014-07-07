package clients

import (
	"log"
	"net/url"
	"tidepool.org/tide-whisperer/clients/disc"
	"tidepool.org/tide-whisperer/clients/hakken"
)

type HostGetterConfig interface{}

func ToHostGetter(name string, c *HostGetterConfig, discovery disc.Discovery) disc.HostGetter {
	switch c := (*c).(type) {
	case string:
		return discovery.Watch(c).Random()
	case map[string]interface{}:
		theType := c["type"].(string)
		switch theType {
		case "static":
			hostStrings := c["hosts"].([]interface{})
			hosts := make([]url.URL, len(hostStrings))
			for i, v := range hostStrings {
				host, err := url.Parse(v.(string))
				if err != nil {
					panic(err.Error())
				}
				hosts[i] = *host
			}

			log.Printf("service[%s] with static watch for hosts[%v]", name, hostStrings)
			return &disc.StaticHostGetter{Hosts: hosts}
		case "random":
			return discovery.Watch(c["service"].(string)).Random()
		}
	default:
		log.Panicf("Unexpected type for HostGetterConfig[%T]", c)
	}

	panic("Appease the compiler, code should never get here")
}

type GatekeeperConfig struct {
	HostGetter HostGetterConfig `json:"serviceSpec"`
}

func (gc *GatekeeperConfig) ToHostGetter(discovery disc.Discovery) disc.HostGetter {
	return ToHostGetter("gatekeeper", &gc.HostGetter, discovery)
}

type SeagullConfig struct {
	HostGetter HostGetterConfig `json:"serviceSpec"`
}

func (sc *SeagullConfig) ToHostGetter(discovery disc.Discovery) disc.HostGetter {
	return ToHostGetter("seagull", &sc.HostGetter, discovery)
}

type UserApiConfig struct {
	UserApiClientConfig
	HostGetter HostGetterConfig `json:"serviceSpec"`
}

func (uac *UserApiConfig) ToHostGetter(discovery disc.Discovery) disc.HostGetter {
	return ToHostGetter("user-api", &uac.HostGetter, discovery)
}

type Config struct {
	HakkenConfig     hakken.HakkenClientConfig `json:"hakken"`
	GatekeeperConfig GatekeeperConfig          `json:"gatekeeper"`
	SeagullConfig    SeagullConfig             `json:"seagull"`
	UserApiConfig    UserApiConfig             `json:"userApi"`
}
