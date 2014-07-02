package hakken

import (
	"sync"
	"tidepool.org/tide-whisperer/clients"
	"time"
	"math/rand"
	"net/url"
	"log"
)

var random *rand.Rand
func init() {
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
}

type watch struct {
	incoming chan *payload
	listings []ServiceListing

	mut sync.RWMutex
}

type payload struct {
	listings []ServiceListing
	done chan bool
}

func newWatch(theChan chan *payload) *watch {
	retVal := &watch{incoming: theChan}
	retVal.start()
	return retVal
}

func (g *watch) ServiceListingsGet() []ServiceListing {
	g.mut.RLock()
	defer g.mut.RUnlock()
	return g.listings
}

func (g *watch) start() {
	go func(){
		more := true
		for ;more; {
			var payload *payload
			payload, more = <-g.incoming
			theList := payload.listings
			addedItems := make([]ServiceListing, len(theList))
			copy(addedItems, theList)
			var removedItems []ServiceListing

			g.mut.Lock()
			for _, listing := range g.listings {
				found := false
				for i, newListing := range addedItems {
					if newListing.Equals(listing) {
						addedItems = append(addedItems[0:i], addedItems[i+1:]...)
						found = true
						break
					}
				}
				if !found {
					removedItems = append(removedItems, listing)
				}
			}
			g.listings = theList
			g.mut.Unlock()

			for _, listing := range removedItems {
				log.Printf("Removing listing[%+v]", listing)
			}
			for _, listing := range addedItems {
				log.Printf("Adding listing[%+v]", listing)
			}
			close(payload.done)
		}
	}()
}

func (g *watch) Random() clients.HostGetter {
	return clients.HostGetterFunc(func() []url.URL {
		listings := g.ServiceListingsGet()
		if (len(listings) == 0) {
			return nil
		}

		return []url.URL{listings[random.Intn(len(listings))].URL}
	})
}
