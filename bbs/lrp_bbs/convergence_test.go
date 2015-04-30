package lrp_bbs_test

import (
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/dropsonde/metric_sender/fake"
	"github.com/cloudfoundry/dropsonde/metrics"
	"github.com/cloudfoundry/storeadapter"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("LrpConvergence", func() {
	const freshDomain = "some-fresh-domain"
	var dummyAction = &models.DownloadAction{
		From: "http://example.com",
		To:   "/tmp/internet",
	}

	var (
		sender *fake.FakeMetricSender
	)

	BeforeEach(func() {
		sender = fake.NewFakeMetricSender()
		metrics.Initialize(sender)

		err := domainBBS.UpsertDomain(freshDomain, 0)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("convergence counters", func() {
		It("bumps the convergence counter", func() {
			Expect(sender.GetCounter("ConvergenceLRPRuns")).To(Equal(uint64(0)))
			bbs.ConvergeLRPs(logger, servicesBBS.NewCellsLoader())
			Expect(sender.GetCounter("ConvergenceLRPRuns")).To(Equal(uint64(1)))
			bbs.ConvergeLRPs(logger, servicesBBS.NewCellsLoader())
			Expect(sender.GetCounter("ConvergenceLRPRuns")).To(Equal(uint64(2)))
		})

		It("reports the duration that it took to converge", func() {
			clock.IntervalToAdvance = 500 * time.Nanosecond
			bbs.ConvergeLRPs(logger, servicesBBS.NewCellsLoader())

			reportedDuration := sender.GetValue("ConvergenceLRPDuration")
			Expect(reportedDuration.Unit).To(Equal("nanos"))
			Expect(reportedDuration.Value).NotTo(BeZero())
		})
	})

	Describe("converging missing actual LRPs", func() {
		const processGuid = "process-guid-for-missing"
		const cellId = "cell-id"
		var desiredLRP models.DesiredLRP

		BeforeEach(func() {
			desiredLRP = models.DesiredLRP{
				ProcessGuid: processGuid,
				Instances:   2,
				Domain:      freshDomain,
				RootFS:      "some:rootfs",
				Action:      dummyAction,
			}

			setRawDesiredLRP(desiredLRP)
			registerCell(models.NewCellPresence(cellId, "example.com", "the-zone", models.NewCellCapacity(128, 1024, 3)))
			registerAuctioneer(models.NewAuctioneerPresence(cellId, "example.com"))
		})

		JustBeforeEach(func() {
			bbs.ConvergeLRPs(logger, servicesBBS.NewCellsLoader())
		})

		It("logs", func() {
			Expect(logger.TestSink).To(gbytes.Say("adding-start-auction"))
		})

		It("logs the convergence", func() {
			logMessages := logger.TestSink.LogMessages()
			Expect(logMessages).To(ContainElement("test.converge-lrps.starting-convergence"))
			Expect(logMessages).To(ContainElement("test.converge-lrps.finished-convergence"))
		})

		Context("when there are no actuals for desired LRP", func() {
			It("emits a start auction request for the correct indices", func() {
				Expect(fakeAuctioneerClient.RequestLRPAuctionsCallCount()).To(Equal(1))

				_, startAuctions := fakeAuctioneerClient.RequestLRPAuctionsArgsForCall(0)
				Expect(startAuctions).To(HaveLen(1))
				Expect(startAuctions[0].DesiredLRP).To(Equal(desiredLRP))
				Expect(startAuctions[0].Indices).To(ConsistOf(uint(0), uint(1)))
			})
		})

		Context("when there are fewer actuals for desired LRP", func() {
			BeforeEach(func() {
				actualLRP := models.ActualLRP{
					ActualLRPKey:         models.NewActualLRPKey(desiredLRP.ProcessGuid, 0, desiredLRP.Domain),
					ActualLRPInstanceKey: models.NewActualLRPInstanceKey("some-instance-guid", cellId),
					ActualLRPNetInfo:     defaultNetInfo(),
					State:                models.ActualLRPStateRunning,
					Since:                clock.Now().Add(-time.Minute).UnixNano(),
				}
				setRawActualLRP(actualLRP)
			})

			It("emits a start auction request for the missing index", func() {
				Expect(fakeAuctioneerClient.RequestLRPAuctionsCallCount()).To(Equal(1))

				_, startAuctions := fakeAuctioneerClient.RequestLRPAuctionsArgsForCall(0)
				Expect(startAuctions).To(HaveLen(1))
				Expect(startAuctions[0].DesiredLRP).To(Equal(desiredLRP))
				Expect(startAuctions[0].Indices).To(ConsistOf(uint(1)))
			})
		})

		Context("when instances are crashing", func() {
			const missingIndex = 0

			BeforeEach(func() {
				now := clock.Now().UnixNano()
				twentyMinutesAgo := clock.Now().Add(-20 * time.Minute).UnixNano()

				crashedRecently := models.ActualLRP{
					ActualLRPKey: models.NewActualLRPKey(desiredLRP.ProcessGuid, 0, desiredLRP.Domain),
					CrashCount:   5,
					State:        models.ActualLRPStateCrashed,
					Since:        now,
				}

				crashedLongAgo := models.ActualLRP{
					ActualLRPKey: models.NewActualLRPKey(desiredLRP.ProcessGuid, 1, desiredLRP.Domain),
					CrashCount:   5,
					State:        models.ActualLRPStateCrashed,
					Since:        twentyMinutesAgo,
				}

				setRawActualLRP(crashedRecently)
				setRawActualLRP(crashedLongAgo)
			})

			It("emits a start auction request for the crashed index", func() {
				Expect(fakeAuctioneerClient.RequestLRPAuctionsCallCount()).To(Equal(1))

				_, startAuctions := fakeAuctioneerClient.RequestLRPAuctionsArgsForCall(0)
				Expect(startAuctions).To(HaveLen(1))
				Expect(startAuctions[0].DesiredLRP).To(Equal(desiredLRP))
				Expect(startAuctions[0].Indices).To(ConsistOf(uint(1)))
			})
		})
	})

	Context("when the desired LRP has malformed JSON", func() {
		const processGuid = "bogus-desired"
		BeforeEach(func() {
			err := etcdClient.SetMulti([]storeadapter.StoreNode{
				{
					Key:   shared.DesiredLRPSchemaPathByProcessGuid(processGuid),
					Value: []byte("ß"),
				},
			})

			Expect(err).NotTo(HaveOccurred())

			bbs.ConvergeLRPs(logger, servicesBBS.NewCellsLoader())
		})

		It("should delete the bogus entry", func() {
			_, err := etcdClient.Get(shared.DesiredLRPSchemaPathByProcessGuid(processGuid))
			Expect(err).To(MatchError(storeadapter.ErrorKeyNotFound))
		})

		It("logs", func() {
			Expect(logger.TestSink).To(gbytes.Say("pruning-invalid-desired-lrp-json"))
		})
	})

	Describe("pruning LRPs by cell", func() {
		var cellPresence models.CellPresence
		var processGuid string
		var desiredLRP models.DesiredLRP
		var index int

		JustBeforeEach(func() {
			bbs.ConvergeLRPs(logger, servicesBBS.NewCellsLoader())
		})

		BeforeEach(func() {
			processGuid = "process-guid-for-pruning"

			index = 0

			desiredLRP = models.DesiredLRP{
				ProcessGuid: processGuid,
				Instances:   2,
				Domain:      freshDomain,
				RootFS:      "some:rootfs",
				Action:      dummyAction,
			}

			err := bbs.DesireLRP(logger, desiredLRP)
			Expect(err).NotTo(HaveOccurred())

			cellPresence = models.NewCellPresence("cell-id", "cell.example.com", "the-zone", models.CellCapacity{128, 1024, 3})
			registerCell(cellPresence)

			actualLRPGroup, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
			Expect(err).NotTo(HaveOccurred())

			err = bbs.ClaimActualLRP(
				logger,
				actualLRPGroup.Instance.ActualLRPKey,
				models.NewActualLRPInstanceKey("instance-guid", cellPresence.CellID),
			)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when the cell is present", func() {
			It("should not prune any LRPs", func() {
				Expect(bbs.ActualLRPs()).To(HaveLen(2))
			})
		})

		Context("when the cell goes away", func() {
			BeforeEach(func() {
				kv := consulRunner.NewClient().KV()
				pair, _, err := kv.Get(shared.CellSchemaPath(cellPresence.CellID), nil)
				Expect(err).NotTo(HaveOccurred())
				_, _, err = kv.Release(pair, nil)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should delete LRPs associated with said cell but not the unclaimed LRP", func() {
				lrps, err := bbs.ActualLRPs()
				Expect(err).NotTo(HaveOccurred())
				Expect(lrps).To(HaveLen(2))

				indices := make([]int, len(lrps))
				for i, lrp := range lrps {
					Expect(lrp.ProcessGuid).To(Equal(processGuid))
					Expect(lrp.State).To(Equal(models.ActualLRPStateUnclaimed))

					indices[i] = lrp.Index
				}

				Expect(indices).To(ConsistOf([]int{0, 1}))
			})

			It("should prune LRP directories for apps that are no longer running", func() {
				actual, err := etcdClient.ListRecursively(shared.ActualLRPSchemaRoot)
				Expect(err).NotTo(HaveOccurred())
				Expect(actual.ChildNodes).To(HaveLen(1))
				Expect(actual.ChildNodes[0].Key).To(Equal(shared.ActualLRPProcessDir(processGuid)))
			})

			It("logs", func() {
				Expect(logger.TestSink).To(gbytes.Say("missing-cell"))
			})
		})
	})

	Describe("converging extra actual LRPs", func() {
		var processGuid string
		var index int
		var domain string

		BeforeEach(func() {
			domain = freshDomain
			processGuid = "process-guid"
			index = 0
		})

		Context("when the actual LRP has no corresponding desired LRP", func() {
			JustBeforeEach(func() {

				actualUnclaimedLRP := models.ActualLRP{
					ActualLRPKey: models.NewActualLRPKey(processGuid, index, domain),
					State:        models.ActualLRPStateUnclaimed,
					Since:        clock.Now().UnixNano(),
				}

				setRawActualLRP(actualUnclaimedLRP)
			})

			Context("when the actual LRP is UNCLAIMED", func() {
				It("removes the actual LRP", func() {
					bbs.ConvergeLRPs(logger, servicesBBS.NewCellsLoader())

					_, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
					Expect(err).To(Equal(bbserrors.ErrStoreResourceNotFound))
				})

				It("logs", func() {
					bbs.ConvergeLRPs(logger, servicesBBS.NewCellsLoader())

					Expect(logger.TestSink).To(gbytes.Say("no-longer-desired"))
				})

				Context("when the LRP domain is not fresh", func() {
					BeforeEach(func() {
						domain = "expired-domain"
					})

					It("does not delete the actual LRP", func() {
						bbs.ConvergeLRPs(logger, servicesBBS.NewCellsLoader())

						_, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
						Expect(err).NotTo(HaveOccurred())

						Expect(logger.TestSink).To(gbytes.Say("skipping-unfresh-domain"))
					})
				})
			})

			Context("when the actual LRP is CLAIMED", func() {
				var cellPresence models.CellPresence

				JustBeforeEach(func() {
					cellPresence = models.NewCellPresence("cell-id", "cell.example.com", "the-zone", models.NewCellCapacity(128, 1024, 3))
					registerCell(cellPresence)

					actualLRPGroup, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
					Expect(err).NotTo(HaveOccurred())

					err = bbs.ClaimActualLRP(
						logger,
						actualLRPGroup.Instance.ActualLRPKey,
						models.NewActualLRPInstanceKey("instance-guid", cellPresence.CellID),
					)
					Expect(err).NotTo(HaveOccurred())
				})

				It("sends a stop request to the corresponding cell", func() {
					bbs.ConvergeLRPs(logger, servicesBBS.NewCellsLoader())

					addr, key, instanceKey := fakeCellClient.StopLRPInstanceArgsForCall(0)
					Expect(addr).To(Equal(cellPresence.RepAddress))
					Expect(key.ProcessGuid).To(Equal(processGuid))
					Expect(key.Index).To(Equal(index))
					Expect(instanceKey.InstanceGuid).To(Equal("instance-guid"))
				})

				It("logs", func() {
					bbs.ConvergeLRPs(logger, servicesBBS.NewCellsLoader())
					Expect(logger.TestSink).To(gbytes.Say("no-longer-desired"))
				})

				Context("when the LRP domain is not fresh", func() {
					BeforeEach(func() {
						domain = "expired-domain"
					})

					It("does not stop the actual LRP", func() {
						bbs.ConvergeLRPs(logger, servicesBBS.NewCellsLoader())

						Expect(fakeCellClient.StopLRPInstanceCallCount()).To(Equal(0))
						Expect(logger.TestSink).To(gbytes.Say("skipping-unfresh-domain"))
					})
				})
			})

			Context("when the actual LRP is RUNNING", func() {
				var cellPresence models.CellPresence

				JustBeforeEach(func() {
					cellPresence = models.NewCellPresence("cell-id", "cell.example.com", "the-zone", models.NewCellCapacity(128, 1024, 3))
					registerCell(cellPresence)

					actualLRPGroup, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
					Expect(err).NotTo(HaveOccurred())

					err = bbs.ClaimActualLRP(
						logger,
						actualLRPGroup.Instance.ActualLRPKey,
						models.NewActualLRPInstanceKey("instance-guid", cellPresence.CellID),
					)
					Expect(err).NotTo(HaveOccurred())

					err = bbs.StartActualLRP(
						logger,
						actualLRPGroup.Instance.ActualLRPKey,
						models.NewActualLRPInstanceKey("instance-guid", cellPresence.CellID),
						models.NewActualLRPNetInfo("host", []models.PortMapping{{HostPort: 1234, ContainerPort: 5678}}),
					)
					Expect(err).NotTo(HaveOccurred())
				})

				It("sends a stop request to the corresponding cell", func() {
					bbs.ConvergeLRPs(logger, servicesBBS.NewCellsLoader())

					Expect(fakeCellClient.StopLRPInstanceCallCount()).To(Equal(1))

					addr, key, instanceKey := fakeCellClient.StopLRPInstanceArgsForCall(0)
					Expect(addr).To(Equal(cellPresence.RepAddress))
					Expect(key.ProcessGuid).To(Equal(processGuid))
					Expect(key.Index).To(Equal(index))
					Expect(instanceKey.InstanceGuid).To(Equal("instance-guid"))
				})

				It("logs", func() {
					bbs.ConvergeLRPs(logger, servicesBBS.NewCellsLoader())
					Expect(logger.TestSink).To(gbytes.Say("no-longer-desired"))
					Expect(logger.TestSink).To(gbytes.Say(`test.converge-lrps.retiring-actual-lrps","log_level":0,"data":{"num-actual-lrps":1`))
				})

				Context("when the LRP domain is not fresh", func() {
					BeforeEach(func() {
						domain = "expired-domain"
					})

					It("does not stop the actual LRP", func() {
						bbs.ConvergeLRPs(logger, servicesBBS.NewCellsLoader())

						Expect(fakeCellClient.StopLRPInstanceCallCount()).To(Equal(0))
						Expect(logger.TestSink).To(gbytes.Say("skipping-unfresh-domain"))
					})
				})
			})
		})

		Context("when the actual LRP index is too large for its corresponding desired LRP", func() {
			var desiredLRP models.DesiredLRP
			var numInstances int

			BeforeEach(func() {
				processGuid = "process-guid"
				numInstances = 2
				domain = freshDomain
			})

			JustBeforeEach(func() {
				desiredLRP = models.DesiredLRP{
					ProcessGuid: processGuid,
					Instances:   numInstances,
					Domain:      domain,
					RootFS:      "some:rootfs",
					Action:      dummyAction,
				}

				err := bbs.DesireLRP(logger, desiredLRP)
				Expect(err).NotTo(HaveOccurred())
			})

			Context("when the actual LRP is UNCLAIMED", func() {
				JustBeforeEach(func() {
					index = numInstances

					higherIndexActualLRP := models.ActualLRP{
						ActualLRPKey: models.NewActualLRPKey(desiredLRP.ProcessGuid, index, desiredLRP.Domain),
						State:        models.ActualLRPStateUnclaimed,
						Since:        clock.Now().UnixNano(),
					}

					setRawActualLRP(higherIndexActualLRP)
				})

				It("removes the actual LRP", func() {
					bbs.ConvergeLRPs(logger, servicesBBS.NewCellsLoader())

					_, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
					Expect(err).To(Equal(bbserrors.ErrStoreResourceNotFound))
				})

				It("logs", func() {
					bbs.ConvergeLRPs(logger, servicesBBS.NewCellsLoader())

					Expect(logger.TestSink).To(gbytes.Say("retire-actual-lrps.remove-actual-lrp.succeeded"))
				})

				Context("when the LRP domain is not fresh", func() {
					BeforeEach(func() {
						domain = "expired-domain"
					})

					It("does not delete the actual LRP", func() {
						bbs.ConvergeLRPs(logger, servicesBBS.NewCellsLoader())

						_, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
						Expect(err).NotTo(HaveOccurred())
					})
				})
			})

			Context("when the actual LRP is CLAIMED", func() {
				var cellPresence models.CellPresence

				JustBeforeEach(func() {
					cellPresence = models.NewCellPresence("cell-id", "cell.example.com", "the-zone", models.NewCellCapacity(128, 1024, 100))
					registerCell(cellPresence)

					index = numInstances

					higherIndexActualLRP := models.ActualLRP{
						ActualLRPKey:         models.NewActualLRPKey(desiredLRP.ProcessGuid, index, desiredLRP.Domain),
						ActualLRPInstanceKey: models.NewActualLRPInstanceKey("instance-guid", "cell-id"),
						State:                models.ActualLRPStateClaimed,
						Since:                clock.Now().UnixNano(),
					}

					setRawActualLRP(higherIndexActualLRP)

					actualLRPGroup, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
					Expect(err).NotTo(HaveOccurred())

					err = bbs.ClaimActualLRP(
						logger,
						actualLRPGroup.Instance.ActualLRPKey,
						models.NewActualLRPInstanceKey("instance-guid", cellPresence.CellID),
					)
					Expect(err).NotTo(HaveOccurred())
				})

				It("sends a stop request to the corresponding cell", func() {
					bbs.ConvergeLRPs(logger, servicesBBS.NewCellsLoader())

					Expect(fakeCellClient.StopLRPInstanceCallCount()).To(Equal(1))

					addr, key, instanceKey := fakeCellClient.StopLRPInstanceArgsForCall(0)
					Expect(addr).To(Equal(cellPresence.RepAddress))
					Expect(key.ProcessGuid).To(Equal(processGuid))
					Expect(key.Index).To(Equal(index))
					Expect(instanceKey.InstanceGuid).To(Equal("instance-guid"))
				})

				It("logs", func() {
					bbs.ConvergeLRPs(logger, servicesBBS.NewCellsLoader())
					Expect(logger.TestSink).To(gbytes.Say("stopping-actual"))
				})

				Context("when the LRP domain is not fresh", func() {
					BeforeEach(func() {
						domain = "expired-domain"
					})

					It("does not stop the actual LRP", func() {
						bbs.ConvergeLRPs(logger, servicesBBS.NewCellsLoader())

						Expect(fakeCellClient.StopLRPInstanceCallCount()).To(Equal(0))
					})
				})
			})

			Context("when the actual LRP is RUNNING", func() {
				var cellPresence models.CellPresence

				JustBeforeEach(func() {
					cellPresence = models.NewCellPresence("cell-id", "cell.example.com", "the-zone", models.NewCellCapacity(124, 1024, 6))
					registerCell(cellPresence)

					index = numInstances

					higherIndexActualLRP := models.ActualLRP{
						ActualLRPKey:         models.NewActualLRPKey(desiredLRP.ProcessGuid, index, desiredLRP.Domain),
						ActualLRPInstanceKey: models.NewActualLRPInstanceKey("instance-guid", "cell-id"),
						ActualLRPNetInfo:     models.NewActualLRPNetInfo("127.0.0.1", []models.PortMapping{{8080, 80}}),
						State:                models.ActualLRPStateRunning,
						Since:                clock.Now().UnixNano(),
					}

					setRawActualLRP(higherIndexActualLRP)

					actualLRPGroup, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
					Expect(err).NotTo(HaveOccurred())

					err = bbs.ClaimActualLRP(
						logger,
						actualLRPGroup.Instance.ActualLRPKey,
						models.NewActualLRPInstanceKey("instance-guid", cellPresence.CellID),
					)
					Expect(err).NotTo(HaveOccurred())

					err = bbs.StartActualLRP(
						logger,
						actualLRPGroup.Instance.ActualLRPKey,
						models.NewActualLRPInstanceKey("instance-guid", cellPresence.CellID),
						models.NewActualLRPNetInfo("host", []models.PortMapping{{HostPort: 1234, ContainerPort: 5678}}),
					)
					Expect(err).NotTo(HaveOccurred())
				})

				It("sends a stop request to the corresponding cell", func() {
					bbs.ConvergeLRPs(logger, servicesBBS.NewCellsLoader())

					Expect(fakeCellClient.StopLRPInstanceCallCount()).To(Equal(1))

					addr, key, instanceKey := fakeCellClient.StopLRPInstanceArgsForCall(0)
					Expect(addr).To(Equal(cellPresence.RepAddress))
					Expect(key.ProcessGuid).To(Equal(processGuid))
					Expect(key.Index).To(Equal(index))
					Expect(instanceKey.InstanceGuid).To(Equal("instance-guid"))
				})

				It("logs", func() {
					bbs.ConvergeLRPs(logger, servicesBBS.NewCellsLoader())
					Expect(logger.TestSink).To(gbytes.Say("stopping-actual"))
				})

				Context("when the LRP domain is not fresh", func() {
					BeforeEach(func() {
						domain = "expired-domain"
					})

					It("does not stop the actual LRP", func() {
						bbs.ConvergeLRPs(logger, servicesBBS.NewCellsLoader())

						Expect(fakeCellClient.StopLRPInstanceCallCount()).To(Equal(0))
					})
				})
			})
		})
	})

	Describe("converging actual LRPs that are UNCLAIMED for too long", func() {
		var desiredLRP models.DesiredLRP

		BeforeEach(func() {
			desiredLRP = models.DesiredLRP{
				ProcessGuid: "process-guid-for-unclaimed",
				Domain:      freshDomain,
				Instances:   1,
				RootFS:      "some:rootfs",
				Action:      dummyAction,
			}

			err := bbs.DesireLRP(logger, desiredLRP)
			Expect(err).NotTo(HaveOccurred())

			desiredLRP, err = bbs.DesiredLRPByProcessGuid("process-guid-for-unclaimed")
			Expect(err).NotTo(HaveOccurred())

			auctioneerPresence := models.NewAuctioneerPresence("auctioneer-id", "example.com")
			registerAuctioneer(auctioneerPresence)

			clock.Increment(models.StaleUnclaimedActualLRPDuration + 1*time.Second)
		})

		It("logs", func() {
			bbs.ConvergeLRPs(logger, servicesBBS.NewCellsLoader())

			Expect(logger.TestSink).To(gbytes.Say("adding-start-auction"))
		})

		It("re-emits start auction requests", func() {
			originalAuctionCallCount := fakeAuctioneerClient.RequestLRPAuctionsCallCount()
			bbs.ConvergeLRPs(logger, servicesBBS.NewCellsLoader())
			Consistently(fakeAuctioneerClient.RequestLRPAuctionsCallCount).Should(Equal(originalAuctionCallCount + 1))

			_, startAuctions := fakeAuctioneerClient.RequestLRPAuctionsArgsForCall(originalAuctionCallCount)
			Expect(startAuctions).To(HaveLen(1))
			Expect(startAuctions[0].DesiredLRP).To(Equal(desiredLRP))
			Expect(startAuctions[0].Indices).To(ConsistOf(uint(0)))
		})
	})
})
