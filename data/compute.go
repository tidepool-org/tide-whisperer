package data

import (
	"context"
	"errors"
	"math"
	"strconv"
	"time"

	"github.com/tidepool-org/go-common/clients/mongo"
	"github.com/tidepool-org/tide-whisperer/store"
)

type (
	// summaryGlyLimits is the return struct of getDataSummaryGlyLimits()
	summaryGlyLimits struct {
		hypoMgdl   float64
		hyperMgdl  float64
		hypoMmoll  float64
		hyperMmoll float64
		// The unit found in the parameters
		unit string
	}
	// summaryTresholds is the return struct of getDataSummaryThresholds()
	summaryTresholds struct {
		totalNumBgValues      int
		percentTimeInRange    int
		percentTimeBelowRange int
	}
)

const (
	patientGlyHypoLimitMgdl   float64 = 70.0
	patientGlyHyperLimitMgdl  float64 = 180
	patientGlyHypoLimitMmoll  float64 = 3.9
	patientGlyHyperLimitMmoll float64 = 10.0
)

func getParameterLimitValue(p deviceParameter) (mgdl float64, mmoll float64, unit string, err error) {
	limitValue, err := strconv.ParseFloat(p.Value, 64)
	if err != nil {
		return 0, 0, "", err
	}
	if p.Unit == unitMgdL {
		mgdl = limitValue
		mmoll, err = convertBG(mgdl, unitMgdL)
	} else if p.Unit == unitMmolL {
		mmoll = limitValue
		mgdl, err = convertBG(mgdl, unitMmolL)
	}
	return mgdl, mmoll, p.Unit, err
}

// Get the first / last data time for the specified user
func (a *API) getDataSummaryRangeV1(ctx context.Context, traceID string, userID string, numDays int64) (*store.Date, string, error) {
	timeIt(ctx, "getDataRange")
	defer timeEnd(ctx, "getDataRange")

	dates, err := a.store.GetDataRangeV1(ctx, traceID, userID)
	if err != nil {
		return nil, "", err
	}

	// If no data, we can stop here
	if dates == nil || dates.Start == "" || dates.End == "" {
		return nil, "", errors.New(errorNotfound.Code)
	}

	endRange, err := time.Parse(time.RFC3339Nano, dates.End)
	if err != nil {
		return nil, "", err
	}
	subDaysDuration := time.Duration(-24 * int64(time.Hour) * numDays)
	startTime := endRange.Add(subDaysDuration)

	return dates, startTime.Format(time.RFC3339), nil
}

// Get the current device parameters (to get hypo/hyper values)
func (a *API) getDataSummaryGlyLimits(ctx context.Context, traceID, userID string) (*summaryGlyLimits, error) {
	var err error
	var glyLimits *summaryGlyLimits = &summaryGlyLimits{
		hypoMgdl:   patientGlyHypoLimitMgdl,
		hyperMgdl:  patientGlyHyperLimitMgdl,
		hypoMmoll:  patientGlyHypoLimitMmoll,
		hyperMmoll: patientGlyHyperLimitMmoll,
		unit:       unitMgdL,
	}
	var iterPumpSettings mongo.StorageIterator

	timeIt(ctx, "getLastPumpSettings")
	iterPumpSettings, err = a.store.GetLatestPumpSettingsV1(ctx, traceID, userID)
	if err != nil {
		return nil, err
	}
	defer iterPumpSettings.Close(ctx)

	// This algorithm assume that the parameters units are the same for hypo and hyper
	if iterPumpSettings.Next(ctx) {
		var pumpSettings PumpSettings
		err = iterPumpSettings.Decode(&pumpSettings)
		if err == nil {
			haveHypo := false
			haveHyper := false
			for _, parameter := range pumpSettings.Payload.Parameters {
				if !haveHypo && parameter.Name == "PATIENT_GLY_HYPO_LIMIT" {
					haveHypo = true
					glyLimits.hypoMgdl, glyLimits.hypoMmoll, glyLimits.unit, err = getParameterLimitValue(parameter)
				} else if !haveHyper && parameter.Name == "PATIENT_GLY_HYPER_LIMIT" {
					haveHyper = true
					glyLimits.hyperMgdl, glyLimits.hyperMmoll, glyLimits.unit, err = getParameterLimitValue(parameter)
				}
				if err != nil {
					a.logger.Printf("{%s} - getDataSummaryGlyLimits for %s: Parse device parameter value error: %s (%v)", traceID, userID, err.Error(), parameter)
					break
				}
				if haveHypo && haveHyper {
					break
				}
			}
		} else {
			a.logger.Printf("{%s} - getDataSummaryGlyLimits for %s: Decode pump settings error: %s", traceID, userID, err.Error())
		}
	} // else: No pump settings (Use the default values)!
	timeEnd(ctx, "getLastPumpSettings")

	return glyLimits, nil
}

// Get the TIR & TBR percent, and the number of BG results used
func (a *API) getDataSummaryThresholds(ctx context.Context, traceID string, userID string, startTime string, glyLimits *summaryGlyLimits) (*summaryTresholds, error) {
	var totalNumBgValues int = 0
	var totalNumInRange int = 0
	var totalNumBelowRange int = 0
	var totalWrongUnit int = 0
	var tresholds *summaryTresholds = &summaryTresholds{}

	timeIt(ctx, "getBG")
	iterBG, err := a.store.GetCbgForSummaryV1(ctx, traceID, userID, startTime)
	if err != nil {
		return nil, err
	}
	defer iterBG.Close(ctx)
	timeEnd(ctx, "getBG")

	timeIt(ctx, "computeBG")
	for iterBG.Next(ctx) {
		bgValue := simplifiedBgDatum{}
		err = iterBG.Decode(&bgValue)
		if err != nil {
			return nil, err
		}
		totalNumBgValues++
		if bgValue.Unit == unitMgdL {
			if bgValue.Value < glyLimits.hypoMgdl {
				totalNumBelowRange++
			} else if bgValue.Value < glyLimits.hyperMgdl {
				totalNumInRange++
			}
		} else if bgValue.Unit == unitMmolL {
			if bgValue.Value < glyLimits.hypoMmoll {
				totalNumBelowRange++
			} else if bgValue.Value < glyLimits.hyperMmoll {
				totalNumInRange++
			}
		} else {
			totalWrongUnit++
		}
	}
	timeEnd(ctx, "computeBG")

	if totalNumBgValues > 0 {
		tresholds.totalNumBgValues = totalNumBgValues
		tresholds.percentTimeBelowRange = int(math.Round(100.0 * float64(totalNumBelowRange) / float64(totalNumBgValues)))
		tresholds.percentTimeInRange = int(math.Round(100.0 * float64(totalNumInRange) / float64(totalNumBgValues)))
	}
	if totalWrongUnit > 0 {
		a.logger.Printf("Found %d wrong unit in CBG data for user %s", totalWrongUnit, userID)
	}

	return tresholds, nil
}
