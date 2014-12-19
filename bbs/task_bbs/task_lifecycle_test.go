package task_bbs_test

import (
	"errors"
	"net/url"
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	. "github.com/cloudfoundry-incubator/runtime-schema/bbs/task_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter/fakestoreadapter"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager/lagertest"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"
)

var _ = Describe("Task BBS", func() {
	var task models.Task

	Describe("DesireTask", func() {
		var errDesire error

		JustBeforeEach(func() {
			errDesire = bbs.DesireTask(task)
		})

		Context("when given a valid task", func() {
			Context("when a task is not already present at the desired key", func() {
				Context("when given a task with a CreatedAt time", func() {
					var taskGuid string
					var domain string
					var stack string
					var createdAtTime int64

					BeforeEach(func() {
						taskGuid = "some-guid"
						domain = "tests"
						stack = "pancakes"
						createdAtTime = 1234812

						task = models.Task{
							Domain:    domain,
							TaskGuid:  taskGuid,
							Stack:     stack,
							Action:    dummyAction,
							CreatedAt: createdAtTime,
						}
					})

					It("does not error", func() {
						Ω(errDesire).ShouldNot(HaveOccurred())
					})

					It("persists the task", func() {
						persistedTask, err := bbs.TaskByGuid(taskGuid)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(persistedTask.Domain).Should(Equal(domain))
						Ω(persistedTask.Stack).Should(Equal(stack))
						Ω(persistedTask.Action).Should(Equal(dummyAction))
					})

					It("honours the CreatedAt time", func() {
						persistedTask, err := bbs.TaskByGuid(taskGuid)
						Ω(err).ShouldNot(HaveOccurred())
						Ω(persistedTask.CreatedAt).Should(Equal(createdAtTime))
					})

					It("sets the UpdatedAt time", func() {
						persistedTask, err := bbs.TaskByGuid(taskGuid)
						Ω(err).ShouldNot(HaveOccurred())
						Ω(persistedTask.UpdatedAt).Should(Equal(timeProvider.Now().UnixNano()))
					})

					Context("when able to fetch the Auctioneer address", func() {
						var auctioneerPresence models.AuctioneerPresence

						BeforeEach(func() {
							auctioneerPresence = models.AuctioneerPresence{
								AuctioneerID:      "the-auctioneer-id",
								AuctioneerAddress: "the-address",
							}

							registerAuctioneer(auctioneerPresence)
						})

						It("requests an auction", func() {
							Ω(fakeAuctioneerClient.RequestTaskAuctionCallCount()).Should(Equal(1))

							requestAddress, requestedTask := fakeAuctioneerClient.RequestTaskAuctionArgsForCall(0)
							Ω(requestAddress).Should(Equal(auctioneerPresence.AuctioneerAddress))
							Ω(requestedTask.TaskGuid).Should(Equal(taskGuid))
						})

						Context("when requesting a task auction succeeds", func() {
							BeforeEach(func() {
								fakeAuctioneerClient.RequestTaskAuctionReturns(nil)
							})

							It("does not return an error", func() {
								Ω(errDesire).ShouldNot(HaveOccurred())
							})
						})

						Context("when requesting a task auction fails", func() {
							BeforeEach(func() {
								fakeAuctioneerClient.RequestLRPStartAuctionReturns(errors.New("oops"))
							})

							It("does not return an error", func() {
								// The creation succeeded, we can ignore the auction request error (converger will eventually do it)
								Ω(errDesire).ShouldNot(HaveOccurred())
							})
						})
					})

					Context("when unable to fetch the Auctioneer address", func() {
						It("does not request an auction", func() {
							Consistently(fakeAuctioneerClient.RequestTaskAuctionCallCount).Should(BeZero())
						})

						It("does not return an error", func() {
							// The creation succeeded, we can ignore the auction request error (converger will eventually do it)
							Ω(errDesire).ShouldNot(HaveOccurred())
						})
					})
				})

				Context("when given a task without a CreatedAt time", func() {
					var taskGuid string
					var domain string
					var stack string

					BeforeEach(func() {
						taskGuid = "some-guid"
						domain = "tests"
						stack = "pancakes"
						task = models.Task{
							Domain:   domain,
							TaskGuid: taskGuid,
							Stack:    stack,
							Action:   dummyAction,
						}
					})

					It("does not error", func() {
						Ω(errDesire).ShouldNot(HaveOccurred())
					})

					It("persists the task", func() {
						persistedTask, err := bbs.TaskByGuid(taskGuid)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(persistedTask.Domain).Should(Equal(domain))
						Ω(persistedTask.Stack).Should(Equal(stack))
						Ω(persistedTask.Action).Should(Equal(dummyAction))
					})

					It("provides a CreatedAt time", func() {
						persistedTask, err := bbs.TaskByGuid(taskGuid)
						Ω(err).ShouldNot(HaveOccurred())
						Ω(persistedTask.CreatedAt).Should(Equal(timeProvider.Now().UnixNano()))
					})

					It("sets the UpdatedAt time", func() {
						persistedTask, err := bbs.TaskByGuid(taskGuid)
						Ω(err).ShouldNot(HaveOccurred())
						Ω(persistedTask.UpdatedAt).Should(Equal(timeProvider.Now().UnixNano()))
					})
				})
			})

			Context("when a task is already present at the desired key", func() {
				BeforeEach(func() {
					task = models.Task{
						Domain:   "tests",
						TaskGuid: "some-guid",
						Stack:    "pancakes",
						Action:   dummyAction,
					}

					err := bbs.DesireTask(task)
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("does not persist another", func() {
					Consistently(bbs.Tasks).Should(HaveLen(1))
				})

				It("does not request an auction", func() {
					Consistently(fakeAuctioneerClient.RequestTaskAuctionCallCount).Should(BeZero())
				})

				It("returns an error", func() {
					Ω(errDesire).Should(Equal(bbserrors.ErrStoreResourceExists))
				})
			})
		})

		Context("when given an invalid task", func() {
			BeforeEach(func() {
				task = models.Task{
					TaskGuid: "some-guid",
					Stack:    "pancakes",
					Action:   dummyAction,
					// missing Domain
				}
			})

			It("does not persist a task", func() {
				Consistently(bbs.Tasks).Should(BeEmpty())
			})

			It("does not request an auction", func() {
				Consistently(fakeAuctioneerClient.RequestTaskAuctionCallCount).Should(BeZero())
			})

			It("returns an error", func() {
				Ω(errDesire).Should(ContainElement(models.ErrInvalidField{"domain"}))
			})
		})

		Context("when the store is out of commission", func() {
			var anotherTask models.Task

			BeforeEach(func() {
				anotherTask = models.Task{
					Domain:   "tests",
					TaskGuid: "another-guid",
					Stack:    "pancakes",
					Action:   dummyAction,
				}
			})

			itRetriesUntilStoreComesBack(func() error {
				return bbs.DesireTask(anotherTask)
			})
		})
	})

	Describe("StartTask", func() {
		BeforeEach(func() {
			task = models.Task{
				TaskGuid:  "some-guid",
				Domain:    "tests",
				Stack:     "pancakes",
				Action:    dummyAction,
				CreatedAt: 1234812,
			}
		})

		Context("when starting a pending Task", func() {
			BeforeEach(func() {
				err := bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("sets the state to running", func() {
				err := bbs.StartTask(task.TaskGuid, "cell-ID")
				Ω(err).ShouldNot(HaveOccurred())

				tasks, err := bbs.RunningTasks()
				Ω(err).ShouldNot(HaveOccurred())

				Ω(tasks[0].TaskGuid).Should(Equal(task.TaskGuid))
				Ω(tasks[0].State).Should(Equal(models.TaskStateRunning))
			})

			It("should bump UpdatedAt", func() {
				timeProvider.IncrementBySeconds(1)

				err := bbs.StartTask(task.TaskGuid, "cell-ID")
				Ω(err).ShouldNot(HaveOccurred())

				tasks, err := bbs.RunningTasks()
				Ω(err).ShouldNot(HaveOccurred())

				Ω(tasks[0].UpdatedAt).Should(Equal(timeProvider.Now().UnixNano()))
			})

			Context("when the store is out of commission", func() {
				itRetriesUntilStoreComesBack(func() error {
					return bbs.StartTask(task.TaskGuid, "cell-ID")
				})
			})
		})

		Context("When starting a Task that is already started", func() {
			BeforeEach(func() {
				err := bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.StartTask(task.TaskGuid, "cell-ID")
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("returns an error", func() {
				err := bbs.StartTask(task.TaskGuid, "cell-ID")
				Ω(err).Should(HaveOccurred())
			})
		})
	})

	Describe("CancelTask", func() {
		BeforeEach(func() {
			task = models.Task{
				TaskGuid:  "some-guid",
				Domain:    "tests",
				Stack:     "pancakes",
				Action:    dummyAction,
				CreatedAt: 1234812,
			}
		})

		Context("when the store is reachable", func() {
			var cancelError error
			var taskAfterCancel *models.Task

			JustBeforeEach(func() {
				cancelError = bbs.CancelTask(task.TaskGuid)
				taskAfterCancel, _ = bbs.TaskByGuid(task.TaskGuid)
			})

			itMarksTaskAsCancelled := func() {
				It("does not error", func() {
					Ω(cancelError).ShouldNot(HaveOccurred())
				})

				It("marks the task as completed", func() {
					Ω(taskAfterCancel.State).Should(Equal(models.TaskStateCompleted))
				})

				It("marks the task as failed", func() {
					Ω(taskAfterCancel.Failed).Should(BeTrue())
				})

				It("sets the failure reason to cancelled", func() {
					Ω(taskAfterCancel.FailureReason).Should(Equal("task was cancelled"))
				})

				It("bumps UpdatedAt", func() {
					Ω(taskAfterCancel.UpdatedAt).Should(Equal(timeProvider.Now().UnixNano()))
				})
			}

			Context("when the task is in pending state", func() {
				BeforeEach(func() {
					err := bbs.DesireTask(task)
					Ω(err).ShouldNot(HaveOccurred())
				})

				itMarksTaskAsCancelled()
			})

			Context("when the task is in running state", func() {
				BeforeEach(func() {
					err := bbs.DesireTask(task)
					Ω(err).ShouldNot(HaveOccurred())
					err = bbs.StartTask(task.TaskGuid, "cell-id")
					Ω(err).ShouldNot(HaveOccurred())
				})

				itMarksTaskAsCancelled()
			})

			Context("when the task is in completed state", func() {
				BeforeEach(func() {
					err := bbs.DesireTask(task)
					Ω(err).ShouldNot(HaveOccurred())

					err = bbs.StartTask(task.TaskGuid, "cell-id")
					Ω(err).ShouldNot(HaveOccurred())

					err = bbs.CompleteTask(task.TaskGuid, false, "", "")
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("returns an error", func() {
					Ω(cancelError).Should(HaveOccurred())
					Ω(cancelError).Should(Equal(bbserrors.NewTaskStateTransitionError(models.TaskStateCompleted, models.TaskStateCompleted)))
				})
			})

			Context("when the task is in resolving state", func() {
				BeforeEach(func() {
					err := bbs.DesireTask(task)
					Ω(err).ShouldNot(HaveOccurred())

					err = bbs.StartTask(task.TaskGuid, "cell-id")
					Ω(err).ShouldNot(HaveOccurred())

					err = bbs.CompleteTask(task.TaskGuid, false, "", "")
					Ω(err).ShouldNot(HaveOccurred())

					err = bbs.ResolvingTask(task.TaskGuid)
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("returns an error", func() {
					Ω(cancelError).Should(HaveOccurred())
					Ω(cancelError).Should(Equal(bbserrors.NewTaskStateTransitionError(models.TaskStateResolving, models.TaskStateCompleted)))
				})
			})

			Context("when the task does not exist", func() {
				It("returns an error", func() {
					Ω(cancelError).Should(HaveOccurred())
					Ω(cancelError).Should(Equal(bbserrors.ErrStoreResourceNotFound))
				})
			})

			Context("when the store returns some error other than key not found or timeout", func() {
				var storeError = errors.New("store error")

				BeforeEach(func() {
					fakeStoreAdapter := fakestoreadapter.New()
					fakeStoreAdapter.GetErrInjector = fakestoreadapter.NewFakeStoreAdapterErrorInjector(``, storeError)

					bbs = New(fakeStoreAdapter, timeProvider, fakeTaskClient, fakeAuctioneerClient, servicesBBS, lagertest.NewTestLogger("test"))
				})

				It("returns an error", func() {
					Ω(cancelError).Should(HaveOccurred())
					Ω(cancelError).Should(Equal(storeError))
				})
			})
		})

		Context("when the store is out of commission", func() {
			BeforeEach(func() {
				err := bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())
			})

			itRetriesUntilStoreComesBack(func() error {
				return bbs.CancelTask(task.TaskGuid)
			})
		})
	})

	Describe("CompleteTask", func() {
		BeforeEach(func() {
			task = models.Task{
				TaskGuid:  "some-guid",
				Domain:    "tests",
				Stack:     "pancakes",
				Action:    dummyAction,
				CreatedAt: 1234812,
			}
		})

		Context("when completing a pending Task", func() {
			JustBeforeEach(func() {
				err := bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("sets the Task in the completed state", func() {
				err := bbs.CompleteTask(task.TaskGuid, true, "because i said so", "a result")
				Ω(err).ShouldNot(HaveOccurred())

				tasks, err := bbs.CompletedTasks()
				Ω(err).ShouldNot(HaveOccurred())

				Ω(tasks[0].Failed).Should(BeTrue())
				Ω(tasks[0].FailureReason).Should(Equal("because i said so"))
			})

			It("should bump UpdatedAt", func() {
				timeProvider.IncrementBySeconds(1)

				err := bbs.CompleteTask(task.TaskGuid, true, "because i said so", "a result")
				Ω(err).ShouldNot(HaveOccurred())

				tasks, err := bbs.CompletedTasks()
				Ω(err).ShouldNot(HaveOccurred())

				Ω(tasks[0].UpdatedAt).Should(Equal(timeProvider.Now().UnixNano()))
			})

			It("sets FirstCompletedAt", func() {
				timeProvider.IncrementBySeconds(1)

				err := bbs.CompleteTask(task.TaskGuid, true, "because i said so", "a result")
				Ω(err).ShouldNot(HaveOccurred())

				tasks, err := bbs.CompletedTasks()
				Ω(err).ShouldNot(HaveOccurred())

				Ω(tasks[0].FirstCompletedAt).Should(Equal(timeProvider.Now().UnixNano()))
			})

			Context("when a receptor is present", func() {
				var receptorPresence ifrit.Process

				BeforeEach(func() {
					presence := models.ReceptorPresence{
						ReceptorID:  "some-receptor",
						ReceptorURL: "some-receptor-url",
					}

					heartbeat := servicesBBS.NewReceptorHeartbeat(presence, 1*time.Second)

					receptorPresence = ifrit.Invoke(heartbeat)
				})

				AfterEach(func() {
					ginkgomon.Interrupt(receptorPresence)
				})

				Context("and completing succeeds", func() {
					BeforeEach(func() {
						fakeTaskClient.CompleteTaskReturns(nil)
					})

					Context("and the task has a complete URL", func() {
						BeforeEach(func() {
							task.CompletionCallbackURL = &url.URL{Host: "bogus"}
						})

						It("completes the task using its address", func() {
							err := bbs.CompleteTask(task.TaskGuid, true, "because", "a result")
							Ω(err).ShouldNot(HaveOccurred())

							Ω(fakeTaskClient.CompleteTaskCallCount()).Should(Equal(1))
							receptorURL, completedTask := fakeTaskClient.CompleteTaskArgsForCall(0)
							Ω(receptorURL).Should(Equal("some-receptor-url"))
							Ω(completedTask.TaskGuid).Should(Equal(task.TaskGuid))
							Ω(completedTask.Failed).Should(BeTrue())
							Ω(completedTask.FailureReason).Should(Equal("because"))
							Ω(completedTask.Result).Should(Equal("a result"))
						})
					})

					Context("but the task has no complete URL", func() {
						BeforeEach(func() {
							task.CompletionCallbackURL = nil
						})

						It("does not complete the task via the receptor", func() {
							err := bbs.CompleteTask(task.TaskGuid, true, "because", "a result")
							Ω(err).ShouldNot(HaveOccurred())

							Ω(fakeTaskClient.CompleteTaskCallCount()).Should(BeZero())
						})
					})
				})

				Context("and completing fails", func() {
					BeforeEach(func() {
						fakeTaskClient.CompleteTaskReturns(errors.New("welp"))
					})

					It("swallows the error, as we'll retry again eventually (via convergence)", func() {
						err := bbs.CompleteTask(task.TaskGuid, true, "because", "a result")
						Ω(err).ShouldNot(HaveOccurred())
					})
				})
			})

			Context("when no receptors are present", func() {
				It("swallows the error, as we'll retry again eventually (via convergence)", func() {
					err := bbs.CompleteTask(task.TaskGuid, true, "because", "a result")
					Ω(err).ShouldNot(HaveOccurred())
				})
			})

			Context("when the store is out of commission", func() {
				itRetriesUntilStoreComesBack(func() error {
					return bbs.CompleteTask(task.TaskGuid, false, "", "a result")
				})
			})
		})

		Context("when completing a running Task", func() {
			JustBeforeEach(func() {
				err := bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.StartTask(task.TaskGuid, "cell-ID")
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("sets the Task in the completed state", func() {
				err := bbs.CompleteTask(task.TaskGuid, true, "because i said so", "a result")
				Ω(err).ShouldNot(HaveOccurred())

				tasks, err := bbs.CompletedTasks()
				Ω(err).ShouldNot(HaveOccurred())

				Ω(tasks[0].Failed).Should(BeTrue())
				Ω(tasks[0].FailureReason).Should(Equal("because i said so"))
			})

			It("should bump UpdatedAt", func() {
				timeProvider.IncrementBySeconds(1)

				err := bbs.CompleteTask(task.TaskGuid, true, "because i said so", "a result")
				Ω(err).ShouldNot(HaveOccurred())

				tasks, err := bbs.CompletedTasks()
				Ω(err).ShouldNot(HaveOccurred())

				Ω(tasks[0].UpdatedAt).Should(Equal(timeProvider.Now().UnixNano()))
			})

			It("sets FirstCompletedAt", func() {
				timeProvider.IncrementBySeconds(1)

				err := bbs.CompleteTask(task.TaskGuid, true, "because i said so", "a result")
				Ω(err).ShouldNot(HaveOccurred())

				tasks, err := bbs.CompletedTasks()
				Ω(err).ShouldNot(HaveOccurred())

				Ω(tasks[0].FirstCompletedAt).Should(Equal(timeProvider.Now().UnixNano()))
			})

			Context("when a receptor is present", func() {
				var receptorPresence ifrit.Process

				BeforeEach(func() {
					presence := models.ReceptorPresence{
						ReceptorID:  "some-receptor",
						ReceptorURL: "some-receptor-url",
					}

					heartbeat := servicesBBS.NewReceptorHeartbeat(presence, 1*time.Second)

					receptorPresence = ifrit.Invoke(heartbeat)
				})

				AfterEach(func() {
					ginkgomon.Interrupt(receptorPresence)
				})

				Context("and completing succeeds", func() {
					BeforeEach(func() {
						fakeTaskClient.CompleteTaskReturns(nil)
					})

					Context("and the task has a complete URL", func() {
						BeforeEach(func() {
							task.CompletionCallbackURL = &url.URL{Host: "bogus"}
						})

						It("completes the task using its address", func() {
							err := bbs.CompleteTask(task.TaskGuid, true, "because", "a result")
							Ω(err).ShouldNot(HaveOccurred())

							Ω(fakeTaskClient.CompleteTaskCallCount()).Should(Equal(1))
							receptorURL, completedTask := fakeTaskClient.CompleteTaskArgsForCall(0)
							Ω(receptorURL).Should(Equal("some-receptor-url"))
							Ω(completedTask.TaskGuid).Should(Equal(task.TaskGuid))
							Ω(completedTask.Failed).Should(BeTrue())
							Ω(completedTask.FailureReason).Should(Equal("because"))
							Ω(completedTask.Result).Should(Equal("a result"))
						})
					})

					Context("but the task has no complete URL", func() {
						BeforeEach(func() {
							task.CompletionCallbackURL = nil
						})

						It("does not complete the task via the receptor", func() {
							err := bbs.CompleteTask(task.TaskGuid, true, "because", "a result")
							Ω(err).ShouldNot(HaveOccurred())

							Ω(fakeTaskClient.CompleteTaskCallCount()).Should(BeZero())
						})
					})
				})

				Context("and completing fails", func() {
					BeforeEach(func() {
						fakeTaskClient.CompleteTaskReturns(errors.New("welp"))
					})

					It("swallows the error, as we'll retry again eventually (via convergence)", func() {
						err := bbs.CompleteTask(task.TaskGuid, true, "because", "a result")
						Ω(err).ShouldNot(HaveOccurred())
					})
				})
			})

			Context("when no receptors are present", func() {
				It("swallows the error, as we'll retry again eventually (via convergence)", func() {
					err := bbs.CompleteTask(task.TaskGuid, true, "because", "a result")
					Ω(err).ShouldNot(HaveOccurred())
				})
			})

			Context("when the store is out of commission", func() {
				itRetriesUntilStoreComesBack(func() error {
					return bbs.CompleteTask(task.TaskGuid, false, "", "a result")
				})
			})
		})

		Context("When completing a Task that is already completed", func() {
			BeforeEach(func() {
				err := bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.StartTask(task.TaskGuid, "cell-ID")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.CompleteTask(task.TaskGuid, true, "some failure reason", "")
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("returns an error", func() {
				err := bbs.CompleteTask(task.TaskGuid, true, "another failure reason", "")
				Ω(err).Should(HaveOccurred())
			})
		})

		Context("When completing a Task that is resolving", func() {
			BeforeEach(func() {
				err := bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.StartTask(task.TaskGuid, "cell-ID")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.CompleteTask(task.TaskGuid, false, "", "result")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.ResolvingTask(task.TaskGuid)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("returns an error", func() {
				err := bbs.CompleteTask(task.TaskGuid, false, "", "another result")
				Ω(err).Should(HaveOccurred())
			})
		})
	})

	Describe("ResolvingTask", func() {
		BeforeEach(func() {
			task = models.Task{
				TaskGuid:  "some-guid",
				Domain:    "tests",
				Stack:     "pancakes",
				Action:    dummyAction,
				CreatedAt: 1234812,
			}
		})

		Context("when the task is complete", func() {
			BeforeEach(func() {
				err := bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.StartTask(task.TaskGuid, "some-cell-id")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.CompleteTask(task.TaskGuid, true, "because i said so", "a result")
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("swaps /task/<guid>'s state to resolving", func() {
				err := bbs.ResolvingTask(task.TaskGuid)
				Ω(err).ShouldNot(HaveOccurred())

				tasks, err := bbs.ResolvingTasks()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(tasks[0].TaskGuid).Should(Equal(task.TaskGuid))
				Ω(tasks[0].State).Should(Equal(models.TaskStateResolving))
			})

			It("bumps UpdatedAt", func() {
				timeProvider.IncrementBySeconds(1)

				err := bbs.ResolvingTask(task.TaskGuid)
				Ω(err).ShouldNot(HaveOccurred())

				tasks, err := bbs.ResolvingTasks()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(tasks[0].UpdatedAt).Should(Equal(timeProvider.Now().UnixNano()))
			})

			Context("when the Task is already resolving", func() {
				BeforeEach(func() {
					err := bbs.ResolvingTask(task.TaskGuid)
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("fails", func() {
					err := bbs.ResolvingTask(task.TaskGuid)
					Ω(err).Should(HaveOccurred())
				})
			})
		})

		Context("when the task is not complete", func() {
			BeforeEach(func() {
				err := bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.StartTask(task.TaskGuid, "some-cell-id")
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("should fail", func() {
				err := bbs.ResolvingTask(task.TaskGuid)
				Ω(err).Should(Equal(bbserrors.NewTaskStateTransitionError(models.TaskStateRunning, models.TaskStateResolving)))
			})
		})
	})

	Describe("ResolveTask", func() {
		BeforeEach(func() {
			task = models.Task{
				TaskGuid:  "some-guid",
				Domain:    "tests",
				Stack:     "pancakes",
				Action:    dummyAction,
				CreatedAt: 1234812,
			}
		})

		Context("when the task is resolving", func() {
			BeforeEach(func() {
				err := bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.StartTask(task.TaskGuid, "some-cell-id")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.CompleteTask(task.TaskGuid, true, "because i said so", "a result")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.ResolvingTask(task.TaskGuid)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("should remove /task/<guid>", func() {
				err := bbs.ResolveTask(task.TaskGuid)
				Ω(err).ShouldNot(HaveOccurred())

				tasks, err := bbs.Tasks()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(tasks).Should(BeEmpty())
			})

			Context("when the store is out of commission", func() {
				itRetriesUntilStoreComesBack(func() error {
					return bbs.ResolveTask(task.TaskGuid)
				})
			})
		})

		Context("when the task is not resolving", func() {
			BeforeEach(func() {
				err := bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.StartTask(task.TaskGuid, "some-cell-id")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.CompleteTask(task.TaskGuid, true, "because i said so", "a result")
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("should fail", func() {
				err := bbs.ResolveTask(task.TaskGuid)
				Ω(err).Should(HaveOccurred())
			})
		})
	})
})
