package hakken

import (
	"encoding/json"
	"log"
	"net/url"
	"sync"
	"time"
)

type Coordinator struct {
	url.URL
}

func (c *Coordinator) UnmarshalJSON(data []byte) error {
	asMap := make(map[string]json.RawMessage)
	err := json.Unmarshal(data, &asMap)
	if err != nil {
		return err
	}

	c.Scheme = "http"
	json.Unmarshal(([]byte)(asMap["host"]), &c.Host)
	return nil
}

func (c *Coordinator) MarshalJSON() ([]byte, error) {
	objs := make(map[string]string)

	objs["host"] = c.Host
	objs["scheme"] = c.Scheme

	return json.Marshal(objs)
}

type coordinatorManager struct {
	resyncClient   coordinatorClient
	resyncInterval time.Duration
	pollInterval   time.Duration
	dropCooChan    chan *coordinatorClient

	mut sync.Mutex

	clients []coordinatorClient
	stop    chan chan error
}

func (manager *coordinatorManager) getClient() *coordinatorClient {
	manager.mut.Lock()
	defer manager.mut.Unlock()
	if manager.clients == nil || len(manager.clients) == 0 {
		return nil
	} else {
		return &manager.clients[0]
	}
}

func (manager *coordinatorManager) getClients() *[]coordinatorClient {
	manager.mut.Lock()
	defer manager.mut.Unlock()
	return &manager.clients
}

func (manager *coordinatorManager) reportBadClient(client *coordinatorClient) {
	manager.dropCooChan <- client
}

func (manager *coordinatorManager) start() error {
	manager.mut.Lock()
	defer manager.mut.Unlock()

	if manager.stop != nil {
		return nil
	}

	log.Printf("Starting coordinatorManager at[%s]", manager.resyncClient.coordinator.URL.String())
	manager.stop = make(chan chan error)
	coordinators, err := addUnknownCoordinators(nil, &manager.resyncClient)
	if err != nil {
		return err
	}

	manager.clients = coordinators

	go func() {
		resyncTimer := time.After(manager.resyncInterval)
		pollTimer := time.After(manager.pollInterval)

		for {
			manager.mut.Lock()
			coordinators = manager.clients
			manager.mut.Unlock()
			select {
			case <-resyncTimer:
				coordinators, _ = addUnknownCoordinators(coordinators, &manager.resyncClient)
				manager.setCoordinators(&coordinators)
				resyncTimer = time.After(manager.resyncInterval)
			case <-pollTimer:
				for _, coo := range coordinators {
					coordinators, err = addUnknownCoordinators(coordinators, &coo)
					if err != nil {
						manager.setCoordinators(removeCoordinator(coordinators, &coo))
					}
				}
				pollTimer = time.After(manager.pollInterval)
			case errChan := <-manager.stop:
				// Empty out the dropCooChan
				for {
					select {
					case <-manager.dropCooChan:
						// Do nothing
					default:
						// Be done
						errChan <- nil
						return
					}
				}
			case droppedCoo := <-manager.dropCooChan:
				manager.setCoordinators(removeCoordinator(coordinators, droppedCoo))
			}
		}
	}()

	return nil
}

func (manager *coordinatorManager) Close() error {
	manager.mut.Lock()
	defer manager.mut.Unlock()

	if manager.stop == nil {
		return nil
	}

	errChan := make(chan error)
	manager.stop <- errChan

	err := <-errChan

	manager.stop = nil
	manager.clients = nil
	return err
}

func (manager *coordinatorManager) setCoordinators(coordinators *[]coordinatorClient) {
	manager.mut.Lock()
	defer manager.mut.Unlock()
	manager.clients = *coordinators
}

func addUnknownCoordinators(coordinators []coordinatorClient, client *coordinatorClient) ([]coordinatorClient, error) {
	coos, err := client.getCoordinators()
	if err != nil {
		return coordinators, err
	}

	unknown := make([]coordinatorClient, 0, len(coos))
	for _, coo := range coos {
		found := false
		for _, known := range coordinators {
			if coo == known.coordinator {
				found = true
			}
		}
		if !found {
			unknown = append(unknown, coordinatorClient{coo})
		}
	}

	for _, coo := range unknown {
		log.Printf("Adding coordinator[%+v]", coo.coordinator)
	}

	return append(coordinators, unknown...), nil
}

func removeCoordinator(coordinators []coordinatorClient, toRemove *coordinatorClient) *[]coordinatorClient {
	for i, coo := range coordinators {
		if &coo == toRemove {
			log.Printf("Removing coordinator[%+v]", coo)
			retVal := append(coordinators[0:i], coordinators[i+1:]...)
			return &retVal
		}
	}
	return &coordinators
}

func getOrNil(arr []coordinatorClient, i int) *coordinatorClient {
	if len(arr) > i {
		return &arr[i]
	} else {
		return nil
	}
}
