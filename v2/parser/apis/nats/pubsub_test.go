package nats

import (
	"context"
	"go/ast"
	"go/token"
	"testing"

	"encr.dev/v2/internals/perr"
	"encr.dev/v2/parser/apis/directive"
)

func TestParse_ReportsErrors(t *testing.T) {
	mkErrs := func() *perr.List {
		return perr.NewList(context.Background(), token.NewFileSet())
	}

	t.Run("invalid option count", func(t *testing.T) {
		errs := mkErrs()
		sub := Parse(ParseData{
			Errs: errs,
			Func: handlerDecl(),
			Dir:  &directive.Directive{Name: "pubsub"},
		})
		if sub != nil {
			t.Fatal("expected nil subscription")
		}
		if errs.Len() == 0 {
			t.Fatal("expected parse error for missing subject option")
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		errs := mkErrs()
		fn := handlerDecl()
		fn.Type.Results = &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("string")}}}

		sub := Parse(ParseData{
			Errs: errs,
			Func: fn,
			Dir: &directive.Directive{
				Name:    "nats",
				Options: []directive.Field{{Value: "orders.created"}},
			},
		})
		if sub != nil {
			t.Fatal("expected nil subscription")
		}
		if errs.Len() == 0 {
			t.Fatal("expected parse error for invalid signature")
		}
	})
}

func TestParse_Valid(t *testing.T) {
	errs := perr.NewList(context.Background(), token.NewFileSet())
	fn := handlerDecl()
	fn.Name = ast.NewIdent("HandleOrderCreated")

	sub := Parse(ParseData{
		Errs: errs,
		Func: fn,
		Dir: &directive.Directive{
			Name:    "nats",
			Options: []directive.Field{{Value: "orders.created"}},
		},
		Doc: "handler docs",
	})
	if sub == nil {
		t.Fatal("expected subscription")
	}
	if errs.Len() != 0 {
		t.Fatalf("did not expect parse errors, got %d", errs.Len())
	}
	if sub.Subject != "orders.created" {
		t.Fatalf("unexpected subject %q", sub.Subject)
	}
	if sub.Name != "handle-order-created" {
		t.Fatalf("unexpected subscription name %q", sub.Name)
	}
	if sub.HandlerName != "HandleOrderCreated" {
		t.Fatalf("unexpected handler name %q", sub.HandlerName)
	}
}

func handlerDecl() *ast.FuncDecl {
	return &ast.FuncDecl{
		Type: &ast.FuncType{
			Params: &ast.FieldList{List: []*ast.Field{
				{Type: &ast.SelectorExpr{X: ast.NewIdent("context"), Sel: ast.NewIdent("Context")}},
				{Type: &ast.StarExpr{X: ast.NewIdent("OrderCreated")}},
			}},
			Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("error")}}},
		},
	}
}
