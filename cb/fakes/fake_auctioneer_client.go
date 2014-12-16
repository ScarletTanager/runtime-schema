// This file was generated by counterfeiter
package fakes

import (
	"sync"

	"github.com/cloudfoundry-incubator/runtime-schema/cb"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

type FakeAuctioneerClient struct {
	RequestLRPStartAuctionStub        func(auctioneerURL string, startAuction models.LRPStartAuction) error
	requestLRPStartAuctionMutex       sync.RWMutex
	requestLRPStartAuctionArgsForCall []struct {
		auctioneerURL string
		startAuction  models.LRPStartAuction
	}
	requestLRPStartAuctionReturns struct {
		result1 error
	}
	RequestTaskAuctionStub        func(auctioneerURL string, task models.Task) error
	requestTaskAuctionMutex       sync.RWMutex
	requestTaskAuctionArgsForCall []struct {
		auctioneerURL string
		task          models.Task
	}
	requestTaskAuctionReturns struct {
		result1 error
	}
}

func (fake *FakeAuctioneerClient) RequestLRPStartAuction(auctioneerURL string, startAuction models.LRPStartAuction) error {
	fake.requestLRPStartAuctionMutex.Lock()
	fake.requestLRPStartAuctionArgsForCall = append(fake.requestLRPStartAuctionArgsForCall, struct {
		auctioneerURL string
		startAuction  models.LRPStartAuction
	}{auctioneerURL, startAuction})
	fake.requestLRPStartAuctionMutex.Unlock()
	if fake.RequestLRPStartAuctionStub != nil {
		return fake.RequestLRPStartAuctionStub(auctioneerURL, startAuction)
	} else {
		return fake.requestLRPStartAuctionReturns.result1
	}
}

func (fake *FakeAuctioneerClient) RequestLRPStartAuctionCallCount() int {
	fake.requestLRPStartAuctionMutex.RLock()
	defer fake.requestLRPStartAuctionMutex.RUnlock()
	return len(fake.requestLRPStartAuctionArgsForCall)
}

func (fake *FakeAuctioneerClient) RequestLRPStartAuctionArgsForCall(i int) (string, models.LRPStartAuction) {
	fake.requestLRPStartAuctionMutex.RLock()
	defer fake.requestLRPStartAuctionMutex.RUnlock()
	return fake.requestLRPStartAuctionArgsForCall[i].auctioneerURL, fake.requestLRPStartAuctionArgsForCall[i].startAuction
}

func (fake *FakeAuctioneerClient) RequestLRPStartAuctionReturns(result1 error) {
	fake.RequestLRPStartAuctionStub = nil
	fake.requestLRPStartAuctionReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeAuctioneerClient) RequestTaskAuction(auctioneerURL string, task models.Task) error {
	fake.requestTaskAuctionMutex.Lock()
	fake.requestTaskAuctionArgsForCall = append(fake.requestTaskAuctionArgsForCall, struct {
		auctioneerURL string
		task          models.Task
	}{auctioneerURL, task})
	fake.requestTaskAuctionMutex.Unlock()
	if fake.RequestTaskAuctionStub != nil {
		return fake.RequestTaskAuctionStub(auctioneerURL, task)
	} else {
		return fake.requestTaskAuctionReturns.result1
	}
}

func (fake *FakeAuctioneerClient) RequestTaskAuctionCallCount() int {
	fake.requestTaskAuctionMutex.RLock()
	defer fake.requestTaskAuctionMutex.RUnlock()
	return len(fake.requestTaskAuctionArgsForCall)
}

func (fake *FakeAuctioneerClient) RequestTaskAuctionArgsForCall(i int) (string, models.Task) {
	fake.requestTaskAuctionMutex.RLock()
	defer fake.requestTaskAuctionMutex.RUnlock()
	return fake.requestTaskAuctionArgsForCall[i].auctioneerURL, fake.requestTaskAuctionArgsForCall[i].task
}

func (fake *FakeAuctioneerClient) RequestTaskAuctionReturns(result1 error) {
	fake.RequestTaskAuctionStub = nil
	fake.requestTaskAuctionReturns = struct {
		result1 error
	}{result1}
}

var _ cb.AuctioneerClient = new(FakeAuctioneerClient)
