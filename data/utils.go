package data

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type timerAddValue struct {
	start time.Time
	µs    int64
	num   int
}

var timerValues map[string]time.Time = make(map[string]time.Time)
var timerAddValues map[string]*timerAddValue = make(map[string]*timerAddValue)

// Utility functions:

// IsValidUUID check if the uuid is valid
func isValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}

// contains search an element in an array
//
// go seems to not have this helper in the base API
func contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

func timeIt(name string) {
	if _, present := timerValues[name]; present {
		fmt.Printf("Timer %s already started\n", name)
		return
	}
	timerValues[name] = time.Now()
}

func timeEnd(name string) int64 {
	start, present := timerValues[name]
	if !present {
		fmt.Printf("Timer %s has not started\n", name)
		return 0
	}
	end := time.Now()
	delete(timerValues, name)
	dur := end.Sub(start).Milliseconds()
	fmt.Printf("%s: %d ms\n", name, dur)
	return dur
}

func timeAddIt(name string, start bool) {
	tAdd, present := timerAddValues[name]

	if present {
		if start {
			tAdd.start = time.Now()
		} else {
			end := time.Now()
			tAdd.num++
			tAdd.µs += end.Sub(tAdd.start).Microseconds()
			tAdd.start = end
		}
	} else {
		timerAddValues[name] = &timerAddValue{
			start: time.Now(),
			µs:    0,
			num:   1,
		}
	}
}

func timeAddEnd(name string) (int64, int) {
	tAdd, present := timerAddValues[name]
	if !present {
		fmt.Printf("Timer %s has not started\n", name)
		return 0, 0
	}

	delete(timerAddValues, name)
	fmt.Printf("%s: %d µs, %d runs, ~%f µs/run\n", name, tAdd.µs, tAdd.num, float64(tAdd.µs)/float64(tAdd.num))
	return tAdd.µs, tAdd.num
}
