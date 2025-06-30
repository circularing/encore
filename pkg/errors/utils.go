package errors

import (
	"go/ast"

	"github.com/circularing/encore/pkg/option"
)

// AtOptionalNode returns an error at the given node if it is present.
// Otherwise, it returns the error unchanged.
func AtOptionalNode[T ast.Node](err Template, opt option.Option[T]) Template {
	if node, ok := opt.Get(); ok {
		return err.AtGoNode(node)
	}
	return err
}
