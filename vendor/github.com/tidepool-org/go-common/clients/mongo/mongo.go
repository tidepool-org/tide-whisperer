package mongo

import (
	"crypto/tls"
	"os"
	"net"
	"time"

	"github.com/globalsign/mgo"
	"github.com/tidepool-org/go-common/errors"
	"github.com/tidepool-org/go-common/jepson"
)

type Config struct {
	ConnectionString string           `json:"connectionString"`
	Timeout          *jepson.Duration `json:"timeout"`
	User             string           `json:"user"`
	Password         string           `json:"password"`
	Database         string           `json:"database"`
	Ssl              bool             `json:"ssl"`
	Hosts            string           `json:"hosts"`
	OptParams        string           `json:"optParams"`
}

func(config *Config) FromEnv() {
	config.Hosts, _ = os.LookupEnv("MONGO_HOSTS")
	config.User, _ = os.LookupEnv("MONGO_USER")
	config.Password, _ = os.LookupEnv("MONGO_PASSWORD")
	config.Database, _ = os.LookupEnv("MONGO_DATABASE")
	config.OptParams, _ = os.LookupEnv("MONGO_OPT_PARAMS")
	ssl, found := os.LookupEnv("MONGO_SSL")
	config.Ssl = found && ssl == "true"
}

func (config *Config) ToConnectionString() (string, error) {
	if config.ConnectionString != "" {
		return config.ConnectionString, nil
	}
	if config.Database == "" {
		return "", errors.New("Must specify a database in Mongo config")
	}
	cs := "mongodb://"
	if config.User != "" {
		cs += config.User
		if config.Password != "" {
			cs += ":"
			cs += config.Password
		}
		cs += "@"
	}

	if config.Hosts != "" {
		cs += config.Hosts
		cs += "/"
	} else {
		cs += "localhost/"
	}

	if config.Database != "" {
		cs += config.Database
	}

	if config.Ssl {
		cs += "?ssl=true"
	} else {
		cs += "?ssl=false"
	}

	if config.OptParams != "" {
		cs += "&"
		cs += config.OptParams
	}
	return cs, nil
}

func Connect(config *Config) (*mgo.Session, error) {
	connectionString, err := config.ToConnectionString()
	if err != nil {
		return nil, err
	}
	dur := 20 * time.Second
	if config.Timeout != nil {
		dur = time.Duration(*config.Timeout)
	}

	dialInfo, err := mgo.ParseURL(connectionString)
	if err != nil {
		return nil, err
	}
	dialInfo.Timeout = dur

	if dialInfo.DialServer != nil {
		// TODO Ignore server cert for now.  We should install proper CA to verify cert.
		dialInfo.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {
			return tls.Dial("tcp", addr.String(), &tls.Config{InsecureSkipVerify: true})
		}
	}
	return mgo.DialWithInfo(dialInfo)
}

