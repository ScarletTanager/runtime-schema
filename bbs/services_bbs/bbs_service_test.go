package services_bbs_test

import (
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager/lagertest"
	"github.com/tedsuo/ifrit"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/lock_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/services_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/pivotal-golang/clock/fakeclock"
)

var _ = Describe("BBS Presence", func() {
	var clock *fakeclock.FakeClock
	var bbs *services_bbs.ServicesBBS
	var logger *lagertest.TestLogger

	BeforeEach(func() {
		clock = fakeclock.NewFakeClock(time.Now())
		logger = lagertest.NewTestLogger("test")
		bbs = services_bbs.New(consulSession, clock, logger)
	})

	Describe("MasterURL", func() {
		Context("when able to get a master bbs presence", func() {
			var heartbeater ifrit.Process
			var bbsPresence models.BBSPresence

			JustBeforeEach(func() {
				lockBbs := lock_bbs.New(consulSession, clock, logger)
				bbsLock, err := lockBbs.NewBBSMasterLock(bbsPresence, 100*time.Millisecond)
				Expect(err).NotTo(HaveOccurred())
				heartbeater = ifrit.Invoke(bbsLock)
			})

			AfterEach(func() {
				heartbeater.Signal(os.Interrupt)
				Eventually(heartbeater.Wait()).Should(Receive(BeNil()))
			})

			Context("when the master bbs URL is present", func() {
				BeforeEach(func() {
					bbsPresence = models.NewBBSPresence("a-bbs-id", "https://database-z1-0.database.consul.cf.internal:7085")
				})

				It("returns the URL", func() {
					url, err := bbs.MasterURL()
					Expect(err).NotTo(HaveOccurred())
					Expect(url).To(Equal(bbsPresence.URL))
				})
			})
		})

		Context("when unable to get any bbs presences", func() {
			It("returns ErrServiceUnavailable", func() {
				_, err := bbs.MasterURL()
				Expect(err).To(Equal(bbserrors.ErrServiceUnavailable))
			})
		})
	})
})
