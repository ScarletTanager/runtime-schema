package lrp_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/metric"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/pivotal-golang/lager"
)

var (
	lrpStartInstanceCounter = metric.Counter("LRPInstanceStartRequests")
	lrpStopInstanceCounter  = metric.Counter("LRPInstanceStopRequests")
)

type reconcileInfo struct {
	desiredLRP models.DesiredLRP
	actualLRPs models.ActualLRPsByIndex
	Result
}

func (bbs *LRPBBS) processDesiredChange(desiredChange models.DesiredLRPChange, logger lager.Logger) {
	var desiredLRP models.DesiredLRP

	changeLogger := logger.Session("desired-lrp-change", lager.Data{
		"desired-lrp": desiredLRP,
	})

	if desiredChange.After == nil {
		desiredLRP = *desiredChange.Before
		desiredLRP.Instances = 0
	} else {
		desiredLRP = *desiredChange.After
	}

	actuals, err := bbs.ActualLRPsByProcessGuid(desiredLRP.ProcessGuid)
	if err != nil {
		changeLogger.Error("fetch-actuals-failed", err, lager.Data{"desired-app-message": desiredLRP})
		return
	}

	bbs.reconcile([]reconcileInfo{{desiredLRP, actuals, Reconcile(desiredLRP.Instances, actuals)}}, logger)
}

func (bbs *LRPBBS) reconcile(infos []reconcileInfo, logger lager.Logger) {
	startAuctions := []models.LRPStart{}
	lrpsToRetire := []models.ActualLRP{}

	for _, delta := range infos {
		for _, lrpIndex := range delta.IndicesToStart {
			logger.Info("request-start", lager.Data{
				"process-guid": delta.desiredLRP.ProcessGuid,
				"index":        lrpIndex,
			})

			lrpStartInstanceCounter.Increment()

			err := bbs.createActualLRP(delta.desiredLRP, lrpIndex, logger)
			if err != nil {
				logger.Error("failed-to-create-actual-lrp", err, lager.Data{
					"process-guid": delta.desiredLRP.ProcessGuid,
					"index":        lrpIndex,
				})
				continue
			}

			startAuctions = append(startAuctions, models.LRPStart{
				DesiredLRP: delta.desiredLRP,
				Index:      lrpIndex,
			})
		}

		for _, index := range delta.IndicesToStop {
			actualLRP := delta.actualLRPs[index]
			logger.Info("request-stop", lager.Data{
				"process-guid":  actualLRP.ProcessGuid,
				"instance-guid": actualLRP.InstanceGuid,
				"index":         index,
			})

			lrpsToRetire = append(lrpsToRetire, actualLRP)
		}
	}

	if len(startAuctions) > 0 {
		err := bbs.requestLRPStartAuctions(startAuctions)
		if err != nil {
			logger.Error("failed-to-request-start-auctions", err, lager.Data{"lrp-starts": startAuctions})
			// The creation succeeded, the start request error can be dropped
		}
	}

	if len(lrpsToRetire) > 0 {
		lrpStopInstanceCounter.Add(uint64(len(lrpsToRetire)))
		bbs.RetireActualLRPs(lrpsToRetire, logger)
	}
}
