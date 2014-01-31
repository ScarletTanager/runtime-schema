package fakebbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	"time"
)

type FakeExecutorBBS struct {
	CallsToConverge int
	LockIsGrabbable bool
	ErrorOnGrabLock error
}

func (fakeBBS *FakeExecutorBBS) WatchForDesiredRunOnce() (<-chan models.RunOnce, chan<- bool, <-chan error) {
	return nil, nil, nil
}

func (fakeBBS *FakeExecutorBBS) ClaimRunOnce(models.RunOnce) error {
	return nil
}

func (fakeBBS *FakeExecutorBBS) StartRunOnce(models.RunOnce) error {
	return nil
}

func (fakeBBS *FakeExecutorBBS) CompletedRunOnce(models.RunOnce) error {
	return nil
}

func (fakeBBS *FakeExecutorBBS) ConvergeRunOnce() {
	fakeBBS.CallsToConverge++
}

func (fakeBBS *FakeExecutorBBS) GrabRunOnceLock(time.Duration) (bool, error) {
	return fakeBBS.LockIsGrabbable, fakeBBS.ErrorOnGrabLock
}
