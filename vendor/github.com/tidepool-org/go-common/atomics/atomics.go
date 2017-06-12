package atomics

import "sync"

type AtomicString struct {
	mut sync.RWMutex
	val string
}

// Set sets the value of the AtomicString.  It returns the old value
func (as *AtomicString) Set(val string) string {
	as.mut.Lock()
	defer as.mut.Unlock()
	retVal := as.val
	as.val = val
	return retVal
}

// Get gets the current value of the AtomicString
func (as *AtomicString) Get() string {
	as.mut.RLock()
	defer as.mut.RUnlock()
	return as.val
}

type AtomicInterface struct {
	mut sync.RWMutex
	val interface{}
}

// Set sets the value of the AtomicInterface.  It returns the old value
func (as *AtomicInterface) Set(val interface{}) interface{} {
	as.mut.Lock()
	defer as.mut.Unlock()
	retVal := as.val
	as.val = val
	return retVal
}

// Get gets the current value of the AtomicInterface
func (as *AtomicInterface) Get() interface{} {
	as.mut.RLock()
	defer as.mut.RUnlock()
	return as.val
}
