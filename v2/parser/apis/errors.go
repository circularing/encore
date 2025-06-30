package apis

import (
	"github.com/circularing/encore/pkg/errors"
)

var (
	errRange = errors.Range(
		"parser/apis",
		"",
	)

	errUnexpectedDirective = errRange.Newf(
		"Invalid directive",
		"Unexpected directive %q on function declaration.",
	)
)
