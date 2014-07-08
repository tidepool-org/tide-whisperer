package disc

import "net/url"

type HostGetter interface {
	HostGet() []url.URL
}

type HostGetterFunc func() []url.URL

func (h HostGetterFunc) HostGet() []url.URL {
	return h()
}

type StaticHostGetter struct {
	Hosts []url.URL
}

func NewStaticHostGetter(retVal url.URL) *StaticHostGetter {
	return &StaticHostGetter{Hosts: []url.URL{retVal}}
}

func (h *StaticHostGetter) HostGet() []url.URL {
	return h.Hosts
}
