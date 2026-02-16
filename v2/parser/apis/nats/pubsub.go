package nats

import (
	"go/ast"
	"go/token"

	"encr.dev/v2/internals/perr"
	"encr.dev/v2/internals/pkginfo"
	"encr.dev/v2/parser/apis/directive"
	"encr.dev/v2/parser/resource"
	"encr.dev/v2/parser/resource/resourceparser"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
)

// Subscription models a NATS subscription resource.
type Subscription struct {
	Name    string // Go fn name, e.g. HandleOrderCreated
	Subject string // NATS subject, e.g. "orders.created"
	File    *pkginfo.File
	Decl    *ast.FuncDecl
	Doc     string
}

var Parser = &resourceparser.Parser{
	Name:               "PubSub",
	InterestingImports: resourceparser.RunAlways,
	Run:                runParser,
}

func runParser(p *resourceparser.Pass) {
	for _, file := range p.Pkg.Files {
		for _, decl := range file.AST().Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Doc == nil {
				continue
			}
			dir, doc, ok := directive.Parse(p.Errs, fn)
			if !ok || dir == nil || dir.Name != "nats" {
				continue
			}

			// Delegate to our pubsub.Parse
			sub := Parse(ParseData{
				Errs: p.Errs,
				File: file,
				Func: fn,
				Dir:  dir,
				Doc:  doc,
			})
			if sub == nil {
				continue
			}
			p.RegisterResource(sub)
			p.AddNamedBind(file, fn.Name, sub)
		}
	}
}

// ParseData carries everything needed to parse a pubsub directive.
type ParseData struct {
	Errs *perr.List
	File *pkginfo.File
	Func *ast.FuncDecl
	Dir  *directive.Directive
	Doc  string
}

// Parse validates and builds a Subscription or returns nil on error.
func Parse(d ParseData) *Subscription {
	if d.Errs == nil || d.Func == nil || d.Func.Type == nil || d.Dir == nil {
		return nil
	}

	// Exactly one bare Option: the subject name
	opts := d.Dir.Options
	if len(opts) != 1 {
		d.Errs.Addf(d.Func.Pos(), "nats directive requires exactly one subject option")
		return nil
	}
	subject := opts[0].Value

	// Signature must be: func(context.Context, *T) error
	sig := d.Func.Type
	if sig.Params == nil || len(sig.Params.List) != 2 {
		d.Errs.Addf(d.Func.Pos(), "nats handler must have two parameters (context.Context, *Event)")
		return nil
	}
	if !isContextParam(sig.Params.List[0].Type) {
		d.Errs.Addf(d.Func.Pos(), "nats first handler parameter must be context.Context")
		return nil
	}
	if _, ok := sig.Params.List[1].Type.(*ast.StarExpr); !ok {
		d.Errs.Addf(d.Func.Pos(), "nats second handler parameter must be a pointer type (*Event)")
		return nil
	}
	if sig.Results == nil || len(sig.Results.List) != 1 {
		d.Errs.Addf(d.Func.Pos(), "nats handler must return exactly one value (error)")
		return nil
	}
	if ident, ok := sig.Results.List[0].Type.(*ast.Ident); !ok || ident.Name != "error" {
		d.Errs.Addf(d.Func.Pos(), "nats handler must return error")
		return nil
	}

	// All good—return our Subscription resource
	return &Subscription{
		Name:    d.Func.Name.Name,
		Subject: subject,
		File:    d.File,
		Decl:    d.Func,
		Doc:     d.Doc,
	}
}

func isContextParam(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return pkg.Name == "context" && sel.Sel != nil && sel.Sel.Name == "Context"
}

// Implement resource.Resource:
func (s *Subscription) Kind() resource.Kind       { return resource.PubSubSubscription }
func (s *Subscription) Package() *pkginfo.Package { return s.File.Pkg }
func (s *Subscription) Pos() token.Pos            { return s.Decl.Pos() }
func (s *Subscription) End() token.Pos            { return s.Decl.End() }
func (s *Subscription) SortKey() string           { return s.File.Pkg.ImportPath.String() + "." + s.Name }

// Generate is a placeholder for future NATS code generation wiring.
func (s *Subscription) Generate(_ *generator.Generator) {}
