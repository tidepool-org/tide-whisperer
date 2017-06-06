package common

import "net/url"

// URLOrPanic parses a string as a url, if there is an error, it panics
func URLOrPanic(val string) *url.URL {
	retVal, err := url.Parse(val)
	if err != nil {
		panic(err.Error())
	}
	return retVal
}
