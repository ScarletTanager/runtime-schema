package lrp_bbs_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Actual LRP Getters", func() {
	const (
		cellID          = "cell-id"
		noExpirationTTL = 0

		baseProcessGuid  = "base-process-guid"
		baseDomain       = "base-domain"
		baseInstanceGuid = "base-instance-guid"

		baseIndex       = 1
		otherIndex      = 2
		yetAnotherIndex = 3

		evacuatingInstanceGuid = "evacuating-instance-guid"

		otherCellID            = "other-cell-id"
		otherCellIDProcessGuid = "other-cell-id-process-guid"

		otherDomainProcessGuid = "other-domain-process-guid"
		otherDomain            = "other-domain"

		otherProcessGuid = "other-process-guid"
	)

	var (
		baseLRP             models.ActualLRP
		otherIndexLRP       models.ActualLRP
		yetAnotherIndexLRP  models.ActualLRP
		evacuatingLRP       models.ActualLRP
		otherCellIDLRP      models.ActualLRP
		otherDomainLRP      models.ActualLRP
		otherProcessGuidLRP models.ActualLRP

		baseLRPKey         models.ActualLRPKey
		baseLRPInstanceKey models.ActualLRPInstanceKey

		netInfo models.ActualLRPNetInfo
	)

	BeforeEach(func() {
		baseLRPKey = models.NewActualLRPKey(baseProcessGuid, baseIndex, baseDomain)
		baseLRPInstanceKey = models.NewActualLRPInstanceKey(baseInstanceGuid, cellID)
		netInfo = models.NewActualLRPNetInfo("127.0.0.1", []models.PortMapping{{8080, 80}})

		baseLRP = models.ActualLRP{
			ActualLRPKey:         baseLRPKey,
			ActualLRPInstanceKey: baseLRPInstanceKey,
			ActualLRPNetInfo:     netInfo,
			State:                models.ActualLRPStateRunning,
			Since:                clock.Now().UnixNano(),
		}
		evacuatingLRP = models.ActualLRP{
			ActualLRPKey:         baseLRPKey,
			ActualLRPInstanceKey: models.NewActualLRPInstanceKey(evacuatingInstanceGuid, cellID),
			ActualLRPNetInfo:     netInfo,
			State:                models.ActualLRPStateRunning,
			Since:                clock.Now().UnixNano() - 1000,
		}

		otherIndexLRP = models.ActualLRP{
			ActualLRPKey:         models.NewActualLRPKey(baseProcessGuid, otherIndex, baseDomain),
			ActualLRPInstanceKey: baseLRPInstanceKey,
			State:                models.ActualLRPStateClaimed,
			Since:                clock.Now().UnixNano(),
		}

		yetAnotherIndexLRP = models.ActualLRP{
			ActualLRPKey:         models.NewActualLRPKey(baseProcessGuid, yetAnotherIndex, baseDomain),
			ActualLRPInstanceKey: baseLRPInstanceKey,
			ActualLRPNetInfo:     netInfo,
			State:                models.ActualLRPStateRunning,
			Since:                clock.Now().UnixNano(),
		}

		otherCellIDLRP = models.ActualLRP{
			ActualLRPKey:         models.NewActualLRPKey(otherCellIDProcessGuid, baseIndex, baseDomain),
			ActualLRPInstanceKey: models.NewActualLRPInstanceKey(baseInstanceGuid, otherCellID),
			ActualLRPNetInfo:     netInfo,
			State:                models.ActualLRPStateRunning,
			Since:                clock.Now().UnixNano(),
		}

		otherDomainLRP = models.ActualLRP{
			ActualLRPKey:         models.NewActualLRPKey(otherDomainProcessGuid, baseIndex, otherDomain),
			ActualLRPInstanceKey: baseLRPInstanceKey,
			ActualLRPNetInfo:     netInfo,
			State:                models.ActualLRPStateRunning,
			Since:                clock.Now().UnixNano(),
		}

		otherProcessGuidLRP = models.ActualLRP{
			ActualLRPKey:         models.NewActualLRPKey(otherProcessGuid, baseIndex, baseDomain),
			ActualLRPInstanceKey: baseLRPInstanceKey,
			State:                models.ActualLRPStateUnclaimed,
			Since:                clock.Now().UnixNano(),
		}
	})

	Describe("ActualLRPGroupsByProcessGuid", func() {
		Context("when there are both /instance and /evacuating LRPs", func() {
			BeforeEach(func() {
				testHelper.SetRawActualLRP(baseLRP)
				testHelper.SetRawActualLRP(otherIndexLRP)
				testHelper.SetRawActualLRP(yetAnotherIndexLRP)
				testHelper.SetRawEvacuatingActualLRP(yetAnotherIndexLRP, noExpirationTTL)
				testHelper.SetRawActualLRP(otherProcessGuidLRP)
			})

			It("returns all the /instance LRPs and /evacuating LRPs in groups", func() {
				actualLRPGroupsByIndex, err := lrpBBS.ActualLRPGroupsByProcessGuid(logger, baseProcessGuid)
				Expect(err).NotTo(HaveOccurred())
				Expect(actualLRPGroupsByIndex).To(Equal(models.ActualLRPGroupsByIndex{
					baseIndex:       {Instance: &baseLRP, Evacuating: nil},
					otherIndex:      {Instance: &otherIndexLRP, Evacuating: nil},
					yetAnotherIndex: {Instance: &yetAnotherIndexLRP, Evacuating: &yetAnotherIndexLRP},
				}))
			})
		})

		Context("when there are no LRPs", func() {
			BeforeEach(func() {
				// leave some intermediate directories in the store
				testHelper.SetRawActualLRP(baseLRP)
				err := lrpBBS.RemoveActualLRP(logger, baseLRPKey, baseLRPInstanceKey)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns an empty map", func() {
				actualLRPGroupsByIndex, err := lrpBBS.ActualLRPGroupsByProcessGuid(logger, baseProcessGuid)
				Expect(err).NotTo(HaveOccurred())
				Expect(actualLRPGroupsByIndex).NotTo(BeNil())
				Expect(actualLRPGroupsByIndex).To(BeEmpty())
			})
		})

		Context("when given an empty process guid", func() {
			It("returns an error", func() {
				_, err := lrpBBS.ActualLRPGroupsByProcessGuid(logger, "")
				Expect(err).To(Equal(bbserrors.ErrNoProcessGuid))
			})
		})
	})

	Describe("ActualLRPGroupsByCellID", func() {
		Context("when there are /instance and /evacuating LRPs", func() {
			BeforeEach(func() {
				testHelper.SetRawActualLRP(baseLRP)
				testHelper.SetRawActualLRP(otherIndexLRP)
				testHelper.SetRawActualLRP(otherDomainLRP)
				testHelper.SetRawEvacuatingActualLRP(otherDomainLRP, noExpirationTTL)
				testHelper.SetRawEvacuatingActualLRP(yetAnotherIndexLRP, noExpirationTTL)
				testHelper.SetRawActualLRP(otherCellIDLRP)
			})

			It("returns both /instance and /evacuting actual lrps for the requested cell id", func() {
				actualLRPGroups, err := lrpBBS.ActualLRPGroupsByCellID(logger, cellID)
				Expect(err).NotTo(HaveOccurred())
				Expect(actualLRPGroups).To(ConsistOf(
					models.ActualLRPGroup{Instance: &baseLRP, Evacuating: nil},
					models.ActualLRPGroup{Instance: &otherIndexLRP, Evacuating: nil},
					models.ActualLRPGroup{Instance: &otherDomainLRP, Evacuating: &otherDomainLRP},
					models.ActualLRPGroup{Instance: nil, Evacuating: &yetAnotherIndexLRP},
				))
			})
		})

		Context("when there are no LRPs", func() {
			BeforeEach(func() {
				// leave some intermediate directories in the store
				testHelper.SetRawActualLRP(baseLRP)
				err := lrpBBS.RemoveActualLRP(logger, baseLRPKey, baseLRPInstanceKey)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns an empty list", func() {
				actualLRPGroups, err := lrpBBS.ActualLRPGroupsByCellID(logger, cellID)
				Expect(err).NotTo(HaveOccurred())
				Expect(actualLRPGroups).NotTo(BeNil())
				Expect(actualLRPGroups).To(BeEmpty())
			})
		})

		Context("when the root node exists with no child nodes", func() {
			BeforeEach(func() {
				testHelper.SetRawActualLRP(baseLRP)

				processGuid := baseLRP.ActualLRPKey.ProcessGuid
				err := etcdClient.Delete(shared.ActualLRPProcessDir(processGuid))
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns an empty list", func() {
				actualLRPGroups, err := lrpBBS.ActualLRPGroupsByCellID(logger, cellID)
				Expect(err).NotTo(HaveOccurred())
				Expect(actualLRPGroups).NotTo(BeNil())
				Expect(actualLRPGroups).To(BeEmpty())
			})
		})

		Context("when given an empty cell id", func() {
			It("returns an error", func() {
				_, err := lrpBBS.ActualLRPGroupsByCellID(logger, "")
				Expect(err).To(Equal(bbserrors.ErrNoCellID))
			})
		})

		Context("when there is invalid data", func() {
			BeforeEach(func() {
				testHelper.CreateValidActualLRP("some-guid", 0)
				testHelper.CreateMalformedActualLRP("some-other-guid", 0)
				testHelper.CreateValidActualLRP("some-third-guid", 0)
			})

			It("errors", func() {
				_, err := lrpBBS.ActualLRPGroupsByCellID(logger, cellID)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("ActualLRPGroupByProcessGuidAndIndex", func() {
		Context("when there is an /instance entry", func() {
			BeforeEach(func() {
				testHelper.SetRawActualLRP(baseLRP)
			})

			It("returns the /instance entry", func() {
				returnedLRPGroup, returnedErr := lrpBBS.ActualLRPGroupByProcessGuidAndIndex(logger, baseProcessGuid, baseIndex)
				Expect(returnedErr).NotTo(HaveOccurred())
				Expect(returnedLRPGroup).To(Equal(models.ActualLRPGroup{
					Instance:   &baseLRP,
					Evacuating: nil,
				}))

			})

			Context("when there is also an /evacuating entry", func() {
				BeforeEach(func() {
					testHelper.SetRawEvacuatingActualLRP(evacuatingLRP, noExpirationTTL)
				})

				It("returns both the /instance LRPs and /evacuating LRPs the group", func() {
					returnedLRPGroup, returnedErr := lrpBBS.ActualLRPGroupByProcessGuidAndIndex(logger, baseProcessGuid, baseIndex)
					Expect(returnedErr).NotTo(HaveOccurred())
					Expect(returnedLRPGroup).To(Equal(models.ActualLRPGroup{
						Instance:   &baseLRP,
						Evacuating: &evacuatingLRP,
					}))

				})
			})
		})

		Context("when there is only an /evacuating entry", func() {
			BeforeEach(func() {
				testHelper.SetRawEvacuatingActualLRP(evacuatingLRP, noExpirationTTL)
			})

			It("returns an ErrStoreResourceNotFound", func() {
				returnedLRPGroup, returnedErr := lrpBBS.ActualLRPGroupByProcessGuidAndIndex(logger, baseProcessGuid, baseIndex)
				Expect(returnedErr).NotTo(HaveOccurred())
				Expect(returnedLRPGroup).To(Equal(models.ActualLRPGroup{
					Instance:   nil,
					Evacuating: &evacuatingLRP,
				}))

			})
		})

		Context("when there are no entries", func() {
			It("returns an ErrStoreResourceNotFound", func() {
				_, returnedErr := lrpBBS.ActualLRPGroupByProcessGuidAndIndex(logger, baseProcessGuid, baseIndex)
				Expect(returnedErr).To(Equal(bbserrors.ErrStoreResourceNotFound))
			})
		})

		Context("when given an empty process guid", func() {
			It("returns an error", func() {
				_, returnedErr := lrpBBS.ActualLRPGroupByProcessGuidAndIndex(logger, "", baseIndex)
				Expect(returnedErr).To(Equal(bbserrors.ErrNoProcessGuid))
			})
		})

		Context("when there is an index entry without /instance or /evacuating", func() {
			BeforeEach(func() {
				testHelper.SetRawActualLRP(baseLRP)
				err := etcdClient.Delete(shared.ActualLRPSchemaPath(baseLRP.ProcessGuid, baseLRP.Index))
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns an ErrStoreResourceNotFound", func() {
				_, returnedErr := lrpBBS.ActualLRPGroupByProcessGuidAndIndex(logger, baseProcessGuid, baseIndex)
				Expect(returnedErr).To(Equal(bbserrors.ErrStoreResourceNotFound))
			})
		})
	})

	Describe("EvacuatingActualLRPByProcessGuidAndIndex", func() {
		Context("when there is both an /instance and an /evacuating entry", func() {
			BeforeEach(func() {
				testHelper.SetRawActualLRP(baseLRP)
				testHelper.SetRawEvacuatingActualLRP(evacuatingLRP, noExpirationTTL)
			})

			It("returns the /evacuating entry", func() {
				returnedLRP, returnedErr := lrpBBS.EvacuatingActualLRPByProcessGuidAndIndex(logger, baseProcessGuid, baseIndex)
				Expect(returnedErr).NotTo(HaveOccurred())
				Expect(returnedLRP).To(Equal(evacuatingLRP))
			})
		})

		Context("when there is only an /instance entry", func() {
			BeforeEach(func() {
				testHelper.SetRawActualLRP(baseLRP)
			})

			It("returns ErrStoreResourceNotFound", func() {
				_, returnedErr := lrpBBS.EvacuatingActualLRPByProcessGuidAndIndex(logger, baseProcessGuid, baseIndex)
				Expect(returnedErr).To(Equal(bbserrors.ErrStoreResourceNotFound))
			})
		})

		Context("when there is only an /evacuating entry", func() {
			BeforeEach(func() {
				testHelper.SetRawEvacuatingActualLRP(evacuatingLRP, noExpirationTTL)
			})

			It("returns the /evacuating entry", func() {
				returnedLRP, returnedErr := lrpBBS.EvacuatingActualLRPByProcessGuidAndIndex(logger, baseProcessGuid, baseIndex)
				Expect(returnedErr).NotTo(HaveOccurred())
				Expect(returnedLRP).To(Equal(evacuatingLRP))
			})
		})

		Context("when there are no entries", func() {
			It("returns ErrStoreResourceNotFound", func() {
				_, returnedErr := lrpBBS.EvacuatingActualLRPByProcessGuidAndIndex(logger, baseProcessGuid, baseIndex)
				Expect(returnedErr).To(Equal(bbserrors.ErrStoreResourceNotFound))
			})
		})

		Context("when given an empty process guid", func() {
			It("returns an error", func() {
				_, returnedErr := lrpBBS.EvacuatingActualLRPByProcessGuidAndIndex(logger, "", baseIndex)
				Expect(returnedErr).To(Equal(bbserrors.ErrNoProcessGuid))
			})
		})
	})
})
