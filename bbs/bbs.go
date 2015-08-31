package bbs

import (
	"time"

	"github.com/cloudfoundry-incubator/consuladapter"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/lock_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/lrp_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/services_bbs"
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
	// cells
	Cells() ([]models.CellPresence, error)
}

//go:generate counterfeiter -o fake_bbs/fake_rep_bbs.go . RepBBS
type RepBBS interface {
	//services
	NewCellPresence(cellPresence models.CellPresence, retryInterval time.Duration) ifrit.Runner
}

//go:generate counterfeiter -o fake_bbs/fake_converger_bbs.go . ConvergerBBS
type ConvergerBBS interface {
	//lock
	NewConvergeLock(convergerID string, retryInterval time.Duration) ifrit.Runner

	//cells
	CellEvents() <-chan services_bbs.CellEvent
}

const ConvergerBBSWorkPoolSize = 50

//go:generate counterfeiter -o fake_bbs/fake_nsync_bbs.go . NsyncBBS
type NsyncBBS interface {
	//lock
	NewNsyncBulkerLock(bulkerID string, retryInterval time.Duration) ifrit.Runner
}

//go:generate counterfeiter -o fake_bbs/fake_auctioneer_bbs.go . AuctioneerBBS
type AuctioneerBBS interface {
	//services
	Cells() ([]models.CellPresence, error)

	//lock
	NewAuctioneerLock(auctioneerPresence models.AuctioneerPresence, retryInterval time.Duration) (ifrit.Runner, error)
}

//go:generate counterfeiter -o fake_bbs/fake_metrics_bbs.go . MetricsBBS
type MetricsBBS interface {
	//lock
	NewRuntimeMetricsLock(runtimeMetricsID string, retryInterval time.Duration) ifrit.Runner
}

//go:generate counterfeiter -o fake_bbs/fake_route_emitter_bbs.go . RouteEmitterBBS
type RouteEmitterBBS interface {
	//lock
	NewRouteEmitterLock(emitterID string, retryInterval time.Duration) ifrit.Runner
}

//go:generate counterfeiter -o fake_bbs/fake_tps_bbs.go . TpsBBS
type TpsBBS interface {
	//lock
	NewTpsWatcherLock(watcherID string, retryInterval time.Duration) ifrit.Runner
}

type VeritasBBS interface {
	//services
	Cells() ([]models.CellPresence, error)
	AuctioneerAddress() (string, error)
}

type BBSPresenceManager interface {
	NewBBSMasterLock(bbsPresence models.BBSPresence, retryInterval time.Duration) (ifrit.Runner, error)
}

func NewReceptorBBS(store storeadapter.StoreAdapter, consul *consuladapter.Session, clock clock.Clock, logger lager.Logger) ReceptorBBS {
	return NewBBS(store, consul, clock, logger)
}

func NewRepBBS(store storeadapter.StoreAdapter, consul *consuladapter.Session, clock clock.Clock, logger lager.Logger) RepBBS {
	return NewBBS(store, consul, clock, logger)
}

func NewConvergerBBS(store storeadapter.StoreAdapter, consul *consuladapter.Session, clock clock.Clock, logger lager.Logger) ConvergerBBS {
	return NewBBS(store, consul, clock, logger)
}

func NewNsyncBBS(consul *consuladapter.Session, clock clock.Clock, logger lager.Logger) NsyncBBS {
	return lock_bbs.New(consul, clock, logger.Session("lock-bbs"))
}

func NewAuctioneerBBS(store storeadapter.StoreAdapter, consul *consuladapter.Session, clock clock.Clock, logger lager.Logger) AuctioneerBBS {
	return NewBBS(store, consul, clock, logger)
}

func NewMetricsBBS(consul *consuladapter.Session, clock clock.Clock, logger lager.Logger) MetricsBBS {
	return lock_bbs.New(consul, clock, logger.Session("metrics-bbs"))
}

func NewRouteEmitterBBS(consul *consuladapter.Session, clock clock.Clock, logger lager.Logger) RouteEmitterBBS {
	return lock_bbs.New(consul, clock, logger.Session("lock-bbs"))
}

func NewTpsBBS(consul *consuladapter.Session, clock clock.Clock, logger lager.Logger) TpsBBS {
	return lock_bbs.New(consul, clock, logger.Session("lock-bbs"))
}

func NewVeritasBBS(store storeadapter.StoreAdapter, consul *consuladapter.Session, clock clock.Clock, logger lager.Logger) VeritasBBS {
	return NewBBS(store, consul, clock, logger)
}

func NewBBSPresenceManager(consul *consuladapter.Session, clock clock.Clock, logger lager.Logger) BBSPresenceManager {
	return lock_bbs.New(consul, clock, logger.Session("lock-bbs"))
}

func NewBBS(store storeadapter.StoreAdapter, consul *consuladapter.Session, clock clock.Clock, logger lager.Logger) *BBS {
	services := services_bbs.New(consul, clock, logger.Session("services-bbs"))
	auctioneerClient := cb.NewAuctioneerClient()
	cellClient := cb.NewCellClient()

	return &BBS{
		LockBBS:     lock_bbs.New(consul, clock, logger.Session("lock-bbs")),
		LRPBBS:      lrp_bbs.New(store, clock, cellClient, auctioneerClient, services),
		ServicesBBS: services,
	}
}

type BBS struct {
	*lock_bbs.LockBBS
	*lrp_bbs.LRPBBS
	*services_bbs.ServicesBBS
}
