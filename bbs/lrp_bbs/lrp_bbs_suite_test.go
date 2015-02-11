package lrp_bbs_test

import (
	"encoding/json"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/domain_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/lrp_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/services_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	cbfakes "github.com/cloudfoundry-incubator/runtime-schema/cb/fakes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
	"github.com/cloudfoundry/storeadapter/storerunner/etcdstorerunner"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/clock/fakeclock"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"

	"testing"
	"time"
)

var etcdRunner *etcdstorerunner.ETCDClusterRunner
var etcdClient storeadapter.StoreAdapter
var bbs *lrp_bbs.LRPBBS
var domainBBS *domain_bbs.DomainBBS
var clock *AdvancingFakeClock
var fakeCellClient *cbfakes.FakeCellClient
var fakeAuctioneerClient *cbfakes.FakeAuctioneerClient

var logger *lagertest.TestLogger

func TestLRPBbs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Long Running Process BBS Suite")
}

var _ = BeforeSuite(func() {
	etcdRunner = etcdstorerunner.NewETCDClusterRunner(5001+config.GinkgoConfig.ParallelNode, 1)
	etcdClient = etcdRunner.RetryableAdapter()

	etcdRunner.Start()
})

var _ = AfterSuite(func() {
	etcdClient.Disconnect()
	etcdRunner.Stop()
})

var _ = BeforeEach(func() {
	etcdRunner.Reset()

	fakeCellClient = &cbfakes.FakeCellClient{}
	fakeAuctioneerClient = &cbfakes.FakeAuctioneerClient{}
	clock = &AdvancingFakeClock{
		FakeClock: fakeclock.NewFakeClock(time.Unix(0, 1138)),
	}

	logger = lagertest.NewTestLogger("test")

	servicesBBS := services_bbs.New(etcdClient, clock, lagertest.NewTestLogger("test"))

	bbs = lrp_bbs.New(
		etcdClient,
		clock,
		fakeCellClient,
		fakeAuctioneerClient,
		servicesBBS,
	)

	domainBBS = domain_bbs.New(etcdClient, lagertest.NewTestLogger("test"))
})

func registerCell(cell models.CellPresence) {
	jsonBytes, err := models.ToJSON(cell)
	Ω(err).ShouldNot(HaveOccurred())

	etcdClient.Create(storeadapter.StoreNode{
		Key:   shared.CellSchemaPath(cell.CellID),
		Value: jsonBytes,
	})
}

func registerAuctioneer(auctioneer models.AuctioneerPresence) {
	jsonBytes, err := models.ToJSON(auctioneer)
	Ω(err).ShouldNot(HaveOccurred())

	etcdClient.Create(storeadapter.StoreNode{
		Key:   shared.LockSchemaPath("auctioneer_lock"),
		Value: jsonBytes,
	})
}

func claimDesireLRPByIndex(d models.DesiredLRP, index int, containerKey models.ActualLRPContainerKey, logger lager.Logger) {
	unclaimed, err := bbs.ActualLRPByProcessGuidAndIndex(d.ProcessGuid, index)
	Ω(err).ShouldNot(HaveOccurred())

	err = bbs.ClaimActualLRP(logger, unclaimed.ActualLRPKey, containerKey)
	Ω(err).ShouldNot(HaveOccurred())
}

func setRawActualLRP(lrp models.ActualLRP) {
	value, err := json.Marshal(lrp) // do NOT use models.ToJSON; don't want validations
	Ω(err).ShouldNot(HaveOccurred())

	err = etcdClient.SetMulti([]storeadapter.StoreNode{
		{
			Key:   shared.ActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index),
			Value: value,
		},
	})

	Ω(err).ShouldNot(HaveOccurred())
}

func setRawEvacuatingActualLRP(lrp models.ActualLRP, ttlInSeconds uint64) {
	value, err := json.Marshal(lrp) // do NOT use models.ToJSON; don't want validations
	Ω(err).ShouldNot(HaveOccurred())

	err = etcdClient.SetMulti([]storeadapter.StoreNode{
		{
			Key:   shared.EvacuatingActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index),
			Value: value,
			TTL:   ttlInSeconds,
		},
	})

	Ω(err).ShouldNot(HaveOccurred())
}

func setRawDesiredLRP(d models.DesiredLRP) {
	value, err := json.Marshal(d) // do NOT use models.ToJSON; don't want validations
	Ω(err).ShouldNot(HaveOccurred())

	err = etcdClient.SetMulti([]storeadapter.StoreNode{
		{
			Key:   shared.DesiredLRPSchemaPath(d),
			Value: value,
		},
	})

	Ω(err).ShouldNot(HaveOccurred())
}

func deleteActualLRP(key models.ActualLRPKey) {
	err := etcdClient.Delete(shared.ActualLRPSchemaPath(key.ProcessGuid, key.Index))
	Ω(err).ShouldNot(HaveOccurred())
}

func deleteEvacuatingActualLRP(key models.ActualLRPKey) {
	err := etcdClient.Delete(shared.EvacuatingActualLRPSchemaPath(key.ProcessGuid, key.Index))
	Ω(err).ShouldNot(HaveOccurred())
}

func createRawDomain(domain string) {
	err := domainBBS.UpsertDomain(domain, 0)
	Ω(err).ShouldNot(HaveOccurred())
}

func getInstanceActualLRP(lrpKey models.ActualLRPKey) (models.ActualLRP, error) {
	node, err := etcdClient.Get(shared.ActualLRPSchemaPath(lrpKey.ProcessGuid, lrpKey.Index))
	if err == storeadapter.ErrorKeyNotFound {
		return models.ActualLRP{}, bbserrors.ErrStoreResourceNotFound
	}
	Ω(err).ShouldNot(HaveOccurred())

	var lrp models.ActualLRP
	err = models.FromJSON(node.Value, &lrp)
	Ω(err).ShouldNot(HaveOccurred())

	return lrp, nil
}

func getEvacuatingActualLRP(lrpKey models.ActualLRPKey) (models.ActualLRP, uint64, error) {
	node, err := etcdClient.Get(shared.EvacuatingActualLRPSchemaPath(lrpKey.ProcessGuid, lrpKey.Index))
	if err == storeadapter.ErrorKeyNotFound {
		return models.ActualLRP{}, 0, bbserrors.ErrStoreResourceNotFound
	}
	Ω(err).ShouldNot(HaveOccurred())

	var lrp models.ActualLRP
	err = models.FromJSON(node.Value, &lrp)
	Ω(err).ShouldNot(HaveOccurred())

	return lrp, node.TTL, nil
}

func defaultNetInfo() models.ActualLRPNetInfo {
	return models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{{ContainerPort: 1234, HostPort: 5678}})
}
