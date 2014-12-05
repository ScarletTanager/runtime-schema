package fake_bbs_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs"
	. "github.com/cloudfoundry-incubator/runtime-schema/bbs/fake_bbs"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("FakeBbs", func() {
	It("should provide fakes that satisfy the interfaces", func() {
		var repBBS bbs.RepBBS
		repBBS = &FakeRepBBS{}
		Ω(repBBS).ShouldNot(BeNil())

		var convergerBBS bbs.ConvergerBBS
		convergerBBS = new(FakeConvergerBBS)
		Ω(convergerBBS).ShouldNot(BeNil())

		var auctioneerBBS bbs.AuctioneerBBS
		auctioneerBBS = new(FakeAuctioneerBBS)
		Ω(auctioneerBBS).ShouldNot(BeNil())

		var metricsBBS bbs.MetricsBBS
		metricsBBS = new(FakeMetricsBBS)
		Ω(metricsBBS).ShouldNot(BeNil())

		var routeEmitterBBS bbs.RouteEmitterBBS
		routeEmitterBBS = new(FakeRouteEmitterBBS)
		Ω(routeEmitterBBS).ShouldNot(BeNil())

		var nsyncBBS bbs.NsyncBBS
		nsyncBBS = &FakeNsyncBBS{}
		Ω(nsyncBBS).ShouldNot(BeNil())
	})
})
