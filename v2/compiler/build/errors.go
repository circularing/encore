package build

import (
	"github.com/circularing/encore/pkg/errors"
)

var (
	errRange = errors.Range("test", "", errors.WithRangeSize(20))

	ErrTestFailed = errRange.New("Test Failure", "One or more more tests failed.")
)
