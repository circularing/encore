package codegen

import "github.com/circularing/encore/pkg/errors"

var (
	errRange = errors.Range(
		"codegen",
		"",
	)

	errRender = errRange.New(
		"Failed to render codegen",
		"Generated code could not be parsed.",
		errors.MarkAsInternalError(),
	)
)
