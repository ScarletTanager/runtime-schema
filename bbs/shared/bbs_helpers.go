package shared

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry/storeadapter"
)

func ConvertStoreError(originalErr error) error {
	switch originalErr {
	case storeadapter.ErrorKeyNotFound:
		return bbserrors.ErrStoreResourceNotFound
	case storeadapter.ErrorNodeIsDirectory:
		return bbserrors.ErrStoreExpectedNonCollectionRequest
	case storeadapter.ErrorNodeIsNotDirectory:
		return bbserrors.ErrStoreExpectedCollectionRequest
	case storeadapter.ErrorTimeout:
		return bbserrors.ErrStoreTimeout
	case storeadapter.ErrorInvalidFormat:
		return bbserrors.ErrStoreInvalidFormat
	case storeadapter.ErrorInvalidTTL:
		return bbserrors.ErrStoreInvalidTTL
	case storeadapter.ErrorKeyExists:
		return bbserrors.ErrStoreResourceExists
	case storeadapter.ErrorKeyComparisonFailed:
		return bbserrors.ErrStoreComparisonFailed
	default:
		return originalErr
	}
}
