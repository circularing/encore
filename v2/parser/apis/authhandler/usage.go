package authhandler

import (
	"github.com/circularing/encore/pkg/errors"
	"github.com/circularing/encore/v2/parser/resource/usage"
)

func ResolveAuthHandlerUsage(data usage.ResolveData, handler *AuthHandler) usage.Usage {
	switch expr := data.Expr.(type) {
	case *usage.FuncCall:
		if expr.DeclaredIn().Pkg != handler.Package() {
			data.Errs.Add(
				errCannotCallFromAnotherPackage.
					AtGoNode(expr, errors.AsError("called here")).
					AtGoNode(handler.Decl.AST.Name, errors.AsHelp("auth handler defined here")),
			)
		}
	default:
		data.Errs.Add(
			errInvalidReference.
				AtGoNode(expr, errors.AsError("referenced here")).
				AtGoNode(handler.Decl.AST.Name, errors.AsHelp("auth handler defined here")),
		)
	}

	return nil
}
