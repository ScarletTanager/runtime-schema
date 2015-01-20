package domain_bbs

import (
	"path"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
)

type DomainBBS struct {
	store  storeadapter.StoreAdapter
	logger lager.Logger
}

func New(
	store storeadapter.StoreAdapter,
	logger lager.Logger,
) *DomainBBS {
	return &DomainBBS{
		store:  store,
		logger: logger,
	}
}

func (bbs *DomainBBS) UpsertDomain(domain string, ttlInSeconds int) error {
	var validationError models.ValidationError

	if domain == "" {
		validationError = validationError.Append(models.ErrInvalidParameter{"domain"})
	}

	if ttlInSeconds < 0 {
		validationError = validationError.Append(models.ErrInvalidParameter{"ttlInSeconds"})
	}

	if !validationError.Empty() {
		return validationError
	}

	return shared.ConvertStoreError(bbs.store.SetMulti([]storeadapter.StoreNode{
		{
			Key: shared.DomainSchemaPath(domain),
			TTL: uint64(ttlInSeconds),
		},
	}))
}

func (bbs *DomainBBS) Domains() ([]string, error) {
	node, err := bbs.store.ListRecursively(shared.DomainSchemaRoot)
	if err != nil && err != storeadapter.ErrorKeyNotFound {
		return nil, shared.ConvertStoreError(err)
	}

	domains := make([]string, 0, len(node.ChildNodes))

	for _, node := range node.ChildNodes {
		domains = append(domains, path.Base(node.Key))
	}

	return domains, nil
}
