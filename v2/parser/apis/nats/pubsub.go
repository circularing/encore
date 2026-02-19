package nats

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"
	"time"
	"unicode"

	"encr.dev/v2/internals/perr"
	"encr.dev/v2/internals/pkginfo"
	"encr.dev/v2/internals/schema"
	"encr.dev/v2/internals/schema/schemautil"
	"encr.dev/v2/parser/apis/directive"
	"encr.dev/v2/parser/resource"
	"encr.dev/v2/parser/resource/resourceparser"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
)

// Subscription models a NATS subscription resource.
type Subscription struct {
	Name        string // Unique subscription name, used at runtime (kebab-case).
	HandlerName string // Go fn name, e.g. HandleOrderCreated.
	Subject     string // NATS subject, e.g. "orders.created".
	File        *pkginfo.File
	Decl        *ast.FuncDecl
	Doc         string
	MessageType *schema.TypeDeclRef
	ReplyType   *schema.TypeDeclRef
	Cfg         SubscriptionConfig
	NATS        NATSConfig
}

type SubscriptionConfig struct {
	AckDeadline      time.Duration
	MessageRetention time.Duration
	MinRetryBackoff  time.Duration
	MaxRetryBackoff  time.Duration
	MaxRetries       int
	MaxConcurrency   int
}

type DeliveryMode string

const (
	ModeAtLeastOnce DeliveryMode = "at-least-once"
	ModeAtMostOnce  DeliveryMode = "at-most-once"
)

type NATSConfig struct {
	Mode           DeliveryMode
	AckWait        time.Duration
	MaxInflight    int
	MaxInflightSet bool
	QueueGroup     string
	StreamName     string
	StreamSubjects []string
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
	Errs   *perr.List
	File   *pkginfo.File
	Func   *ast.FuncDecl
	Dir    *directive.Directive
	Doc    string
	Schema *schema.Parser
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

	// Signature must be either:
	//   - func(context.Context, *Req) error
	//   - func(context.Context, *Req) (*Resp, error)
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
	if sig.Results == nil || (len(sig.Results.List) != 1 && len(sig.Results.List) != 2) {
		d.Errs.Addf(d.Func.Pos(), "nats handler must return error or (*Reply, error)")
		return nil
	}
	if len(sig.Results.List) == 1 {
		if ident, ok := sig.Results.List[0].Type.(*ast.Ident); !ok || ident.Name != "error" {
			d.Errs.Addf(d.Func.Pos(), "nats handler must return error")
			return nil
		}
	} else {
		if _, ok := sig.Results.List[0].Type.(*ast.StarExpr); !ok {
			d.Errs.Addf(d.Func.Pos(), "nats reply type must be a pointer type (*Reply)")
			return nil
		}
		if ident, ok := sig.Results.List[1].Type.(*ast.Ident); !ok || ident.Name != "error" {
			d.Errs.Addf(d.Func.Pos(), "nats handler second return value must be error")
			return nil
		}
	}
	if d.Func.Recv != nil {
		d.Errs.Addf(d.Func.Pos(), "nats handler must be a package-level function")
		return nil
	}

	var msgType, replyType *schema.TypeDeclRef
	if d.Schema != nil {
		var ok bool
		msgType, ok = schemautil.ResolveNamedStruct(d.Schema.ParseType(d.File, sig.Params.List[1].Type), true)
		if !ok {
			d.Errs.Addf(d.Func.Pos(), "nats second handler parameter must be pointer to a named struct type")
			return nil
		}
		if len(sig.Results.List) == 2 {
			replyType, ok = schemautil.ResolveNamedStruct(d.Schema.ParseType(d.File, sig.Results.List[0].Type), true)
			if !ok {
				d.Errs.Addf(d.Func.Pos(), "nats reply type must be pointer to a named struct type")
				return nil
			}
		}
	}

	cfg := parseConfig(d.Dir)

	// All good—return our Subscription resource
	return &Subscription{
		Name:        toKebab(d.Func.Name.Name),
		HandlerName: d.Func.Name.Name,
		Subject:     subject,
		File:        d.File,
		Decl:        d.Func,
		Doc:         d.Doc,
		MessageType: msgType,
		ReplyType:   replyType,
		Cfg:         cfg,
		NATS:        parseNATSConfig(d.Dir),
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

func parseConfig(dir *directive.Directive) SubscriptionConfig {
	cfg := SubscriptionConfig{
		AckDeadline:      30 * time.Second,
		MessageRetention: 7 * 24 * time.Hour,
		MinRetryBackoff:  10 * time.Second,
		MaxRetryBackoff:  10 * time.Minute,
		MaxRetries:       100,
		MaxConcurrency:   100,
	}

	if dir == nil {
		return cfg
	}

	if v := strings.TrimSpace(dir.Get("ackwait")); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			cfg.AckDeadline = d
		}
	}
	if v := strings.TrimSpace(dir.Get("maxinflight")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MaxConcurrency = n
		}
	}
	return cfg
}

func parseNATSConfig(dir *directive.Directive) NATSConfig {
	cfg := NATSConfig{
		Mode:        ModeAtLeastOnce,
		AckWait:     30 * time.Second,
		MaxInflight: 1,
	}
	if dir == nil {
		return cfg
	}

	if v := strings.TrimSpace(dir.Get("mode")); v == string(ModeAtMostOnce) {
		cfg.Mode = ModeAtMostOnce
	}
	if v := strings.TrimSpace(dir.Get("ackwait")); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			cfg.AckWait = d
		}
	}
	if v := strings.TrimSpace(dir.Get("maxinflight")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MaxInflight = n
			cfg.MaxInflightSet = true
		}
	}
	cfg.QueueGroup = strings.TrimSpace(dir.Get("queue"))
	cfg.StreamName = strings.TrimSpace(dir.Get("stream"))

	rawSubjects := strings.TrimSpace(dir.Get("subjects"))
	if rawSubjects != "" {
		parts := strings.Split(rawSubjects, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				cfg.StreamSubjects = append(cfg.StreamSubjects, p)
			}
		}
	}
	return cfg
}

func toKebab(name string) string {
	if name == "" {
		return "subscription"
	}
	var b strings.Builder
	lastWasDash := false
	for i, r := range name {
		if unicode.IsUpper(r) {
			if i > 0 && !lastWasDash {
				b.WriteByte('-')
			}
			b.WriteRune(unicode.ToLower(r))
			lastWasDash = false
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
			lastWasDash = false
			continue
		}
		if !lastWasDash {
			b.WriteByte('-')
			lastWasDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "subscription"
	}
	return out
}

// Implement resource.Resource:
func (s *Subscription) Kind() resource.Kind       { return resource.PubSubSubscription }
func (s *Subscription) Package() *pkginfo.Package { return s.File.Pkg }
func (s *Subscription) Pos() token.Pos            { return s.Decl.Pos() }
func (s *Subscription) End() token.Pos            { return s.Decl.End() }
func (s *Subscription) SortKey() string           { return s.File.Pkg.ImportPath.String() + "." + s.Name }

// Generate is a placeholder for future NATS code generation wiring.
func (s *Subscription) Generate(_ *generator.Generator) {}
