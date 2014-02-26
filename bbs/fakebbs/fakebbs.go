package fakebbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	"time"
)

type FakePresence struct {
	Removed bool
}

func (p *FakePresence) Remove() error {
	p.Removed = true
	return nil
}

type FakeExecutorBBS struct {
	CallsToConverge int
	LockIsGrabbable bool
	ErrorOnGrabLock error

	MaintainingPresenceHeartbeatInterval uint64
	MaintainingPresenceExecutorID        string
	MaintainingPresencePresence          *FakePresence
	MaintainingPresenceErrorChannel      chan error
	MaintainingPresenceError             error

	ClaimedRunOnce  models.RunOnce
	ClaimRunOnceErr error

	StartedRunOnce  models.RunOnce
	StartRunOnceErr error

	CompletedRunOnce   models.RunOnce
	CompleteRunOnceErr error
}

func NewFakeExecutorBBS() *FakeExecutorBBS {
	return &FakeExecutorBBS{}
}

func (fakeBBS *FakeExecutorBBS) SetTimeToClaim(time.Duration) {}

func (fakeBBS *FakeExecutorBBS) MaintainExecutorPresence(heartbeatIntervalInSeconds uint64, executorID string) (bbs.PresenceInterface, chan error, error) {
	fakeBBS.MaintainingPresenceHeartbeatInterval = heartbeatIntervalInSeconds
	fakeBBS.MaintainingPresenceExecutorID = executorID
	fakeBBS.MaintainingPresencePresence = &FakePresence{}
	fakeBBS.MaintainingPresenceErrorChannel = make(chan error)

	return fakeBBS.MaintainingPresencePresence, fakeBBS.MaintainingPresenceErrorChannel, fakeBBS.MaintainingPresenceError
}

func (fakeBBS *FakeExecutorBBS) WatchForDesiredRunOnce() (<-chan models.RunOnce, chan<- bool, <-chan error) {
	return nil, nil, nil
}

func (fakeBBS *FakeExecutorBBS) ClaimRunOnce(runOnce models.RunOnce) error {
	fakeBBS.ClaimedRunOnce = runOnce
	return fakeBBS.ClaimRunOnceErr
}

func (fakeBBS *FakeExecutorBBS) StartRunOnce(runOnce models.RunOnce) error {
	fakeBBS.StartedRunOnce = runOnce
	return fakeBBS.StartRunOnceErr
}

func (fakeBBS *FakeExecutorBBS) CompleteRunOnce(runOnce models.RunOnce) error {
	fakeBBS.CompletedRunOnce = runOnce
	return fakeBBS.CompleteRunOnceErr
}

func (fakeBBS *FakeExecutorBBS) ConvergeRunOnce() {
	fakeBBS.CallsToConverge++
}

func (fakeBBS *FakeExecutorBBS) GrabRunOnceLock(time.Duration) (bool, error) {
	return fakeBBS.LockIsGrabbable, fakeBBS.ErrorOnGrabLock
}

type FakeStagerBBS struct {
	ResolvedRunOnce   models.RunOnce
	ResolveRunOnceErr error

	CalledCompletedRunOnce  chan bool
	CompletedRunOnceChan    chan models.RunOnce
	CompletedRunOnceErrChan chan error
}

func NewFakeStagerBBS() *FakeStagerBBS {
	return &FakeStagerBBS{
		CalledCompletedRunOnce: make(chan bool),
	}
}

func (fakeBBS *FakeStagerBBS) WatchForCompletedRunOnce() (<-chan models.RunOnce, chan<- bool, <-chan error) {
	fakeBBS.CompletedRunOnceChan = make(chan models.RunOnce)
	fakeBBS.CompletedRunOnceErrChan = make(chan error)
	fakeBBS.CalledCompletedRunOnce <- true
	return fakeBBS.CompletedRunOnceChan, nil, fakeBBS.CompletedRunOnceErrChan
}

func (fakeBBS *FakeStagerBBS) DesireRunOnce(runOnce models.RunOnce) error {
	panic("implement me!")
}

func (fakeBBS *FakeStagerBBS) ResolveRunOnce(runOnce models.RunOnce) error {
	fakeBBS.ResolvedRunOnce = runOnce
	return fakeBBS.ResolveRunOnceErr
}

func (fakeBBS *FakeStagerBBS) GetAvailableFileServer() (string, error) {
	panic("implement me!")
}
