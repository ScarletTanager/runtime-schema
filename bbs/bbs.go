package bbs

import (
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/lock_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/lrp_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/services_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/start_auction_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/stop_auction_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/task_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/timeprovider"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/ifrit"
)

//Bulletin Board System/Store

type ReceptorBBS interface {
	//task
	DesireTask(models.Task) error
	GetAllTasks() ([]models.Task, error)
	GetAllTasksByDomain(domain string) ([]models.Task, error)
	GetTaskByGuid(taskGuid string) (models.Task, error)
	ResolvingTask(taskGuid string) error
	ResolveTask(taskGuid string) error
	CancelTask(taskGuid string) error
	WatchForCompletedTask() (<-chan models.Task, chan<- bool, <-chan error)

	//desired lrp
	DesireLRP(models.DesiredLRP) error
	UpdateDesiredLRP(processGuid string, update models.DesiredLRPUpdate) error
	RemoveDesiredLRPByProcessGuid(processGuid string) error
	GetAllDesiredLRPs() ([]models.DesiredLRP, error)
	GetAllDesiredLRPsByDomain(domain string) ([]models.DesiredLRP, error)
	GetDesiredLRPByProcessGuid(processGuid string) (models.DesiredLRP, error)

	//actual lrp
	GetAllActualLRPs() ([]models.ActualLRP, error)
	GetAllActualLRPsByDomain(domain string) ([]models.ActualLRP, error)
	GetActualLRPsByProcessGuid(string) ([]models.ActualLRP, error)
	GetActualLRPsByProcessGuidAndIndex(string, int) ([]models.ActualLRP, error)
	RequestStopLRPInstances(stopInstances []models.StopLRPInstance) error

	// cells
	GetAllCells() ([]models.CellPresence, error)

	// freshness
	BumpFreshness(models.Freshness) error
	Freshnesses() ([]models.Freshness, error)
}

type RepBBS interface {
	//services
	NewCellHeartbeat(cellPresence models.CellPresence, interval time.Duration) ifrit.Runner

	//task
	WatchForDesiredTask() (<-chan models.Task, chan<- bool, <-chan error)
	ClaimTask(taskGuid string, cellID string) error
	StartTask(taskGuid string, cellID string) error
	GetTaskByGuid(taskGuid string) (models.Task, error)
	GetAllTasksByCellID(cellID string) ([]models.Task, error)
	CompleteTask(taskGuid string, failed bool, failureReason string, result string) error

	//lrp
	GetActualLRPsByProcessGuid(string) ([]models.ActualLRP, error)
	GetAllActualLRPsByCellID(cellID string) ([]models.ActualLRP, error)
	ReportActualLRPAsStarting(processGuid, instanceGuid, cellID, domain string, index int) (models.ActualLRP, error)
	ReportActualLRPAsRunning(lrp models.ActualLRP, cellId string) error
	RemoveActualLRP(lrp models.ActualLRP) error
	RemoveActualLRPForIndex(processGuid string, index int, instanceGuid string) error
	WatchForStopLRPInstance() (<-chan models.StopLRPInstance, chan<- bool, <-chan error)
	ResolveStopLRPInstance(stopInstance models.StopLRPInstance) error
}

type ConvergerBBS interface {
	//lrp
	ConvergeLRPs()
	GetActualLRPsByProcessGuid(string) ([]models.ActualLRP, error)
	RequestStopLRPInstance(stopInstance models.StopLRPInstance) error
	WatchForDesiredLRPChanges() (<-chan models.DesiredLRPChange, chan<- bool, <-chan error)

	//start auction
	ConvergeLRPStartAuctions(kickPendingDuration time.Duration, expireClaimedDuration time.Duration)
	RequestLRPStartAuction(models.LRPStartAuction) error

	//stop auction
	ConvergeLRPStopAuctions(kickPendingDuration time.Duration, expireClaimedDuration time.Duration)
	RequestLRPStopAuction(models.LRPStopAuction) error

	//task
	ConvergeTask(timeToClaim, convergenceInterval, timeToResolve time.Duration)

	//lock
	NewConvergeLock(convergerID string, interval time.Duration) ifrit.Runner
}

type TPSBBS interface {
	//lrp
	GetActualLRPsByProcessGuid(string) ([]models.ActualLRP, error)
}

type NsyncBBS interface {
	// lrp
	DesireLRP(models.DesiredLRP) error
	RemoveDesiredLRPByProcessGuid(guid string) error
	GetAllDesiredLRPsByDomain(domain string) ([]models.DesiredLRP, error)
	ChangeDesiredLRP(change models.DesiredLRPChange) error
	BumpFreshness(freshness models.Freshness) error

	//lock
	NewNsyncBulkerLock(bulkerID string, interval time.Duration) ifrit.Runner
	NewNsyncListenerLock(listenerID string, interval time.Duration) ifrit.Runner
}

type AuctioneerBBS interface {
	//services
	GetAllCells() ([]models.CellPresence, error)

	//start auction
	WatchForLRPStartAuction() (<-chan models.LRPStartAuction, chan<- bool, <-chan error)
	ClaimLRPStartAuction(models.LRPStartAuction) error
	ResolveLRPStartAuction(models.LRPStartAuction) error

	//stop auction
	WatchForLRPStopAuction() (<-chan models.LRPStopAuction, chan<- bool, <-chan error)
	ClaimLRPStopAuction(models.LRPStopAuction) error
	ResolveLRPStopAuction(models.LRPStopAuction) error

	//lock
	NewAuctioneerLock(auctioneerID string, interval time.Duration) ifrit.Runner
}

type MetricsBBS interface {
	//task
	GetAllTasks() ([]models.Task, error)

	//services
	GetServiceRegistrations() (models.ServiceRegistrations, error)

	//lrps
	Freshnesses() ([]models.Freshness, error)
	GetAllDesiredLRPs() ([]models.DesiredLRP, error)
	GetAllActualLRPs() ([]models.ActualLRP, error)

	//lock
	NewRuntimeMetricsLock(runtimeMetricsID string, interval time.Duration) ifrit.Runner
}

type RouteEmitterBBS interface {
	// lrp
	WatchForDesiredLRPChanges() (<-chan models.DesiredLRPChange, chan<- bool, <-chan error)
	WatchForActualLRPChanges() (<-chan models.ActualLRPChange, chan<- bool, <-chan error)
	GetAllDesiredLRPs() ([]models.DesiredLRP, error)
	GetRunningActualLRPs() ([]models.ActualLRP, error)
	GetDesiredLRPByProcessGuid(processGuid string) (models.DesiredLRP, error)
	GetRunningActualLRPsByProcessGuid(processGuid string) ([]models.ActualLRP, error)

	//lock
	NewRouteEmitterLock(emitterID string, interval time.Duration) ifrit.Runner
}

type VeritasBBS interface {
	//task
	GetAllTasks() ([]models.Task, error)

	//lrp
	GetAllDesiredLRPs() ([]models.DesiredLRP, error)
	GetAllActualLRPs() ([]models.ActualLRP, error)
	GetAllStopLRPInstances() ([]models.StopLRPInstance, error)
	DesireLRP(models.DesiredLRP) error
	RemoveDesiredLRPByProcessGuid(guid string) error
	Freshnesses() ([]models.Freshness, error)

	//start auctions
	GetAllLRPStartAuctions() ([]models.LRPStartAuction, error)

	//stop auctions
	GetAllLRPStopAuctions() ([]models.LRPStopAuction, error)

	//services
	GetAllCells() ([]models.CellPresence, error)
}

func NewReceptorBBS(store storeadapter.StoreAdapter, timeProvider timeprovider.TimeProvider, logger lager.Logger) ReceptorBBS {
	return NewBBS(store, timeProvider, logger)
}

func NewRepBBS(store storeadapter.StoreAdapter, timeProvider timeprovider.TimeProvider, logger lager.Logger) RepBBS {
	return NewBBS(store, timeProvider, logger)
}

func NewConvergerBBS(store storeadapter.StoreAdapter, timeProvider timeprovider.TimeProvider, logger lager.Logger) ConvergerBBS {
	return NewBBS(store, timeProvider, logger)
}

func NewNsyncBBS(store storeadapter.StoreAdapter, timeProvider timeprovider.TimeProvider, logger lager.Logger) NsyncBBS {
	return NewBBS(store, timeProvider, logger)
}

func NewAuctioneerBBS(store storeadapter.StoreAdapter, timeProvider timeprovider.TimeProvider, logger lager.Logger) AuctioneerBBS {
	return NewBBS(store, timeProvider, logger)
}

func NewMetricsBBS(store storeadapter.StoreAdapter, timeProvider timeprovider.TimeProvider, logger lager.Logger) MetricsBBS {
	return NewBBS(store, timeProvider, logger)
}

func NewRouteEmitterBBS(store storeadapter.StoreAdapter, timeProvider timeprovider.TimeProvider, logger lager.Logger) RouteEmitterBBS {
	return NewBBS(store, timeProvider, logger)
}

func NewTPSBBS(store storeadapter.StoreAdapter, timeProvider timeprovider.TimeProvider, logger lager.Logger) TPSBBS {
	return NewBBS(store, timeProvider, logger)
}

func NewVeritasBBS(store storeadapter.StoreAdapter, timeProvider timeprovider.TimeProvider, logger lager.Logger) VeritasBBS {
	return NewBBS(store, timeProvider, logger)
}

func NewBBS(store storeadapter.StoreAdapter, timeProvider timeprovider.TimeProvider, logger lager.Logger) *BBS {
	return &BBS{
		LockBBS:         lock_bbs.New(store, logger.Session("lock-bbs")),
		LRPBBS:          lrp_bbs.New(store, timeProvider, logger.Session("lrp-bbs")),
		StartAuctionBBS: start_auction_bbs.New(store, timeProvider, logger.Session("lrp-start-auction-bbs")),
		StopAuctionBBS:  stop_auction_bbs.New(store, timeProvider, logger.Session("lrp-stop-auction-bbs")),
		ServicesBBS:     services_bbs.New(store, logger.Session("services-bbs")),
		TaskBBS:         task_bbs.New(store, timeProvider, logger.Session("task-bbs")),
	}
}

type BBS struct {
	*lock_bbs.LockBBS
	*lrp_bbs.LRPBBS
	*start_auction_bbs.StartAuctionBBS
	*stop_auction_bbs.StopAuctionBBS
	*services_bbs.ServicesBBS
	*task_bbs.TaskBBS
}
