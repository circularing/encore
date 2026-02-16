package natspubsub

import (
	"go/ast"
	"testing"

	"encr.dev/v2/parser/apis/directive"
)

func TestParseNATS_Valid(t *testing.T) {
	d := &directive.Directive{
		Name:    "nats",
		Options: []directive.Field{{Value: "orders.created"}},
		Fields: []directive.Field{
			{Key: "mode", Value: "at-least-once"},
			{Key: "ackwait", Value: "30s"},
			{Key: "maxinflight", Value: "64"},
			{Key: "queue", Value: "orders-workers"},
			{Key: "stream", Value: "orders_events"},
			{Key: "subjects", Value: "orders.created,orders.updated"},
		},
	}
	decl := handlerDecl(&ast.SelectorExpr{X: ast.NewIdent("context"), Sel: ast.NewIdent("Context")}, &ast.StarExpr{X: ast.NewIdent("OrderCreated")})

	if err := parsePubSub(d, decl); err != nil {
		t.Fatalf("parsePubSub returned error: %v", err)
	}
}

func TestParseNATS_InvalidSignature(t *testing.T) {
	d := &directive.Directive{
		Name:    "nats",
		Options: []directive.Field{{Value: "orders.created"}},
	}
	decl := handlerDecl(ast.NewIdent("int"), ast.NewIdent("OrderCreated"))

	if err := parsePubSub(d, decl); err == nil {
		t.Fatal("expected parsePubSub to reject invalid signature")
	}
}

func TestParseNATS_InvalidFields(t *testing.T) {
	decl := handlerDecl(&ast.SelectorExpr{X: ast.NewIdent("context"), Sel: ast.NewIdent("Context")}, &ast.StarExpr{X: ast.NewIdent("OrderCreated")})

	tests := []struct {
		name   string
		fields []directive.Field
	}{
		{name: "unknown field", fields: []directive.Field{{Key: "foo", Value: "bar"}}},
		{name: "bad mode", fields: []directive.Field{{Key: "mode", Value: "exactly-once"}}},
		{name: "bad ackwait", fields: []directive.Field{{Key: "ackwait", Value: "zzz"}}},
		{name: "bad maxinflight", fields: []directive.Field{{Key: "maxinflight", Value: "0"}}},
		{name: "empty queue", fields: []directive.Field{{Key: "queue", Value: " "}}},
		{name: "bad subjects", fields: []directive.Field{{Key: "subjects", Value: "orders/created"}}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := &directive.Directive{
				Name:    "nats",
				Options: []directive.Field{{Value: "orders.created"}},
				Fields:  tc.fields,
			}
			if err := parsePubSub(d, decl); err == nil {
				t.Fatalf("expected parsePubSub to reject %s", tc.name)
			}
		})
	}
}

func TestValidateNATSSubject(t *testing.T) {
	cases := []struct {
		subject string
		ok      bool
	}{
		{subject: "orders.created", ok: true},
		{subject: "orders.*", ok: true},
		{subject: "orders.>", ok: true},
		{subject: "orders..created", ok: false},
		{subject: "orders.>.created", ok: false},
		{subject: "orders.crea*ted", ok: false},
		{subject: "", ok: false},
	}

	for _, tc := range cases {
		err := validateNATSSubject(tc.subject)
		if tc.ok && err != nil {
			t.Errorf("expected subject %q to be valid, got error: %v", tc.subject, err)
		}
		if !tc.ok && err == nil {
			t.Errorf("expected subject %q to be invalid", tc.subject)
		}
	}
}

func handlerDecl(firstParam, secondParam ast.Expr) *ast.FuncDecl {
	return &ast.FuncDecl{
		Type: &ast.FuncType{
			Params:  &ast.FieldList{List: []*ast.Field{{Type: firstParam}, {Type: secondParam}}},
			Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("error")}}},
		},
	}
}
