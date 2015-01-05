// This file was generated by counterfeiter
package fakes

import (
	"sync"

	"github.com/cloudfoundry-incubator/runtime-schema/cb"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

type FakeCellClient struct {
	StopLRPInstanceStub        func(cellAddr string, key models.ActualLRPKey, containerKey models.ActualLRPContainerKey) error
	stopLRPInstanceMutex       sync.RWMutex
	stopLRPInstanceArgsForCall []struct {
		cellAddr     string
		key          models.ActualLRPKey
		containerKey models.ActualLRPContainerKey
	}
	stopLRPInstanceReturns struct {
		result1 error
	}
}

func (fake *FakeCellClient) StopLRPInstance(cellAddr string, key models.ActualLRPKey, containerKey models.ActualLRPContainerKey) error {
	fake.stopLRPInstanceMutex.Lock()
	fake.stopLRPInstanceArgsForCall = append(fake.stopLRPInstanceArgsForCall, struct {
		cellAddr     string
		key          models.ActualLRPKey
		containerKey models.ActualLRPContainerKey
	}{cellAddr, key, containerKey})
	fake.stopLRPInstanceMutex.Unlock()
	if fake.StopLRPInstanceStub != nil {
		return fake.StopLRPInstanceStub(cellAddr, key, containerKey)
	} else {
		return fake.stopLRPInstanceReturns.result1
	}
}

func (fake *FakeCellClient) StopLRPInstanceCallCount() int {
	fake.stopLRPInstanceMutex.RLock()
	defer fake.stopLRPInstanceMutex.RUnlock()
	return len(fake.stopLRPInstanceArgsForCall)
}

func (fake *FakeCellClient) StopLRPInstanceArgsForCall(i int) (string, models.ActualLRPKey, models.ActualLRPContainerKey) {
	fake.stopLRPInstanceMutex.RLock()
	defer fake.stopLRPInstanceMutex.RUnlock()
	return fake.stopLRPInstanceArgsForCall[i].cellAddr, fake.stopLRPInstanceArgsForCall[i].key, fake.stopLRPInstanceArgsForCall[i].containerKey
}

func (fake *FakeCellClient) StopLRPInstanceReturns(result1 error) {
	fake.StopLRPInstanceStub = nil
	fake.stopLRPInstanceReturns = struct {
		result1 error
	}{result1}
}

var _ cb.CellClient = new(FakeCellClient)
