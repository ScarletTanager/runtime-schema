package lrp_bbs_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/lrp_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LrpGetters", func() {
	var (
		desiredLrp1 models.DesiredLRP
		desiredLrp2 models.DesiredLRP
		desiredLrp3 models.DesiredLRP

		runningLrp1 models.ActualLRP
		runningLrp2 models.ActualLRP
		runningLrp3 models.ActualLRP
		lrpToClaim  models.ActualLRP

		newLrp *models.ActualLRP
	)

	BeforeEach(func() {
		desiredLrp1 = models.DesiredLRP{
			Domain:      "tests",
			ProcessGuid: "guidA",
			Stack:       "stack",
			Instances:   1,
			Action: &models.DownloadAction{
				From: "http://example.com",
				To:   "/tmp/internet",
			},
		}

		desiredLrp2 = models.DesiredLRP{
			Domain:      "tests",
			ProcessGuid: "guidB",
			Stack:       "stack",
			Instances:   1,
			Action: &models.DownloadAction{
				From: "http://example.com",
				To:   "/tmp/internet",
			},
		}

		desiredLrp3 = models.DesiredLRP{
			Domain:      "tests",
			ProcessGuid: "guidC",
			Stack:       "stack",
			Instances:   1,
			Action: &models.DownloadAction{
				From: "http://example.com",
				To:   "/tmp/internet",
			},
		}

		runningLrp1 = models.ActualLRP{
			ProcessGuid:  "guidA",
			Index:        1,
			InstanceGuid: "some-instance-guid-1",
			Domain:       "domain-a",
			State:        models.ActualLRPStateRunning,
			Since:        timeProvider.Now().UnixNano(),
			CellID:       "cell-id",
		}

		runningLrp2 = models.ActualLRP{
			ProcessGuid:  "guidB",
			Index:        2,
			InstanceGuid: "some-instance-guid-2",
			Domain:       "domain-b",
			State:        models.ActualLRPStateRunning,
			Since:        timeProvider.Now().UnixNano(),
			CellID:       "cell-id",
		}

		runningLrp3 = models.ActualLRP{
			ProcessGuid:  "guidC",
			Index:        3,
			InstanceGuid: "some-instance-guid-3",
			Domain:       "domain-b",
			State:        models.ActualLRPStateRunning,
			Since:        timeProvider.Now().UnixNano(),
			CellID:       "cell-id",
		}

		lrpToClaim = models.NewActualLRP("guidA", "some-instance-guid", "cell-id", "test", 2, models.ActualLRPStateClaimed)
	})

	Context("DesiredLRPs", func() {
		JustBeforeEach(func() {
			err := bbs.DesireLRP(desiredLrp1)
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.DesireLRP(desiredLrp2)
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.DesireLRP(desiredLrp3)
			Ω(err).ShouldNot(HaveOccurred())
		})

		Describe("DesiredLRPs", func() {
			It("returns all desired long running processes", func() {
				all, err := bbs.DesiredLRPs()
				Ω(err).ShouldNot(HaveOccurred())

				Ω(all).Should(HaveLen(3))
				Ω(all).Should(ContainElement(desiredLrp1))
				Ω(all).Should(ContainElement(desiredLrp2))
				Ω(all).Should(ContainElement(desiredLrp3))
			})
		})

		Describe("DesiredLRPsByDomain", func() {
			BeforeEach(func() {
				desiredLrp1.Domain = "domain-1"
				desiredLrp2.Domain = "domain-1"
				desiredLrp3.Domain = "domain-2"
			})

			It("returns all desired long running processes for the given domain", func() {
				byDomain, err := bbs.DesiredLRPsByDomain("domain-1")
				Ω(err).ShouldNot(HaveOccurred())
				Ω(byDomain).Should(ConsistOf([]models.DesiredLRP{desiredLrp1, desiredLrp2}))

				byDomain, err = bbs.DesiredLRPsByDomain("domain-2")
				Ω(err).ShouldNot(HaveOccurred())
				Ω(byDomain).Should(ConsistOf([]models.DesiredLRP{desiredLrp3}))
			})

			It("blows up with an empty string domain", func() {
				_, err := bbs.DesiredLRPsByDomain("")
				Ω(err).Should(Equal(lrp_bbs.ErrNoDomain))
			})
		})

		Describe("DesiredLRPByProcessGuid", func() {
			It("returns all desired long running processes", func() {
				desiredLrp, err := bbs.DesiredLRPByProcessGuid("guidA")
				Ω(err).ShouldNot(HaveOccurred())

				Ω(desiredLrp).Should(Equal(&desiredLrp1))
			})
		})
	})

	Context("ActualLRPs", func() {
		BeforeEach(func() {
			_, err := bbs.StartActualLRP(runningLrp1)
			Ω(err).ShouldNot(HaveOccurred())

			_, newLrp, err = createAndClaim(lrpToClaim)
			Ω(err).ShouldNot(HaveOccurred())

			_, err = bbs.StartActualLRP(runningLrp2)
			Ω(err).ShouldNot(HaveOccurred())
		})

		Describe("ActualLRPs", func() {
			It("returns all actual long running processes", func() {
				all, err := bbs.ActualLRPs()
				Ω(err).ShouldNot(HaveOccurred())

				Ω(all).Should(HaveLen(3))
				Ω(all).Should(ContainElement(runningLrp1))
				Ω(all).Should(ContainElement(*newLrp))
				Ω(all).Should(ContainElement(runningLrp2))
			})
		})

		Describe("ActualLRPsByCellID", func() {
			BeforeEach(func() {
				_, _, err := createAndClaim(models.NewActualLRP("some-other-process", "some-other-instance", "some-other-cell", "some-other-domain", 0, models.ActualLRPStateClaimed))
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("returns actual long running processes belongs to 'cell-id'", func() {
				actualLrpsForMainCell, err := bbs.ActualLRPsByCellID("cell-id")
				Ω(err).ShouldNot(HaveOccurred())

				Ω(actualLrpsForMainCell).Should(ConsistOf(runningLrp1, *newLrp, runningLrp2))

				actualLrpsForOtherCell, err := bbs.ActualLRPsByCellID("some-other-cell")
				Ω(err).ShouldNot(HaveOccurred())

				Ω(actualLrpsForOtherCell).Should(HaveLen(1))
			})
		})

		Describe("RunningActualLRPs", func() {
			It("returns all actual long running processes", func() {
				all, err := bbs.RunningActualLRPs()
				Ω(err).ShouldNot(HaveOccurred())

				Ω(all).Should(HaveLen(2))
				Ω(all).Should(ContainElement(runningLrp1))
				Ω(all).Should(ContainElement(runningLrp2))
			})
		})

		Describe("ActualLRPsByProcessGuid", func() {
			It("should fetch all LRPs for the specified guid", func() {
				lrps, err := bbs.ActualLRPsByProcessGuid("guidA")
				Ω(err).ShouldNot(HaveOccurred())
				Ω(lrps).Should(HaveLen(2))
				Ω(lrps).Should(ContainElement(runningLrp1))
				Ω(lrps).Should(ContainElement(*newLrp))
			})
		})

		Describe("ActualLRPByProcessGuidAndIndex", func() {
			It("should fetch all LRPs for the specified guid", func() {
				lrp, err := bbs.ActualLRPByProcessGuidAndIndex("guidA", 1)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(lrp).Should(Equal(&runningLrp1))
			})
		})

		Describe("RunningActualLRPsByProcessGuid", func() {
			It("should fetch all LRPs for the specified guid", func() {
				lrps, err := bbs.RunningActualLRPsByProcessGuid("guidA")
				Ω(err).ShouldNot(HaveOccurred())
				Ω(lrps).Should(HaveLen(1))
				Ω(lrps).Should(ContainElement(runningLrp1))
			})
		})

		Describe("ActualLRPsByDomain", func() {
			BeforeEach(func() {
				_, err := bbs.StartActualLRP(runningLrp3)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("should fetch all LRPs for the specified guid", func() {
				lrps, err := bbs.ActualLRPsByDomain("domain-b")
				Ω(err).ShouldNot(HaveOccurred())

				Ω(lrps).Should(HaveLen(2))
				Ω(lrps).ShouldNot(ContainElement(runningLrp1))
				Ω(lrps).Should(ContainElement(runningLrp2))
				Ω(lrps).Should(ContainElement(runningLrp3))
			})

			Context("when there are no actual LRPs in the requested domain", func() {
				It("returns an empty list", func() {
					lrps, err := bbs.ActualLRPsByDomain("bogus-domain")
					Ω(err).ShouldNot(HaveOccurred())

					Ω(lrps).ShouldNot(BeNil())
					Ω(lrps).Should(HaveLen(0))
				})
			})
		})
	})
})
