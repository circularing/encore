package natspubsub

import (
	"fmt"
	"go/ast"

	"github.com/circularing/encore/v2/parser/apis/directive"
)

func init() {
	directive.RegisterDirectiveParser("pubsub", parsePubSub)
}

// parsePubSub handles "//encore:pubsub <subject>" above a FuncDecl.
func parsePubSub(d *directive.Directive, decl *ast.FuncDecl) error {
	fmt.Println("parsePubSub", d, decl)
	// If decl is nil, we’re just parsing comments—skip until codegen.
	// if decl == nil {
	// 	return nil
	// }

	// Now decl is non-nil: validate and annotate.
	if len(d.Options) != 1 {
		return fmt.Errorf("pubsub directive requires exactly one subject argument, got %d", len(d.Options))
	}
	subject := d.Options[0].Value

	// Validate handler signature: func(context.Context, *T) error
	if len(decl.Type.Params.List) != 2 {
		return fmt.Errorf("pubsub: handler must have two parameters (ctx, *Event)")
	}
	// check return type is exactly ( error )
	if decl.Type.Results == nil || len(decl.Type.Results.List) != 1 {
		return fmt.Errorf("pubsub: handler must return exactly one value (error)")
	}
	if ident, ok := decl.Type.Results.List[0].Type.(*ast.Ident); !ok || ident.Name != "error" {
		return fmt.Errorf("pubsub: handler must return error")
	}

	// All good—mark it for the generator to pick up.
	decl.Doc.List = append(decl.Doc.List, &ast.Comment{
		Text: "// @encore:pubsub:" + subject,
	})
	return nil
}
