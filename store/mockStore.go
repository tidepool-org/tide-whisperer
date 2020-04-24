package store

import (
	"context"
	"encoding/json"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
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

type MockStoreClient struct {
	PingError   bool
	DeviceModel string

	ParametersHistory bson.M

	DeviceData          []string
	GetDeviceDataCall   Params
	GetDeviceDataCalled bool

	TimeInRangeData          []string
	GetTimeInRangeDataCall   AggParams
	GetTimeInRangeDataCalled bool
}

func NewMockStoreClient() *MockStoreClient {
	return &MockStoreClient{
		PingError:           false,
		DeviceModel:         "test",
		GetDeviceDataCalled: false,
	}
}

func (c *MockStoreClient) EnablePingError() {
	c.PingError = true
}

func (c *MockStoreClient) DisablePingError() {
	c.PingError = false
}

func (c *MockStoreClient) EnsureIndexes() error {
	return nil
}
func (c *MockStoreClient) WithContext(ctx context.Context) Storage {
	return c
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
func (c *MockStoreClient) GetDeviceData(p *Params) (StorageIterator, error) {
	c.GetDeviceDataCall = *p
	c.GetDeviceDataCalled = true
	if c.DeviceData != nil {
		iter := &MockStoreIterator{
			numIter: -1,
			maxIter: 1,
			data:    c.DeviceData,
		}
		return iter, nil
	}
	return nil, nil
}
func (c *MockStoreClient) GetDexcomDataSource(userID string) (bson.M, error) {
	return nil, nil
}
func (c *MockStoreClient) GetDiabeloopParametersHistory(userID string, levels []int) (bson.M, error) {
	if c.ParametersHistory != nil {
		return c.ParametersHistory, nil
	}
	return nil, nil
}
func (c *MockStoreClient) GetLoopableMedtronicDirectUploadIdsAfter(userID string, date string) ([]string, error) {
	return nil, nil
}
func (c *MockStoreClient) GetDeviceModel(userID string) (string, error) {
	return c.DeviceModel, nil
}
func (c *MockStoreClient) GetTimeInRangeData(p *AggParams, logQuery bool) (StorageIterator, error) {
	c.GetTimeInRangeDataCall = *p
	c.GetTimeInRangeDataCalled = true
	if c.TimeInRangeData != nil {
		iter := &MockStoreIterator{
			numIter: -1,
			maxIter: 1,
			data:    c.TimeInRangeData,
		}
		return iter, nil
	}
	return nil, nil
}
func (c *MockStoreClient) HasMedtronicDirectData(userID string) (bool, error) {
	return false, nil
}
func (c *MockStoreClient) HasMedtronicLoopDataAfter(userID string, date string) (bool, error) {
	return false, nil
}
