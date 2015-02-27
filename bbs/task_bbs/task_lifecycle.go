package task_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
)

// The stager calls this when it wants to desire a payload
// stagerTaskBBS will retry this repeatedly if it gets a StoreTimeout error (up to N seconds?)
// If this fails, the stager should bail and run its "this-failed-to-stage" routine
func (bbs *TaskBBS) DesireTask(logger lager.Logger, task models.Task) error {
	taskLogger := logger.WithData(lager.Data{"task-guid": task.TaskGuid})

	taskLogger.Info("desiring-task")
	defer taskLogger.Info("finished-desiring-task")

	err := task.Validate()
	if err != nil {
		return err
	}
	task.State = models.TaskStatePending

	if task.CreatedAt == 0 {
		task.CreatedAt = bbs.clock.Now().UnixNano()
	}

	task.UpdatedAt = bbs.clock.Now().UnixNano()

	value, err := models.ToJSON(task)
	if err != nil {
		return err
	}

	taskLogger.Debug("persisting-task")
	err = bbs.store.Create(storeadapter.StoreNode{
		Key:   shared.TaskSchemaPath(task.TaskGuid),
		Value: value,
	})
	if err != nil {
		taskLogger.Error("failed-persisting-task", err)
		return shared.ConvertStoreError(err)
	}
	taskLogger.Debug("succeeded-persisting-task")

	taskLogger.Debug("requesting-task-auction")
	err = bbs.requestTaskAuctions(taskLogger, []models.Task{task})
	if err != nil {
		taskLogger.Error("failed-requesting-task-auction", err)
		// The creation succeeded, the auction request error can be dropped
	} else {
		taskLogger.Debug("succeeded-requesting-task-auction")
	}

	return nil
}

// The cell calls this when it is about to run the task in the allocated container
// stagerTaskBBS will retry this repeatedly if it gets a StoreTimeout error (up to N seconds?)
// If this fails, the cell should assume that someone else will run it and should clean up and bail
func (bbs *TaskBBS) StartTask(logger lager.Logger, taskGuid string, cellID string) (bool, error) {
	logger = logger.Session("start-task")

	logger.Info("starting-task")
	defer logger.Info("finished-starting-task")

	task, index, err := bbs.getTask(taskGuid)

	if err != nil {
		return false, err
	}

	if task.State == models.TaskStateRunning && task.CellID == cellID {
		logger.Info("task-already-running", lager.Data{"task-guid": taskGuid})
		return false, nil
	}

	err = validateStateTransition(task.State, models.TaskStateRunning)
	if err != nil {
		logger.Info("invalid-state-transition-to-running", lager.Data{"task-guid": taskGuid, "existing-state": task.State})
		return false, err
	}

	task.UpdatedAt = bbs.clock.Now().UnixNano()
	task.State = models.TaskStateRunning
	task.CellID = cellID

	value, err := models.ToJSON(task)
	if err != nil {
		logger.Info("error-converting-to-json")
		return false, err
	}

	err = bbs.store.CompareAndSwapByIndex(index, storeadapter.StoreNode{
		Key:   shared.TaskSchemaPath(taskGuid),
		Value: value,
	})
	if err != nil {
		logger.Error("failed-persisting-task", err)
		return false, shared.ConvertStoreError(err)
	}

	return true, nil
}

// The cell calls this when the user requested to cancel the task
// stagerTaskBBS will retry this repeatedly if it gets a StoreTimeout error (up to N seconds?)
// Will fail if the task has already been cancelled or completed normally
func (bbs *TaskBBS) CancelTask(logger lager.Logger, taskGuid string) error {
	logger = logger.Session("cancel-task")
	logger.Info("canceling-task")
	defer logger.Info("finished-canceling-task")

	task, index, err := bbs.getTask(taskGuid)
	if err != nil {
		logger.Info("failed-to-get-task", lager.Data{"task-guid": taskGuid})
		return err
	}

	if task.State == models.TaskStateResolving || task.State == models.TaskStateCompleted {
		logger.Info("invalid-state-transition-to-completed", lager.Data{"task-guid": taskGuid, "existing-state": task.State})
		return bbserrors.NewTaskStateTransitionError(task.State, models.TaskStateCompleted)
	}

	err = bbs.completeTask(logger, task, index, true, "task was cancelled", "")
	if err != nil {
		return err
	}

	if task.CellID == "" {
		return nil
	}

	cellPresence, err := bbs.services.CellById(task.CellID)
	if err != nil {
		logger.Error("failed-to-cancel-immediately", err)
		return nil
	}

	err = bbs.cellClient.CancelTask(cellPresence.RepAddress, task.TaskGuid)
	if err != nil {
		logger.Error("failed-to-cancel-immediately", err)
		return nil
	}

	return nil
}

func (bbs *TaskBBS) FailTask(logger lager.Logger, taskGuid string, failureReason string) error {
	logger = logger.Session("fail-task")
	logger.Info("failing-task")
	defer logger.Info("finished-failing-task")

	task, index, err := bbs.getTask(taskGuid)
	if err != nil {
		logger.Info("failed-to-get-task", lager.Data{"task-guid": taskGuid})
		return err
	}

	if task.State == models.TaskStateResolving || task.State == models.TaskStateCompleted {
		logger.Info("invalid-state-transition-to-completed", lager.Data{"task-guid": taskGuid, "existing-state": task.State})
		return bbserrors.NewTaskStateTransitionError(task.State, models.TaskStateCompleted)
	}

	return bbs.completeTask(logger, task, index, true, failureReason, "")
}

// The cell calls this when it has finished running the task (be it success or failure)
// stagerTaskBBS will retry this repeatedly if it gets a StoreTimeout error (up to N seconds?)
// This really really shouldn't fail.  If it does, blog about it and walk away. If it failed in a
// consistent way (i.e. key already exists), there's probably a flaw in our design.
func (bbs *TaskBBS) CompleteTask(logger lager.Logger, taskGuid string, cellID string, failed bool, failureReason string, result string) error {
	logger = logger.Session("complete-task")
	logger.Info("completing-task")
	defer logger.Info("finished-completing-task")

	task, index, err := bbs.getTask(taskGuid)
	if err != nil {
		logger.Info("failed-to-get-task", lager.Data{"task-guid": taskGuid})
		return err
	}

	if task.State == models.TaskStateRunning && task.CellID != cellID {
		logger.Info("invalid-cell-id", lager.Data{"task-guid": taskGuid, "task-cell-id": task.CellID, "cell-id": cellID})
		return bbserrors.ErrTaskRunningOnDifferentCell
	}

	err = validateStateTransition(task.State, models.TaskStateCompleted)
	if err != nil {
		logger.Info("invalid-state-transition-to-completed", lager.Data{"task-guid": taskGuid, "existing-state": task.State})
		return err
	}

	return bbs.completeTask(logger, task, index, failed, failureReason, result)
}

func (bbs *TaskBBS) completeTask(logger lager.Logger, task models.Task, index uint64, failed bool, failureReason string, result string) error {
	task = bbs.markTaskCompleted(task, failed, failureReason, result)

	value, err := models.ToJSON(task)
	if err != nil {
		return err
	}

	err = bbs.store.CompareAndSwapByIndex(index, storeadapter.StoreNode{
		Key:   shared.TaskSchemaPath(task.TaskGuid),
		Value: value,
	})
	if err != nil {
		logger.Error("failed-persisting-task", err)
		return shared.ConvertStoreError(err)
	}

	if task.CompletionCallbackURL == nil {
		return nil
	}

	receptorPresence, err := bbs.services.Receptor()
	if err != nil {
		logger.Error("could-not-fetch-receptors", err)
		return nil
	}

	err = bbs.taskClient.CompleteTasks(receptorPresence.ReceptorURL, []models.Task{task})
	if err != nil {
		logger.Error("failed-to-complete-task", err)
		return nil
	}

	return nil
}

// The stager calls this when it wants to claim a completed task.  This ensures that only one
// stager ever attempts to handle a completed task
func (bbs *TaskBBS) ResolvingTask(logger lager.Logger, taskGuid string) error {
	logger = logger.Session("resolving-task")
	logger.Info("resolving-task")
	defer logger.Info("finished-resolving-task")

	task, index, err := bbs.getTask(taskGuid)
	if err != nil {
		logger.Info("failed-to-get-task", lager.Data{"task-guid": taskGuid})
		return err
	}

	err = validateStateTransition(task.State, models.TaskStateResolving)
	if err != nil {
		logger.Info("invalid-state-transition-to-resolving", lager.Data{"task-guid": taskGuid, "existing-state": task.State})
		return err
	}

	task.UpdatedAt = bbs.clock.Now().UnixNano()
	task.State = models.TaskStateResolving

	value, err := models.ToJSON(task)
	if err != nil {
		return err
	}

	return shared.ConvertStoreError(bbs.store.CompareAndSwapByIndex(index, storeadapter.StoreNode{
		Key:   shared.TaskSchemaPath(taskGuid),
		Value: value,
	}))
}

// The stager calls this when it wants to signal that it has received a completion and is handling it
// stagerTaskBBS will retry this repeatedly if it gets a StoreTimeout error (up to N seconds?)
// If this fails, the stager should assume that someone else is handling the completion and should bail
func (bbs *TaskBBS) ResolveTask(logger lager.Logger, taskGuid string) error {
	logger = logger.Session("resolve-task")
	logger.Info("resolve-task")
	defer logger.Info("finished-resolve-task")

	task, _, err := bbs.getTask(taskGuid)

	if err != nil {
		return err
	}

	err = validateCanDelete(task.State)
	if err != nil {
		logger.Info("invalid-state-transition-to-delete", lager.Data{"task-guid": taskGuid, "existing-state": task.State})
		return err
	}

	return shared.ConvertStoreError(bbs.store.Delete(shared.TaskSchemaPath(taskGuid)))
}

func validateStateTransition(from, to models.TaskState) error {
	if (from == models.TaskStatePending && to == models.TaskStateRunning) ||
		(from == models.TaskStateRunning && to == models.TaskStateCompleted) ||
		(from == models.TaskStateCompleted && to == models.TaskStateResolving) {
		return nil
	} else {
		return bbserrors.NewTaskStateTransitionError(from, to)
	}
}

func validateCanDelete(from models.TaskState) error {
	if from != models.TaskStateResolving {
		return bbserrors.NewTaskCannotBeResolvedError(from)
	} else {
		return nil
	}
}

func (bbs *TaskBBS) requestTaskAuctions(logger lager.Logger, tasks []models.Task) error {
	auctioneerAddress, err := bbs.services.AuctioneerAddress()
	if err != nil {
		return err
	}
	logger.Debug("did-fetch-auctioneer-address")

	err = bbs.auctioneerClient.RequestTaskAuctions(auctioneerAddress, tasks)
	if err != nil {
		return err
	}
	logger.Debug("did-request-task-auctions")

	return nil
}
