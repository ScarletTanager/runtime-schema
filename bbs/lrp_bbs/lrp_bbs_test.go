package lrp_bbs_test

import (
	"fmt"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LRP", func() {
	const cellID = "some-cell-id"

	var lrp models.DesiredLRP

	BeforeEach(func() {
		lrp = models.DesiredLRP{
			Domain:      "tests",
			ProcessGuid: "some-process-guid",
			Instances:   5,
			Stack:       "some-stack",
			MemoryMB:    1024,
			DiskMB:      512,
			Routes:      []string{"route-1", "route-2"},
			Action: &models.DownloadAction{
				From: "http://example.com",
				To:   "/tmp/internet",
			},
		}
	})

	Describe("DesireLRP", func() {
		Context("when the desired LRP does not yet exist", func() {
			It("creates /v1/desired/<process-guid>", func() {
				err := bbs.DesireLRP(logger, lrp)
				Ω(err).ShouldNot(HaveOccurred())

				node, err := etcdClient.Get("/v1/desired/some-process-guid")
				Ω(err).ShouldNot(HaveOccurred())
				expected, err := models.ToJSON(lrp)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(node.Value).Should(Equal(expected))
			})

			Context("when an auctioneer is present", func() {
				BeforeEach(func() {
					auctioneerPresence := models.NewAuctioneerPresence("auctioneer-id", "example.com")
					registerAuctioneer(auctioneerPresence)
				})

				It("emits start auction requests", func() {
					originalAuctionCallCount := fakeAuctioneerClient.RequestLRPAuctionsCallCount()

					err := bbs.DesireLRP(logger, lrp)
					Ω(err).ShouldNot(HaveOccurred())

					Consistently(fakeAuctioneerClient.RequestLRPAuctionsCallCount).Should(Equal(originalAuctionCallCount + 1))

					_, startAuctions := fakeAuctioneerClient.RequestLRPAuctionsArgsForCall(originalAuctionCallCount)
					Ω(startAuctions).Should(HaveLen(1))
					Ω(startAuctions[0].DesiredLRP).Should(Equal(lrp))
					Ω(startAuctions[0].Indices).Should(HaveLen(5))
					for i := uint(0); i < 5; i++ {
						Ω(startAuctions[0].Indices).Should(ContainElement(i))
					}
				})
			})
		})

		Context("when the desired LRP does exist", func() {
			var newLRP models.DesiredLRP

			BeforeEach(func() {
				err := bbs.DesireLRP(logger, lrp)
				Ω(err).ShouldNot(HaveOccurred())

				newLRP = lrp
			})

			Context("when the modifications are valid", func() {
				BeforeEach(func() {
					newLRP.Instances = 6
					newLRP.Routes = []string{"example.com", "foobar"}
				})

				It("updates the desired lrp", func() {
					err := bbs.DesireLRP(logger, newLRP)
					Ω(err).ShouldNot(HaveOccurred())

					current, err := bbs.DesiredLRPByProcessGuid(lrp.ProcessGuid)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(current).Should(Equal(newLRP))
				})

				Context("when scaling up", func() {
					BeforeEach(func() {
						newLRP.Instances = 6
					})

					Context("when an auctioneer is present", func() {
						BeforeEach(func() {
							auctioneerPresence := models.NewAuctioneerPresence("auctioneer-id", "example.com")
							registerAuctioneer(auctioneerPresence)
						})

						It("emits start auction requests", func() {
							originalAuctionCallCount := fakeAuctioneerClient.RequestLRPAuctionsCallCount()

							err := bbs.DesireLRP(logger, newLRP)
							Ω(err).ShouldNot(HaveOccurred())

							Consistently(fakeAuctioneerClient.RequestLRPAuctionsCallCount).Should(Equal(originalAuctionCallCount + 1))

							_, startAuctions := fakeAuctioneerClient.RequestLRPAuctionsArgsForCall(originalAuctionCallCount)
							Ω(startAuctions).Should(HaveLen(1))
							Ω(startAuctions[0].DesiredLRP).Should(Equal(newLRP))
							Ω(startAuctions[0].Indices).Should(ConsistOf(uint(5)))
						})
					})
				})

				Context("when scaling down", func() {
					BeforeEach(func() {
						newLRP.Instances = 2
					})

					Context("when there are running instances on a present cell", func() {
						cellPresence := models.CellPresence{
							CellID:     "the-cell-id",
							Stack:      "the-stack",
							RepAddress: "cell.example.com",
						}

						BeforeEach(func() {
							registerCell(cellPresence)

							for i := 0; i < lrp.Instances; i++ {
								err := bbs.ClaimActualLRP(
									models.NewActualLRPKey(lrp.ProcessGuid, i, lrp.Domain),
									models.NewActualLRPContainerKey(fmt.Sprintf("some-instance-guid-%d", i), cellPresence.CellID),
									logger,
								)
								Ω(err).ShouldNot(HaveOccurred())
							}
						})

						It("stops the instances at the removed indices", func() {
							originalStopCallCount := fakeCellClient.StopLRPInstanceCallCount()

							err := bbs.DesireLRP(logger, newLRP)
							Ω(err).ShouldNot(HaveOccurred())

							Ω(fakeCellClient.StopLRPInstanceCallCount()).Should(Equal(originalStopCallCount + (lrp.Instances - newLRP.Instances)))

							stoppedActuals := make([]int, lrp.Instances-newLRP.Instances)
							for i := 0; i < (lrp.Instances - newLRP.Instances); i++ {
								addr, key, _ := fakeCellClient.StopLRPInstanceArgsForCall(originalStopCallCount + i)
								Ω(addr).Should(Equal(cellPresence.RepAddress))

								stoppedActuals[i] = key.Index
							}

							Ω(stoppedActuals).Should(ConsistOf([]int{2, 3, 4}))
						})
					})
				})
			})

			Context("when the modifications are invalid", func() {
				BeforeEach(func() {
					newLRP.Stack = "foo"
				})

				It("fails to update the desired lrp", func() {
					err := bbs.DesireLRP(logger, newLRP)
					Ω(err).Should(HaveOccurred())

					current, err := bbs.DesiredLRPByProcessGuid(lrp.ProcessGuid)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(current).Should(Equal(lrp))
				})
			})
		})

		Context("with an invalid LRP", func() {
			var desireError error

			BeforeEach(func() {
				lrp.Domain = ""
				desireError = bbs.DesireLRP(logger, lrp)
			})

			It("returns an error", func() {
				Ω(desireError).Should(HaveOccurred())
				Ω(desireError).Should(BeAssignableToTypeOf(*new(models.ValidationError)))
			})
		})

		Context("when the store is out of commission", func() {
			itRetriesUntilStoreComesBack(func() error {
				return bbs.DesireLRP(logger, lrp)
			})
		})
	})

	Describe("RemoveDesiredLRPByProcessGuid", func() {
		Context("when the desired LRP exists", func() {
			BeforeEach(func() {
				err := bbs.DesireLRP(logger, lrp)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("should delete it", func() {
				err := bbs.RemoveDesiredLRPByProcessGuid(logger, lrp.ProcessGuid)
				Ω(err).ShouldNot(HaveOccurred())

				_, err = etcdClient.Get("/v1/desired/some-process-guid")
				Ω(err).Should(MatchError(storeadapter.ErrorKeyNotFound))
			})

			Context("when there are running instances on a present cell", func() {
				cellPresence := models.CellPresence{
					CellID:     "the-cell-id",
					Stack:      "the-stack",
					RepAddress: "cell.example.com",
				}

				BeforeEach(func() {
					registerCell(cellPresence)

					for i := 0; i < lrp.Instances; i++ {
						err := bbs.ClaimActualLRP(
							models.NewActualLRPKey(lrp.ProcessGuid, i, lrp.Domain),
							models.NewActualLRPContainerKey(fmt.Sprintf("some-instance-guid-%d", i), cellPresence.CellID),
							logger,
						)
						Ω(err).ShouldNot(HaveOccurred())
					}
				})

				It("stops all actual lrps for the desired lrp", func() {
					originalStopCallCount := fakeCellClient.StopLRPInstanceCallCount()

					err := bbs.RemoveDesiredLRPByProcessGuid(logger, lrp.ProcessGuid)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(fakeCellClient.StopLRPInstanceCallCount()).Should(Equal(originalStopCallCount + (lrp.Instances)))

					stoppedActuals := make([]int, lrp.Instances)
					for i := 0; i < lrp.Instances; i++ {
						addr, key, _ := fakeCellClient.StopLRPInstanceArgsForCall(originalStopCallCount + i)
						Ω(addr).Should(Equal(cellPresence.RepAddress))

						stoppedActuals[i] = key.Index
					}

					Ω(stoppedActuals).Should(ConsistOf([]int{0, 1, 2, 3, 4}))
				})
			})
		})

		Context("when the desired LRP does not exist", func() {
			It("returns an ErrorKeyNotFound", func() {
				err := bbs.RemoveDesiredLRPByProcessGuid(logger, "monkey")
				Ω(err).Should(MatchError(bbserrors.ErrStoreResourceNotFound))
			})
		})
	})

	Describe("Updating DesireLRP", func() {
		var update models.DesiredLRPUpdate

		BeforeEach(func() {
			err := bbs.DesireLRP(logger, lrp)
			Ω(err).ShouldNot(HaveOccurred())

			update = models.DesiredLRPUpdate{}
		})

		Context("When the updates are valid", func() {
			BeforeEach(func() {
				annotation := "new-annotation"
				instances := 16

				update.Routes = []string{"new-route-1", "new-route-2"}
				update.Annotation = &annotation
				update.Instances = &instances
			})

			It("updates an existing DesireLRP", func() {
				err := bbs.UpdateDesiredLRP(logger, lrp.ProcessGuid, update)
				Ω(err).ShouldNot(HaveOccurred())

				updated, err := bbs.DesiredLRPByProcessGuid(lrp.ProcessGuid)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(updated.Routes).Should(Equal(update.Routes))
				Ω(updated.Annotation).Should(Equal(*update.Annotation))
				Ω(updated.Instances).Should(Equal(*update.Instances))
			})

			Context("when the instances are increased", func() {
				BeforeEach(func() {
					instances := 6
					update.Instances = &instances
				})

				Context("when an auctioneer is present", func() {
					BeforeEach(func() {
						auctioneerPresence := models.NewAuctioneerPresence("auctioneer-id", "example.com")
						registerAuctioneer(auctioneerPresence)
					})

					It("emits start auction requests", func() {
						originalAuctionCallCount := fakeAuctioneerClient.RequestLRPAuctionsCallCount()

						err := bbs.UpdateDesiredLRP(logger, lrp.ProcessGuid, update)
						Ω(err).ShouldNot(HaveOccurred())

						Consistently(fakeAuctioneerClient.RequestLRPAuctionsCallCount).Should(Equal(originalAuctionCallCount + 1))

						updated, err := bbs.DesiredLRPByProcessGuid(lrp.ProcessGuid)
						Ω(err).ShouldNot(HaveOccurred())

						_, startAuctions := fakeAuctioneerClient.RequestLRPAuctionsArgsForCall(originalAuctionCallCount)
						Ω(startAuctions).Should(HaveLen(1))
						Ω(startAuctions[0].DesiredLRP).Should(Equal(updated))
						Ω(startAuctions[0].Indices).Should(HaveLen(1))
						Ω(startAuctions[0].Indices).Should(ContainElement(uint(5)))
					})
				})
			})

			Context("when the instances are decreased", func() {
				BeforeEach(func() {
					instances := 2
					update.Instances = &instances
				})

				Context("when the cell is present", func() {
					cellPresence := models.CellPresence{
						CellID:     "the-cell-id",
						Stack:      "the-stack",
						RepAddress: "cell.example.com",
					}

					BeforeEach(func() {
						registerCell(cellPresence)

						for i := 0; i < lrp.Instances; i++ {
							err := bbs.ClaimActualLRP(
								models.NewActualLRPKey(lrp.ProcessGuid, i, lrp.Domain),
								models.NewActualLRPContainerKey(fmt.Sprintf("some-instance-guid-%d", i), cellPresence.CellID),
								logger,
							)
							Ω(err).ShouldNot(HaveOccurred())
						}
					})

					It("stops the instances at the removed indices", func() {
						originalStopCallCount := fakeCellClient.StopLRPInstanceCallCount()

						err := bbs.UpdateDesiredLRP(logger, lrp.ProcessGuid, update)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(fakeCellClient.StopLRPInstanceCallCount()).Should(Equal(originalStopCallCount + (lrp.Instances - *update.Instances)))

						stoppedActuals := make([]int, lrp.Instances-*update.Instances)
						for i := 0; i < (lrp.Instances - *update.Instances); i++ {
							addr, key, _ := fakeCellClient.StopLRPInstanceArgsForCall(originalStopCallCount + i)
							Ω(addr).Should(Equal(cellPresence.RepAddress))

							stoppedActuals[i] = key.Index
						}

						Ω(stoppedActuals).Should(ConsistOf([]int{2, 3, 4}))
					})
				})
			})
		})

		Context("When the updates are invalid", func() {
			It("instances cannot be less than zero", func() {
				instances := -1

				update := models.DesiredLRPUpdate{
					Instances: &instances,
				}

				err := bbs.UpdateDesiredLRP(logger, lrp.ProcessGuid, update)
				Ω(err).Should(HaveOccurred())
				Ω(err.Error()).Should(ContainSubstring("instances"))

				updated, err := bbs.DesiredLRPByProcessGuid(lrp.ProcessGuid)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(updated).Should(Equal(lrp))
			})
		})

		Context("When the LRP does not exist", func() {
			It("returns an ErrorKeyNotFound", func() {
				instances := 0

				err := bbs.UpdateDesiredLRP(logger, "garbage-guid", models.DesiredLRPUpdate{
					Instances: &instances,
				})
				Ω(err).Should(Equal(bbserrors.ErrStoreResourceNotFound))
			})
		})
	})
})
