package directive

import (
	"context"
	"go/ast"
	"go/token"
	"regexp"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/google/go-cmp/cmp/cmpopts"

	"encr.dev/v2/internals/perr"
)

func TestParseDirective(t *testing.T) {
	testcases := []struct {
		desc     string
		line     string
		expected Directive
		wantErr  string
	}{
		{
			desc: "api public endpoint",
			line: "api public",
			expected: Directive{
				Name:    "api",
				Options: []Field{{Value: "public"}},
			},
		},
		{
			desc: "custom method",
			line: "api public method=FOO",
			expected: Directive{
				Name:    "api",
				Options: []Field{{Value: "public"}},
				Fields:  []Field{{Key: "method", Value: "FOO"}},
			},
		},
		{
			desc: "multiple methods",
			line: "api public raw method=GET,POST",
			expected: Directive{
				Name:    "api",
				Options: []Field{{Value: "public"}, {Value: "raw"}},
				Fields:  []Field{{Key: "method", Value: "GET,POST"}},
			},
		},
		{
			desc: "api with tags",
			line: "api public tag:foo method=FOO raw tag:bar",
			expected: Directive{
				Name:    "api",
				Options: []Field{{Value: "public"}, {Value: "raw"}},
				Fields:  []Field{{Key: "method", Value: "FOO"}},
				Tags:    []Field{{Value: "tag:foo"}, {Value: "tag:bar"}},
			},
		},
		{
			desc:    "api with duplicate tag",
			line:    "api public tag:foo tag:foo",
			wantErr: `(?m)The tag "tag:foo" is already defined on this declaration\.`,
		},
		{
			desc: "middleware",
			line: "middleware target=tag:foo,tag:bar",
			expected: Directive{
				Name:   "middleware",
				Fields: []Field{{Key: "target", Value: "tag:foo,tag:bar"}},
			},
		},
		{
			desc: "nats with subject",
			line: "nats orders.created",
			expected: Directive{
				Name:    "nats",
				Options: []Field{{Value: "orders.created"}},
			},
		},
		{
			desc: "nats wildcard subject",
			line: "nats orders.*",
			expected: Directive{
				Name:    "nats",
				Options: []Field{{Value: "orders.*"}},
			},
		},
		{
			desc:    "nats invalid subject character",
			line:    "nats orders/created",
			wantErr: `(?m)Invalid option name "orders/created"\.`,
		},
		{
			desc:    "nats duplicate subject option",
			line:    "nats orders.created orders.created",
			wantErr: `(?m)The option "orders.created" is already defined on this directive\.`,
		},
		{
			desc:    "middleware empty target",
			line:    "middleware target=",
			wantErr: `(?m)Directive fields must have a value\.`,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			c := qt.New(t)
			fs := token.NewFileSet()
			errs := perr.NewList(context.Background(), fs)
			dir, ok := parseOne(errs, 0, tc.line)
			if tc.wantErr != "" {
				re := regexp.MustCompile(tc.wantErr)
				if errStr := errs.FormatErrors(); !re.MatchString(errStr) {
					c.Fatalf("error did not match regexp %s: %s", tc.wantErr, errStr)
				}
			} else {
				c.Assert(ok, qt.IsTrue)

				cmp := qt.CmpEquals(
					cmpopts.EquateEmpty(),
					cmpopts.IgnoreUnexported(Field{}),
					cmpopts.IgnoreUnexported(Directive{}),
				)
				c.Assert(dir, cmp, tc.expected)
			}
		})
	}
}

func TestParse_StandardDirectiveKeepsDocText(t *testing.T) {
	c := qt.New(t)

	fs := token.NewFileSet()
	errList := perr.NewList(context.Background(), fs)

	decl := &ast.FuncDecl{
		Doc: &ast.CommentGroup{List: []*ast.Comment{
			{Text: "//encore:api public"},
			{Text: "// Hello from docs"},
		}},
	}

	dir, doc, ok := Parse(errList, decl)
	c.Assert(ok, qt.IsTrue)
	c.Assert(dir, qt.IsNotNil)
	c.Assert(dir.Name, qt.Equals, "api")
	c.Assert(doc, qt.Not(qt.Contains), "encore:api")
	c.Assert(doc, qt.Contains, "Hello from docs")
}
