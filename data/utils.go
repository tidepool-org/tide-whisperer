package data

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
)

type timeItKey int
type timerAddValue struct {
	start        time.Time
	microSeconds int64
	num          int
}
type timeItType struct {
	timers    map[string]time.Time
	timersAdd map[string]*timerAddValue
	results   string
}

const (
	// To convert mg/dL to mmol/L and vice-versa
	mgdlPerMmoll float64 = 18.01577
	unitMgdL             = "mg/dL"
	unitMmolL            = "mmol/L"
)

// Utility functions:

// convertBG is a common util function to convert bg values
// to/from "mg/dL" and "mmol/L"
//
// - param: value The value to convert
//
// - param: unit The unit of the passed value
//
// - return: The converted value in the opposite unit
func convertBG(value float64, unit string) (float64, error) {
	if value < 0 {
		return 0, errors.New("Invalid glycemia value")
	}
	if unit == unitMgdL {
		return math.Round(10.0*value/mgdlPerMmoll) / 10, nil
	}
	if unit == unitMmolL {
		return math.Round(value * mgdlPerMmoll), nil
	}
	return 0, errors.New("Invalid parameter unit")
}

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
func containsInt(a []int, x int) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

func timeItContext(ctx context.Context) context.Context {
	value := &timeItType{
		timers:    make(map[string]time.Time),
		timersAdd: make(map[string]*timerAddValue),
	}
	return context.WithValue(ctx, timeItKey(0), value)
}

func timeIt(ctx context.Context, name string) {
	ctxValue := ctx.Value(timeItKey(0)).(*timeItType)
	if ctxValue == nil {
		fmt.Printf("timeIt: Invalid context")
		return
	}
	timerValues := ctxValue.timers
	if _, present := timerValues[name]; present {
		fmt.Printf("timeIt: Timer %s already started\n", name)
		return
	}
	timerValues[name] = time.Now()
}

func timeEnd(ctx context.Context, name string) int64 {
	ctxValue := ctx.Value(timeItKey(0)).(*timeItType)
	if ctxValue == nil {
		return 0
	}
	timerValues := ctxValue.timers
	start, present := timerValues[name]
	if !present {
		fmt.Printf("timeEnd: Timer %s has not started\n", name)
		return 0
	}
	end := time.Now()
	delete(timerValues, name)
	dur := end.Sub(start).Milliseconds()
	if len(ctxValue.results) == 0 {
		ctxValue.results = fmt.Sprintf("%s:%dms", name, dur)
	} else {
		ctxValue.results = fmt.Sprintf("%s %s:%dms", ctxValue.results, name, dur)
	}
	return dur
}

func timeResults(ctx context.Context) string {
	ctxValue := ctx.Value(timeItKey(0)).(*timeItType)
	if ctxValue == nil {
		return ""
	}
	return ctxValue.results
}

func timeAddIt(ctx context.Context, name string, start bool) {
	ctxValue := ctx.Value(timeItKey(0)).(*timeItType)
	if ctxValue == nil {
		fmt.Printf("timeAddIt: Invalid context")
		return
	}
	tAdd, present := ctxValue.timersAdd[name]

	if present {
		if start {
			tAdd.start = time.Now()
		} else {
			end := time.Now()
			tAdd.num++
			tAdd.microSeconds += end.Sub(tAdd.start).Microseconds()
			tAdd.start = end
		}
	} else {
		ctxValue.timersAdd[name] = &timerAddValue{
			start:        time.Now(),
			microSeconds: 0,
			num:          1,
		}
	}
}

func timeAddEnd(ctx context.Context, name string) (int64, int) {
	ctxValue := ctx.Value(timeItKey(0)).(*timeItType)
	if ctxValue == nil {
		return 0, 0
	}
	tAdd, present := ctxValue.timersAdd[name]
	if !present {
		fmt.Printf("timeAddEnd: Timer %s has not started\n", name)
		return 0, 0
	}

	delete(ctxValue.timersAdd, name)
	if len(ctxValue.results) == 0 {
		ctxValue.results = fmt.Sprintf("%s:'%d µs, %d runs'", name, tAdd.microSeconds, tAdd.num)
	} else {
		ctxValue.results = fmt.Sprintf("%s %s:'%d µs, %d runs'", ctxValue.results, name, tAdd.microSeconds, tAdd.num)
	}
	return tAdd.microSeconds, tAdd.num
}
