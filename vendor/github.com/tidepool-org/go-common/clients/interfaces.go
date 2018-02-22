// package clients is a set of structs and methods for client libraries that interact with the various
// services in the tidepool platform
package clients

type SecretProvider interface {
	SecretProvide() string
}

type SecretProviderFunc func() string

func (t SecretProviderFunc) SecretProvide() string {
	return t()
}
