// package clients is a set of structs and methods for client libraries that interact with the various
// services in the tidepool platform
package clients

import "net/url"

type HostGetter interface {
	HostGet() url.URL
}

type HostGetterFunc func() url.URL

func (h HostGetterFunc) HostGet() url.URL {
	return h()
}

type TokenProvider interface {
	TokenProvide() string
}

type TokenProviderFunc func() string

func (t TokenProviderFunc) TokenProvide() string {
	return t()
}
