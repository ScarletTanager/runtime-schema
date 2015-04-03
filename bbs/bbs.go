package bbs

import (
	"time"

	"github.com/cloudfoundry-incubator/consuladapter"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/domain_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/lock_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/lrp_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/services_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/task_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/cb"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/ifrit"
)

//Bulletin Board System/Store

//go:generate counterfeiter -o fake_bbs/fake_receptor_bbs.go . ReceptorBBS
type ReceptorBBS interface {
	//task
	DesireTask(lager.Logger, models.Task) error
	Tasks(logger lager.Logger) ([]models.Task, error)
	TasksByDomain(logger lager.Logger, domain string) ([]models.Task, error)
	TaskByGuid(taskGuid string) (models.Task, error)
	ResolvingTask(logger lager.Logger, taskGuid string) error
	ResolveTask(logger lager.Logger, taskGuid string) error
	CancelTask(logger lager.Logger, taskGuid string) error

	//desired lrp
	DesireLRP(lager.Logger, models.DesiredLRP) error
	UpdateDesiredLRP(logger lager.Logger, processGuid string, update models.DesiredLRPUpdate) error
	RemoveDesiredLRPByProcessGuid(logger lager.Logger, processGuid string) error
	DesiredLRPs() ([]models.DesiredLRP, error)
	DesiredLRPsByDomain(domain string) ([]models.DesiredLRP, error)
	DesiredLRPByProcessGuid(processGuid string) (models.DesiredLRP, error)
	WatchForDesiredLRPChanges(logger lager.Logger, created func(models.DesiredLRP), changed func(models.DesiredLRPChange), deleted func(models.DesiredLRP)) (stop chan<- bool, errs <-chan error)

	//actual lrp
	ActualLRPGroups() ([]models.ActualLRPGroup, error)
	ActualLRPGroupsByDomain(domain string) ([]models.ActualLRPGroup, error)
	ActualLRPGroupsByProcessGuid(string) (models.ActualLRPGroupsByIndex, error)
	ActualLRPGroupByProcessGuidAndIndex(processGuid string, index int) (models.ActualLRPGroup, error)
	RetireActualLRPs(lager.Logger, []models.ActualLRPKey)
	WatchForActualLRPChanges(logger lager.Logger, created func(models.ActualLRP, bool), changed func(models.ActualLRPChange, bool), deleted func(models.ActualLRP, bool)) (stop chan<- bool, errs <-chan error)

	// cells
	Cells() ([]models.CellPresence, error)

	// domains
	UpsertDomain(domain string, ttlInSeconds int) error
	Domains() ([]string, error)
}

//go:generate counterfeiter -o fake_bbs/fake_rep_bbs.go . RepBBS
type RepBBS interface {
	//services
	NewCellHeartbeat(cellPresence models.CellPresence, ttl, retryInterval time.Duration) ifrit.Runner

	//task
	StartTask(logger lager.Logger, taskGuid string, cellID string) (bool, error)
	TaskByGuid(taskGuid string) (models.Task, error)
	TasksByCellID(logger lager.Logger, cellID string) ([]models.Task, error)
	FailTask(logger lager.Logger, taskGuid string, failureReason string) error
	CompleteTask(logger lager.Logger, taskGuid string, cellID string, failed bool, failureReason string, result string) error

	//lrp
	ActualLRPGroupsByCellID(cellID string) ([]models.ActualLRPGroup, error)
	ClaimActualLRP(lager.Logger, models.ActualLRPKey, models.ActualLRPInstanceKey) error
	StartActualLRP(lager.Logger, models.ActualLRPKey, models.ActualLRPInstanceKey, models.ActualLRPNetInfo) error
	CrashActualLRP(lager.Logger, models.ActualLRPKey, models.ActualLRPInstanceKey, string) error
	RemoveActualLRP(lager.Logger, models.ActualLRPKey, models.ActualLRPInstanceKey) error

	// LRP evacuation
	EvacuateClaimedActualLRP(lager.Logger, models.ActualLRPKey, models.ActualLRPInstanceKey) (shared.ContainerRetainment, error)
	EvacuateRunningActualLRP(lager.Logger, models.ActualLRPKey, models.ActualLRPInstanceKey, models.ActualLRPNetInfo, uint64) (shared.ContainerRetainment, error)
	EvacuateStoppedActualLRP(lager.Logger, models.ActualLRPKey, models.ActualLRPInstanceKey) (shared.ContainerRetainment, error)
	EvacuateCrashedActualLRP(lager.Logger, models.ActualLRPKey, models.ActualLRPInstanceKey, string) (shared.ContainerRetainment, error)
	RemoveEvacuatingActualLRP(lager.Logger, models.ActualLRPKey, models.ActualLRPInstanceKey) error
}

//go:generate counterfeiter -o fake_bbs/fake_converger_bbs.go . ConvergerBBS
type ConvergerBBS interface {
	//lock
	NewConvergeLock(convergerID string, ttl, retryInterval time.Duration) ifrit.Runner

	//lrp
	ConvergeLRPs(logger lager.Logger, cellsLoader *services_bbs.CellsLoader)

	//task
	ConvergeTasks(logger lager.Logger, timeToClaim, convergenceInterval, timeToResolve time.Duration, cellsLoader *services_bbs.CellsLoader)

	//cell loader
	NewCellsLoader() *services_bbs.CellsLoader

	//cells
	WaitForCellEvent() (services_bbs.CellEvent, error)
}

//go:generate counterfeiter -o fake_bbs/fake_nsync_bbs.go . NsyncBBS
type NsyncBBS interface {
	//lock
	NewNsyncBulkerLock(bulkerID string, ttl, retryInterval time.Duration) ifrit.Runner
}

//go:generate counterfeiter -o fake_bbs/fake_auctioneer_bbs.go . AuctioneerBBS
type AuctioneerBBS interface {
	//services
	Cells() ([]models.CellPresence, error)

	// task
	FailTask(logger lager.Logger, taskGuid string, failureReason string) error

	//lock
	NewAuctioneerLock(auctioneerPresence models.AuctioneerPresence, ttl, retryInterval time.Duration) (ifrit.Runner, error)

	//lrp
	FailActualLRP(lager.Logger, models.ActualLRPKey, string) error
}

//go:generate counterfeiter -o fake_bbs/fake_metrics_bbs.go . MetricsBBS
type MetricsBBS interface {
	//task
	Tasks(logger lager.Logger) ([]models.Task, error)

	// domains
	Domains() ([]string, error)

	//lrps
	DesiredLRPs() ([]models.DesiredLRP, error)
	ActualLRPs() ([]models.ActualLRP, error)

	//lock
	NewRuntimeMetricsLock(runtimeMetricsID string, ttl, retryInterval time.Duration) ifrit.Runner
}

//go:generate counterfeiter -o fake_bbs/fake_route_emitter_bbs.go . RouteEmitterBBS
type RouteEmitterBBS interface {
	//lock
	NewRouteEmitterLock(emitterID string, ttl, retryInterval time.Duration) ifrit.Runner
}

type VeritasBBS interface {
	//task
	Tasks(logger lager.Logger) ([]models.Task, error)

	//lrp
	DesiredLRPs() ([]models.DesiredLRP, error)
	ActualLRPGroups() ([]models.ActualLRPGroup, error)
	DesireLRP(lager.Logger, models.DesiredLRP) error
	RemoveDesiredLRPByProcessGuid(logger lager.Logger, guid string) error

	// domains
	Domains() ([]string, error)
	UpsertDomain(domain string, ttlInSeconds int) error

	//services
	Cells() ([]models.CellPresence, error)
	AuctioneerAddress() (string, error)
}

func NewReceptorBBS(store storeadapter.StoreAdapter, consul *consuladapter.Adapter, clock clock.Clock, logger lager.Logger) ReceptorBBS {
	return NewBBS(store, consul, "", clock, logger)
}

func NewRepBBS(store storeadapter.StoreAdapter, consul *consuladapter.Adapter, receptorTaskHandlerURL string, clock clock.Clock, logger lager.Logger) RepBBS {
	return NewBBS(store, consul, receptorTaskHandlerURL, clock, logger)
}

func NewConvergerBBS(store storeadapter.StoreAdapter, consul *consuladapter.Adapter, receptorTaskHandlerURL string, clock clock.Clock, logger lager.Logger) ConvergerBBS {
	return NewBBS(store, consul, receptorTaskHandlerURL, clock, logger)
}

func NewNsyncBBS(consul *consuladapter.Adapter, clock clock.Clock, logger lager.Logger) NsyncBBS {
	return lock_bbs.New(consul, clock, logger.Session("lock-bbs"))
}

func NewAuctioneerBBS(store storeadapter.StoreAdapter, consul *consuladapter.Adapter, receptorTaskHandlerURL string, clock clock.Clock, logger lager.Logger) AuctioneerBBS {
	return NewBBS(store, consul, receptorTaskHandlerURL, clock, logger)
}

func NewMetricsBBS(store storeadapter.StoreAdapter, consul *consuladapter.Adapter, clock clock.Clock, logger lager.Logger) MetricsBBS {
	return NewBBS(store, consul, "", clock, logger)
}

func NewRouteEmitterBBS(consul *consuladapter.Adapter, clock clock.Clock, logger lager.Logger) RouteEmitterBBS {
	return lock_bbs.New(consul, clock, logger.Session("lock-bbs"))
}

func NewVeritasBBS(store storeadapter.StoreAdapter, clock clock.Clock, logger lager.Logger) VeritasBBS {
	return NewBBS(store, nil, "", clock, logger)
}

func NewBBS(store storeadapter.StoreAdapter, consul *consuladapter.Adapter, receptorTaskHandlerURL string, clock clock.Clock, logger lager.Logger) *BBS {
	services := services_bbs.New(consul, clock, logger.Session("services-bbs"))
	auctioneerClient := cb.NewAuctioneerClient()
	cellClient := cb.NewCellClient()

	return &BBS{
		LockBBS:     lock_bbs.New(consul, clock, logger.Session("lock-bbs")),
		LRPBBS:      lrp_bbs.New(store, clock, cellClient, auctioneerClient, services),
		ServicesBBS: services,
		TaskBBS:     task_bbs.New(store, consul, clock, cb.NewTaskClient(), auctioneerClient, cellClient, services, receptorTaskHandlerURL),
		DomainBBS:   domain_bbs.New(store, logger),
	}
}

type BBS struct {
	*lock_bbs.LockBBS
	*lrp_bbs.LRPBBS
	*services_bbs.ServicesBBS
	*task_bbs.TaskBBS
	*domain_bbs.DomainBBS
}
