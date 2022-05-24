package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	goComMgo "github.com/tidepool-org/go-common/clients/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type MockStoreIterator struct {
	numIter int
	maxIter int
	data    []string
}

func (i *MockStoreIterator) Next(ctx context.Context) bool {
	i.numIter++
	return i.numIter < i.maxIter
}
func (i *MockStoreIterator) Close(ctx context.Context) error {
	return nil
}
func (i *MockStoreIterator) Decode(val interface{}) error {
	json.Unmarshal([]byte(i.data[i.numIter]), &val)
	return nil
}

// MockStoreClient use for unit tests
type MockStoreClient struct {
	PingError   bool
	DeviceModel string

	ParametersHistory    bson.M
	BasalSecurityProfile *DbProfile

	DataRangeV1 []string
	DataV1      []string
	DataIDV1    []string
	DataBGV1    []string
	DataPSV1    *string
}

func NewMockStoreClient() *MockStoreClient {
	return &MockStoreClient{
		PingError:   false,
		DeviceModel: "test",
	}
}

func (c *MockStoreClient) EnablePingError() {
	c.PingError = true
}

func (c *MockStoreClient) DisablePingError() {
	c.PingError = false
}

func (c *MockStoreClient) Close() error {
	return nil
}
func (c *MockStoreClient) Ping() error {
	if c.PingError {
		return errors.New("Mock Ping Error")
	}
	return nil
}
func (c *MockStoreClient) PingOK() bool {
	return !c.PingError
}
func (c *MockStoreClient) Collection(collectionName string, databaseName ...string) *mongo.Collection {
	return nil
}
func (c *MockStoreClient) WaitUntilStarted() {}
func (c *MockStoreClient) Start()            {}

func (c *MockStoreClient) GetDiabeloopParametersHistory(ctx context.Context, userID string, levels []int) (bson.M, error) {
	if c.ParametersHistory != nil {
		return c.ParametersHistory, nil
	}
	return nil, nil
}

// GetDataRangeV1 mock func, return nil,nil
func (c *MockStoreClient) GetDataRangeV1(ctx context.Context, traceID string, userID string) (*Date, error) {
	if c.DataRangeV1 != nil && len(c.DataRangeV1) == 2 {
		return &Date{
			Start: c.DataRangeV1[0],
			End:   c.DataRangeV1[1],
		}, nil
	}
	return nil, fmt.Errorf("{%s} - [%s] - No data", traceID, userID)
}

// GetDataV1 v1 api mock call to fetch diabetes data
func (c *MockStoreClient) GetDataV1(ctx context.Context, traceID string, userID string, dates *Date, excludedType []string) (goComMgo.StorageIterator, error) {
	if c.DataV1 != nil {
		return &MockStoreIterator{
			numIter: -1,
			maxIter: len(c.DataV1),
			data:    c.DataV1,
		}, nil
	}
	return nil, fmt.Errorf("{%s} - [%s] - No data", traceID, userID)
}

// GetLatestPumpSettingsV1 return the latest type == "pumpSettings"
func (c *MockStoreClient) GetLatestPumpSettingsV1(ctx context.Context, traceID string, userID string) (goComMgo.StorageIterator, error) {
	if c.DataPSV1 != nil {
		return &MockStoreIterator{
			numIter: -1,
			maxIter: 1,
			data:    []string{*c.DataPSV1},
		}, nil
	}
	return nil, fmt.Errorf("{%s} - [%s] - No data", traceID, userID)
}

func (c *MockStoreClient) GetLatestBasalSecurityProfile(ctx context.Context, traceID string, userID string) (*DbProfile, error) {
	if c.BasalSecurityProfile != nil {
		return c.BasalSecurityProfile, nil
	}
	return nil, nil
}

// GetUploadDataV1 Fetch upload data from theirs upload ids, using the $in query parameter
func (c *MockStoreClient) GetUploadDataV1(ctx context.Context, traceID string, uploadIds []string) (goComMgo.StorageIterator, error) {
	if c.DataIDV1 != nil {
		return &MockStoreIterator{
			numIter: -1,
			maxIter: len(c.DataIDV1),
			data:    c.DataIDV1,
		}, nil
	}
	return nil, fmt.Errorf("{%s} - No data", traceID)
}

// GetCbgForSummaryV1 return the cbg/smbg values for the given user starting at startDate
func (c *MockStoreClient) GetCbgForSummaryV1(ctx context.Context, traceID string, userID string, startDate string) (goComMgo.StorageIterator, error) {
	if c.DataBGV1 != nil {
		return &MockStoreIterator{
			numIter: -1,
			maxIter: len(c.DataBGV1),
			data:    c.DataBGV1,
		}, nil
	}
	return nil, fmt.Errorf("{%s} - No data", traceID)
}
