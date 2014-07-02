package hakken

import (
	"net/url"
	"sync"
	"time"
	"log"
)

type hakkenClient struct {
	config   hakkenClientConfig
	cooMan   coordinatorManager
	stopChan chan bool

	mut sync.Mutex
}

type hakkenClientConfig struct {
	Host              string `json:host`              // Primary host to bootstrap list of coordinators from
	HeartbeatInterval int64  `json:heartbeatInterval` // Time elapsed between heartbeats and watch polls
	PollInterval      int64  `json:pollInterval`      // Time elapsed between coordinator gossip polls
	ResyncInterval    int64  `json:resyncInterval`    // Time elapsed between checks for new coordinators at Host
}

type HakkenClientBuilder struct {
	config hakkenClientConfig
}

func NewHakkenBuilder() *HakkenClientBuilder {
	return &HakkenClientBuilder{}
}

func (b *HakkenClientBuilder) WithHost(host string) *HakkenClientBuilder {
	b.config.Host = host
	return b
}

func (b *HakkenClientBuilder) WithHeartbeatInterval(intvl int64) *HakkenClientBuilder {
	b.config.HeartbeatInterval = intvl
	return b
}

func (b *HakkenClientBuilder) WithPollInterval(intvl int64) *HakkenClientBuilder {
	b.config.PollInterval = intvl
	return b
}

func (b *HakkenClientBuilder) WithResyncInterval(intvl int64) *HakkenClientBuilder {
	b.config.ResyncInterval = intvl
	return b
}

func (b *HakkenClientBuilder) Build() *hakkenClient {
	if b.config.Host == "" {
		panic("HakkenClientBuilder requires a Host")
	}
	if b.config.HeartbeatInterval == 0 {
		b.config.HeartbeatInterval = 20000
	}
	if b.config.PollInterval == 0 {
		b.config.PollInterval = 60000
	}
	if b.config.ResyncInterval == 0 {
		b.config.ResyncInterval = b.config.PollInterval * 2
	}
	return &hakkenClient{
		config: b.config,
		cooMan: coordinatorManager{
			resyncClient:   coordinatorClient{Coordinator{url.URL{Scheme: "http", Host: b.config.Host}}},
			resyncInterval: time.Duration(b.config.ResyncInterval) * time.Millisecond,
			pollInterval:   time.Duration(b.config.PollInterval) * time.Millisecond,
			dropCooChan:    make(chan *coordinatorClient),
		},
		stopChan:     make(chan bool),
	}
}

func (client *hakkenClient) Start() error {
	log.Println("Starting hakken")
	err := client.cooMan.start()
	if err != nil {
		return err
	}

	return nil
}

func (client *hakkenClient) Close() error {
	err := client.cooMan.Close()
	if err != nil {
		return err
	}

	close(client.stopChan)
	return nil
}

func (client *hakkenClient) Watch(service string) *watch {
	log.Printf("Creating watch for service[%s] with interval[%d]", service, client.config.HeartbeatInterval)
	slChan := make(chan *payload)
	retVal := newWatch(slChan)

	cooClient := client.cooMan.getClient()
	if (cooClient != nil) {
		listings, err := cooClient.getListings(service)
		if (err == nil) {
			done := make(chan bool)
			slChan <- &payload{listings: listings, done: done}
			<-done
		} else {
			log.Printf("Error when getting initial listings[%v]", err)
		}
	} else {
		log.Printf("No known coordinators, cannot load initial watch list for service[%s]", service)
	}

	go func() {
		timer := time.After(time.Duration(client.config.HeartbeatInterval) * time.Millisecond)
		for {
			select {
			case <- client.stopChan:
				close(slChan)
			case <- timer:
				cooClient := client.cooMan.getClient()
				if (cooClient != nil) {
					listings, err := cooClient.getListings(service)
					if err == nil {
						done := make(chan bool)
						slChan <- &payload{listings: listings, done: done}
						<-done
					}
				}

				timer = time.After(time.Duration(client.config.HeartbeatInterval) * time.Millisecond)
			}
		}
	}()

	return retVal
}
