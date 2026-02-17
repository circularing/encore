package nats

import (
	"go/ast"
	"go/token"

	"github.com/circularing/encore/v2/internals/perr"
	"github.com/circularing/encore/v2/internals/pkginfo"
	"github.com/circularing/encore/v2/parser/apis/directive"
	"github.com/circularing/encore/v2/parser/resource"
	"github.com/circularing/encore/v2/parser/resource/resourceparser"
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
			if !ok || dir == nil || dir.Name != "pubsub" {
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
	// Exactly one bare Option: the subject name
	opts := d.Dir.Options
	if len(opts) != 1 {
		// d.Errs.Add(perr.Newf(d.Dir.Pos(), "pubsub directive requires exactly one subject, got %d", len(opts)))
		return nil
	}
	subject := opts[0].Value

	// Signature must be: func(context.Context, *T) error
	sig := d.Func.Type
	if len(sig.Params.List) != 2 || len(sig.Results.List) != 1 {
		// d.Errs.Add(perr.Newf(d.Func.Pos(), "pubsub handler must be: func(ctx context.Context, evt *T) error"))
		return nil
	}
	if ident, ok := sig.Results.List[0].Type.(*ast.Ident); !ok || ident.Name != "error" {
		// d.Errs.Add(perr.Newf(d.Func.Results.Pos(), "pubsub handler must return error"))
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

// Implement resource.Resource:
func (s *Subscription) Kind() resource.Kind       { return resource.PubSubTopic }
func (s *Subscription) Package() *pkginfo.Package { return s.File.Pkg }
func (s *Subscription) Pos() token.Pos            { return s.Decl.Pos() }
func (s *Subscription) End() token.Pos            { return s.Decl.End() }
func (s *Subscription) SortKey() string           { return s.File.Pkg.ImportPath.String() + "." + s.Name }

// Generate code (invoked later by Encore’s codegen) wiring up NATS:
func (s *Subscription) Generate(g *generator.Generator) {
	// same emitPubSub logic you already wrote…
	g.AddImport("time")
	g.AddImport("github.com/nats-io/nats.go")
	g.AddImport("custom/nats/pubsub") // your module path

	g.GenerateAllFiles()

	//	g.Printf(`
	//
	// // init wires %s into NATS subject %q
	//
	//	func init() {
	//	    client := pubsub.NewClient()
	//	    topic := pubsub.NewTopic[%s](
	//	        client,
	//	        %q,
	//	        pubsub.WithAtLeastOnce(),
	//	        pubsub.WithStreamConfig(nats.StreamConfig{
	//	            Name:      %q,
	//	            Subjects:  []string{%q},
	//	            Retention: nats.LimitsPolicy,
	//	            Storage:   nats.FileStorage,
	//	            MaxAge:    24 * time.Hour,
	//	            Replicas:  1,
	//	        }),
	//	        pubsub.WithSubscriptionOptions(30*time.Second, 20, %q),
	//	    )
	//	    go func() {
	//	        if err := topic.Subscribe(
	//	            %q,
	//	            pubsub.SubscriptionConfig[%s]{Handler: %s},
	//	        ); err != nil {
	//	            panic("nats subscribe failed: " + err.Error())
	//	        }
	//	    }()
	//	}
	//
	// `, s.Name, s.Subject,
	//
	//		s.Subject,
	//		"ENCORE_"+strings.ReplaceAll(s.Subject, ".", "_"),
	//		s.Subject,
	//		s.Subject,
	//		s.Name,
	//		s.Name,
	//	)
}
