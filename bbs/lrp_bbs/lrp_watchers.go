package lrp_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/pivotal-golang/lager"
)

func (bbs *LRPBBS) WatchForDesiredLRPChanges(logger lager.Logger) (<-chan models.DesiredLRP, <-chan models.DesiredLRPChange, <-chan models.DesiredLRP, <-chan error) {
	logger = logger.Session("watching-for-desired-lrp-changes")

	creates := make(chan models.DesiredLRP)
	updates := make(chan models.DesiredLRPChange)
	deletes := make(chan models.DesiredLRP)

	events, _, err := bbs.store.Watch(shared.DesiredLRPSchemaRoot)

	go func() {
		for event := range events {
			switch {
			case event.Node != nil && event.PrevNode == nil:
				logger.Debug("received-create")

				var desiredLRP models.DesiredLRP
				err := models.FromJSON(event.Node.Value, &desiredLRP)
				if err != nil {
					logger.Error("failed-to-unmarshal-desired-lrp", err, lager.Data{"value": event.Node.Value})
					continue
				}

				logger.Debug("sending-create", lager.Data{"desired-lrp": desiredLRP})
				creates <- desiredLRP

			case event.Node != nil && event.PrevNode != nil: // update
				logger.Debug("received-update")

				var before models.DesiredLRP
				err := models.FromJSON(event.PrevNode.Value, &before)
				if err != nil {
					logger.Error("failed-to-unmarshal-desired-lrp", err, lager.Data{"value": event.PrevNode.Value})
					continue
				}

				var after models.DesiredLRP
				err = models.FromJSON(event.Node.Value, &after)
				if err != nil {
					logger.Error("failed-to-unmarshal-desired-lrp", err, lager.Data{"value": event.Node.Value})
					continue
				}

				logger.Debug("sending-update", lager.Data{"before": before, "after": after})
				updates <- models.DesiredLRPChange{Before: before, After: after}

			case event.Node == nil && event.PrevNode != nil: // delete
				logger.Debug("received-delete")

				var desiredLRP models.DesiredLRP
				err := models.FromJSON(event.PrevNode.Value, &desiredLRP)
				if err != nil {
					logger.Error("failed-to-unmarshal-desired-lrp", err, lager.Data{"value": event.PrevNode.Value})
					continue
				}

				logger.Debug("sending-delete", lager.Data{"desired-lrp": desiredLRP})
				deletes <- desiredLRP

			default:
				logger.Debug("received-event-with-both-nodes-nil")
			}
		}
	}()

	return creates, updates, deletes, err
}

func (bbs *LRPBBS) WatchForActualLRPChanges(logger lager.Logger) (<-chan models.ActualLRP, <-chan models.ActualLRPChange, <-chan models.ActualLRP, <-chan error) {
	logger = logger.Session("watching-for-actual-lrp-changes")

	creates := make(chan models.ActualLRP)
	updates := make(chan models.ActualLRPChange)
	deletes := make(chan models.ActualLRP)

	events, _, err := bbs.store.Watch(shared.ActualLRPSchemaRoot)

	go func() {
		for event := range events {
			switch {
			case event.Node != nil && event.PrevNode == nil:
				logger.Debug("received-create")

				var actualLRP models.ActualLRP
				err := models.FromJSON(event.Node.Value, &actualLRP)
				if err != nil {
					logger.Error("failed-to-unmarshal-actual-lrp", err, lager.Data{"value": event.Node.Value})
					continue
				}

				logger.Debug("sending-create", lager.Data{"actual-lrp": actualLRP})
				creates <- actualLRP

			case event.Node != nil && event.PrevNode != nil:
				logger.Debug("received-update")

				var before models.ActualLRP
				err := models.FromJSON(event.PrevNode.Value, &before)
				if err != nil {
					logger.Error("failed-to-unmarshal-actual-lrp", err, lager.Data{"value": event.PrevNode.Value})
					continue
				}

				var after models.ActualLRP
				err = models.FromJSON(event.Node.Value, &after)
				if err != nil {
					logger.Error("failed-to-unmarshal-actual-lrp", err, lager.Data{"value": event.Node.Value})
					continue
				}

				logger.Debug("sending-update", lager.Data{"before": before, "after": after})
				updates <- models.ActualLRPChange{Before: before, After: after}

			case event.PrevNode != nil && event.Node == nil:
				logger.Debug("received-delete")

				var actualLRP models.ActualLRP
				err := models.FromJSON(event.PrevNode.Value, &actualLRP)
				if err != nil {
					logger.Error("failed-to-unmarshal-actual-lrp", err, lager.Data{"value": event.PrevNode.Value})
				} else {
					logger.Debug("sending-delete", lager.Data{"actual-lrp": actualLRP})
					deletes <- actualLRP
				}

			default:
				logger.Debug("received-event-with-both-nodes-nil")
			}
		}
	}()

	return creates, updates, deletes, err
}
