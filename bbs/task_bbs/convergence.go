package task_bbs

import (
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/metric"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/workpool"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
)

const workerPoolSize = 20

const (
	convergeTaskRunsCounter = metric.Counter("ConvergenceTaskRuns")
	convergeTaskDuration    = metric.Duration("ConvergenceTaskDuration")

	tasksKickedCounter = metric.Counter("ConvergenceTasksKicked")
	tasksPrunedCounter = metric.Counter("ConvergenceTasksPruned")
)

type compareAndSwappableTask struct {
	OldIndex uint64
	NewTask  models.Task
}

// ConvergeTask is run by *one* cell every X seconds (doesn't really matter what X is.. pick something performant)
// Converge will:
// 1. Kick (by setting) any tasks that are still pending (and have been for > convergence interval)
// 2. Kick any tasks that are completed (and have been for > convergence interval)
// 3. Delete any tasks that are completed (and have been for > timeToResolve interval)
// 5. Demote to completed any resolving tasks that have been resolving for > 30s
// 6. Mark as failed any tasks that have been in the pending state for > expirePendingTaskDuration
// 7. Mark as failed any running tasks whose cell has stopped maintaining presence
func (bbs *TaskBBS) ConvergeTasks(logger lager.Logger, expirePendingTaskDuration, convergenceInterval, timeToResolve time.Duration) {
	convergeTaskRunsCounter.Increment()

	convergeStart := bbs.timeProvider.Now()

	// make sure to get funcy here otherwise the time will be precomputed
	defer func() {
		convergeTaskDuration.Send(time.Since(convergeStart))
	}()

	taskLog := bbs.logger.Session("converge-tasks")

	taskState, err := bbs.store.ListRecursively(shared.TaskSchemaRoot)
	if err != nil {
		return
	}

	cellState, err := bbs.store.ListRecursively(shared.CellSchemaRoot)
	if err == storeadapter.ErrorKeyNotFound {
		cellState = storeadapter.StoreNode{}
	} else if err != nil {
		return
	}

	logError := func(task models.Task, message string) {
		taskLog.Error(message, nil, lager.Data{
			"task": task,
		})
	}

	tasksToComplete := []models.Task{}
	scheduleForCompletion := func(task models.Task) {
		if task.CompletionCallbackURL == nil {
			return
		}
		tasksToComplete = append(tasksToComplete, task)
	}

	keysToDelete := []string{}

	tasksToCAS := []compareAndSwappableTask{}
	scheduleForCASByIndex := func(index uint64, newTask models.Task) {
		tasksToCAS = append(tasksToCAS, compareAndSwappableTask{
			OldIndex: index,
			NewTask:  newTask,
		})
	}

	tasksToAuction := []models.Task{}

	var tasksKicked uint64 = 0

	for _, node := range taskState.ChildNodes {
		var task models.Task
		err = models.FromJSON(node.Value, &task)
		if err != nil {
			taskLog.Error("failed-to-unmarshal-task-json", err, lager.Data{
				"key":   node.Key,
				"value": node.Value,
			})

			keysToDelete = append(keysToDelete, node.Key)
			continue
		}

		shouldKickTask := bbs.durationSinceTaskUpdated(task) >= convergenceInterval

		switch task.State {
		case models.TaskStatePending:
			shouldMarkAsFailed := bbs.durationSinceTaskCreated(task) >= expirePendingTaskDuration
			if shouldMarkAsFailed {
				logError(task, "failed-to-start-in-time")
				scheduleForCASByIndex(node.Index, bbs.markTaskFailed(task, "not started within time limit"))
				tasksKicked++
			} else if shouldKickTask {
				taskLog.Info("requesting-auction-for-pending-task", lager.Data{"task": task})
				tasksToAuction = append(tasksToAuction, task)
				tasksKicked++
			}
		case models.TaskStateRunning:
			_, cellIsAlive := cellState.Lookup(task.CellID)
			if !cellIsAlive {
				logError(task, "cell-disappeared")
				scheduleForCASByIndex(node.Index, bbs.markTaskFailed(task, "cell disappeared before completion"))
				tasksKicked++
			}
		case models.TaskStateCompleted:
			shouldDeleteTask := bbs.durationSinceTaskFirstCompleted(task) >= timeToResolve
			if shouldDeleteTask {
				logError(task, "failed-to-start-resolving-in-time")
				keysToDelete = append(keysToDelete, node.Key)
			} else if shouldKickTask {
				taskLog.Info("kicking-completed-task", lager.Data{"task": task})
				scheduleForCompletion(task)
				tasksKicked++
			}
		case models.TaskStateResolving:
			shouldDeleteTask := bbs.durationSinceTaskFirstCompleted(task) >= timeToResolve
			if shouldDeleteTask {
				logError(task, "failed-to-resolve-in-time")
				keysToDelete = append(keysToDelete, node.Key)
			} else if shouldKickTask {
				taskLog.Info("demoting-resolving-to-completed", lager.Data{"task": task})
				demoted := demoteToCompleted(task)
				scheduleForCASByIndex(node.Index, demoted)
				scheduleForCompletion(demoted)
				tasksKicked++
			}
		}
	}

	if len(tasksToAuction) > 0 {
		if err := bbs.requestTaskAuctions(tasksToAuction); err != nil {
			taskLog.Error("failed-to-request-auctions-for-pending-tasks", err,
				lager.Data{"tasks": tasksToAuction})
		}
	}

	workPool := workpool.NewWorkPool(workerPoolSize)
	tasksKickedCounter.Add(tasksKicked)
	bbs.batchCompareAndSwapTasks(tasksToCAS, workPool, taskLog)
	workPool.Stop()

	bbs.completeTasks(tasksToComplete, taskLog)

	tasksPrunedCounter.Add(uint64(len(keysToDelete)))
	bbs.store.Delete(keysToDelete...)

}

func (bbs *TaskBBS) durationSinceTaskCreated(task models.Task) time.Duration {
	return bbs.timeProvider.Now().Sub(time.Unix(0, task.CreatedAt))
}

func (bbs *TaskBBS) durationSinceTaskUpdated(task models.Task) time.Duration {
	return bbs.timeProvider.Now().Sub(time.Unix(0, task.UpdatedAt))
}

func (bbs *TaskBBS) durationSinceTaskFirstCompleted(task models.Task) time.Duration {
	if task.FirstCompletedAt == 0 {
		return 0
	}
	return bbs.timeProvider.Now().Sub(time.Unix(0, task.FirstCompletedAt))
}

func (bbs *TaskBBS) markTaskFailed(task models.Task, reason string) models.Task {
	return bbs.markTaskCompleted(task, true, reason, "")
}

func (bbs *TaskBBS) markTaskCompleted(task models.Task, failed bool, failureReason string, result string) models.Task {
	task.UpdatedAt = bbs.timeProvider.Now().UnixNano()
	task.FirstCompletedAt = bbs.timeProvider.Now().UnixNano()
	task.State = models.TaskStateCompleted
	task.Failed = failed
	task.FailureReason = failureReason
	task.Result = result
	return task
}

func demoteToCompleted(task models.Task) models.Task {
	task.State = models.TaskStateCompleted
	return task
}

func (bbs *TaskBBS) batchCompareAndSwapTasks(tasksToCAS []compareAndSwappableTask, pool *workpool.WorkPool, taskLog lager.Logger) {
	if len(tasksToCAS) == 0 {
		return
	}

	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(len(tasksToCAS))

	for _, taskToCAS := range tasksToCAS {
		task := taskToCAS.NewTask
		task.UpdatedAt = bbs.timeProvider.Now().UnixNano()
		value, err := models.ToJSON(task)
		if err != nil {
			taskLog.Error("failed-to-marshal", err, lager.Data{
				"task": task,
			})
			continue
		}

		newStoreNode := storeadapter.StoreNode{
			Key:   shared.TaskSchemaPath(task.TaskGuid),
			Value: value,
		}
		index := taskToCAS.OldIndex
		pool.Submit(func() {
			defer waitGroup.Done()
			err := bbs.store.CompareAndSwapByIndex(index, newStoreNode)
			if err != nil {
				taskLog.Error("failed-to-compare-and-swap", err, lager.Data{
					"task": task,
				})
			}
		})
	}

	waitGroup.Wait()
}

func (bbs *TaskBBS) completeTasks(tasksToComplete []models.Task, taskLog lager.Logger) {
	if len(tasksToComplete) == 0 {
		return
	}

	receptor, err := bbs.services.Receptor()
	if err != nil {
		taskLog.Error("failed-to-find-receptor", err)
		return
	}

	err = bbs.taskClient.CompleteTasks(receptor.ReceptorURL, tasksToComplete)
	if err != nil {
		taskLog.Error("failed-to-complete-tasks", err, lager.Data{
			"tasks": tasksToComplete,
		})
	}
}
