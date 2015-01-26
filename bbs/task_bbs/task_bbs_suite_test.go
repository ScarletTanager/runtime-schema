package task_bbs_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/services_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/task_bbs"
	cbfakes "github.com/cloudfoundry-incubator/runtime-schema/cb/fakes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
	"github.com/cloudfoundry/storeadapter/storerunner/etcdstorerunner"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/clock/fakeclock"
	"github.com/pivotal-golang/lager/lagertest"

	"testing"
	"time"
)

var etcdRunner *etcdstorerunner.ETCDClusterRunner
var etcdClient storeadapter.StoreAdapter
var logger *lagertest.TestLogger
var servicesBBS *services_bbs.ServicesBBS
var fakeTaskClient *cbfakes.FakeTaskClient
var fakeAuctioneerClient *cbfakes.FakeAuctioneerClient
var clock *fakeclock.FakeClock
var bbs *task_bbs.TaskBBS

var dummyAction = &models.RunAction{
	Path: "cat",
	Args: []string{"/tmp/file"},
}

func TestTaskBbs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Task BBS Suite")
}

var _ = BeforeSuite(func() {
	etcdRunner = etcdstorerunner.NewETCDClusterRunner(5001+config.GinkgoConfig.ParallelNode, 1)
	etcdClient = etcdRunner.Adapter()
})

var _ = AfterSuite(func() {
	etcdRunner.Stop()
})

var _ = BeforeEach(func() {
	etcdRunner.Stop()
	etcdRunner.Start()

	logger = lagertest.NewTestLogger("test")

	fakeTaskClient = new(cbfakes.FakeTaskClient)
	fakeAuctioneerClient = new(cbfakes.FakeAuctioneerClient)
	clock = fakeclock.NewFakeClock(time.Unix(1238, 0))
	servicesBBS = services_bbs.New(etcdClient, clock, logger)
	bbs = task_bbs.New(etcdClient, clock, fakeTaskClient, fakeAuctioneerClient, servicesBBS)
})

func registerAuctioneer(auctioneer models.AuctioneerPresence) {
	jsonBytes, err := models.ToJSON(auctioneer)
	Ω(err).ShouldNot(HaveOccurred())

	etcdClient.Create(storeadapter.StoreNode{
		Key:   shared.LockSchemaPath("auctioneer_lock"),
		Value: jsonBytes,
	})
}
