package mongo

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/tidepool-org/go-common/errors"
	"github.com/tidepool-org/go-common/jepson"
	mgo "gopkg.in/mgo.v2"
)

type Config struct {
	ConnectionString string           `json:"connectionString"`
	Timeout          *jepson.Duration `json:"timeout"`
}

func Connect(config *Config) (*mgo.Session, error) {
	if config.ConnectionString == "" {
		return nil, errors.New("Must specify a ConnectionString on mongo config")
	}

	dur := 20 * time.Second
	if config.Timeout != nil {
		dur = time.Duration(*config.Timeout)
	}
	log.Printf("Initializing with config[%+v], dur[%v]", config, dur)
	return DialWithTimeout(config.ConnectionString, dur)
}

/*
 All following code originally C&Pd from mgo.  It has been adjusted to allow
 for automatic handling of ssl connections based on a connection string parameter
*/

// mgo - MongoDB driver for Go
//
// Copyright (c) 2010-2012 - Gustavo Niemeyer <gustavo@niemeyer.net>
//
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation
//    and/or other materials provided with the distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

// DialWithTimeout works like Dial, but uses timeout as the amount of time to
// wait for a server to respond when first connecting and also on follow up
// operations in the session. If timeout is zero, the call may block
// forever waiting for a connection to be made.
//
// See SetSyncTimeout for customizing the timeout for the session.
func DialWithTimeout(url string, timeout time.Duration) (*mgo.Session, error) {
	uinfo, err := parseURL(url)
	if err != nil {
		return nil, err
	}
	direct := false
	mechanism := ""
	service := ""
	source := ""
	var dialServer func(*mgo.ServerAddr) (net.Conn, error)
	for k, v := range uinfo.options {
		switch k {
		case "authSource":
			source = v
		case "authMechanism":
			mechanism = v
		case "gssapiServiceName":
			service = v
		case "ssl":
			if ssl, sslErr := strconv.ParseBool(v); sslErr == nil && ssl {
				dialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {
					return tls.Dial("tcp", addr.String(), &tls.Config{InsecureSkipVerify: true})
				}
			}
		case "connect":
			if v == "direct" {
				direct = true
				break
			}
			if v == "replicaSet" {
				break
			}
			fallthrough
		default:
			return nil, errors.New("unsupported connection URL option: " + k + "=" + v)
		}
	}
	info := mgo.DialInfo{
		Addrs:      uinfo.addrs,
		Direct:     direct,
		Timeout:    timeout,
		Database:   uinfo.db,
		Username:   uinfo.user,
		Password:   uinfo.pass,
		Mechanism:  mechanism,
		Service:    service,
		Source:     source,
		DialServer: dialServer,
	}
	return mgo.DialWithInfo(&info)
}

func isOptSep(c rune) bool {
	return c == ';' || c == '&'
}

type urlInfo struct {
	addrs   []string
	user    string
	pass    string
	db      string
	options map[string]string
}

func parseURL(s string) (*urlInfo, error) {
	if strings.HasPrefix(s, "mongodb://") {
		s = s[10:]
	}
	info := &urlInfo{options: make(map[string]string)}
	if c := strings.Index(s, "?"); c != -1 {
		for _, pair := range strings.FieldsFunc(s[c+1:], isOptSep) {
			l := strings.SplitN(pair, "=", 2)
			if len(l) != 2 || l[0] == "" || l[1] == "" {
				return nil, errors.New("connection option must be key=value: " + pair)
			}
			info.options[l[0]] = l[1]
		}
		s = s[:c]
	}
	if c := strings.Index(s, "@"); c != -1 {
		pair := strings.SplitN(s[:c], ":", 2)
		if len(pair) > 2 || pair[0] == "" {
			return nil, errors.New("credentials must be provided as user:pass@host")
		}
		var err error
		info.user, err = url.QueryUnescape(pair[0])
		if err != nil {
			return nil, fmt.Errorf("cannot unescape username in URL: %q", pair[0])
		}
		if len(pair) > 1 {
			info.pass, err = url.QueryUnescape(pair[1])
			if err != nil {
				return nil, fmt.Errorf("cannot unescape password in URL")
			}
		}
		s = s[c+1:]
	}
	if c := strings.Index(s, "/"); c != -1 {
		info.db = s[c+1:]
		s = s[:c]
	}
	info.addrs = strings.Split(s, ",")
	return info, nil
}
