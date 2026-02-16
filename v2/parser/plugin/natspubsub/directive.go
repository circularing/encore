package natspubsub

import (
	"fmt"
	"go/ast"
	"regexp"
	"strconv"
	"strings"
	"time"

	"encr.dev/v2/parser/apis/directive"
)

var natsSubjectRe = regexp.MustCompile(`^[A-Za-z0-9._*>-]+$`)

func init() {
	directive.RegisterDirectiveParser("nats", parsePubSub)
}

// parsePubSub handles "//encore:nats <subject> [field=value ...]" above a FuncDecl.
func parsePubSub(d *directive.Directive, decl *ast.FuncDecl) error {
	if d == nil {
		return fmt.Errorf("nats directive is nil")
	}
	if decl == nil || decl.Type == nil || decl.Type.Params == nil {
		return fmt.Errorf("nats directive must annotate a function declaration")
	}
	if len(d.Tags) > 0 {
		return fmt.Errorf("nats directive does not support tags")
	}
	if len(d.Options) != 1 {
		return fmt.Errorf("nats directive requires exactly one subject argument, got %d", len(d.Options))
	}
	if err := validateNATSSubject(d.Options[0].Value); err != nil {
		return fmt.Errorf("invalid nats subject %q: %w", d.Options[0].Value, err)
	}
	if err := validateDirectiveFields(d.Fields); err != nil {
		return err
	}

	// Validate handler signature: func(context.Context, *T) error
	if len(decl.Type.Params.List) != 2 {
		return fmt.Errorf("nats: handler must have two parameters (context.Context, *Event)")
	}
	if !isContextParam(decl.Type.Params.List[0].Type) {
		return fmt.Errorf("nats: first handler parameter must be context.Context")
	}
	if _, ok := decl.Type.Params.List[1].Type.(*ast.StarExpr); !ok {
		return fmt.Errorf("nats: second handler parameter must be a pointer type (*Event)")
	}
	if decl.Type.Results == nil || len(decl.Type.Results.List) != 1 {
		return fmt.Errorf("nats: handler must return exactly one value (error)")
	}
	if ident, ok := decl.Type.Results.List[0].Type.(*ast.Ident); !ok || ident.Name != "error" {
		return fmt.Errorf("nats: handler must return error")
	}

	return nil
}

func validateDirectiveFields(fields []directive.Field) error {
	for _, f := range fields {
		switch f.Key {
		case "mode":
			if f.Value != "at-most-once" && f.Value != "at-least-once" {
				return fmt.Errorf("nats: invalid mode %q (expected at-most-once or at-least-once)", f.Value)
			}

		case "ackwait":
			d, err := time.ParseDuration(f.Value)
			if err != nil || d <= 0 {
				return fmt.Errorf("nats: invalid ackwait %q", f.Value)
			}

		case "maxinflight":
			n, err := strconv.Atoi(f.Value)
			if err != nil || n <= 0 {
				return fmt.Errorf("nats: invalid maxinflight %q", f.Value)
			}

		case "queue", "stream":
			if strings.TrimSpace(f.Value) == "" {
				return fmt.Errorf("nats: %s cannot be empty", f.Key)
			}

		case "subjects":
			subjects := strings.Split(f.Value, ",")
			if len(subjects) == 0 {
				return fmt.Errorf("nats: subjects cannot be empty")
			}
			for _, s := range subjects {
				s = strings.TrimSpace(s)
				if err := validateNATSSubject(s); err != nil {
					return fmt.Errorf("nats: invalid subjects entry %q: %w", s, err)
				}
			}

		default:
			return fmt.Errorf("nats: unknown field %q", f.Key)
		}
	}
	return nil
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

func validateNATSSubject(subject string) error {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return fmt.Errorf("subject cannot be empty")
	}
	if !natsSubjectRe.MatchString(subject) {
		return fmt.Errorf("subject contains invalid characters")
	}

	tokens := strings.Split(subject, ".")
	for i, tok := range tokens {
		if tok == "" {
			return fmt.Errorf("subject cannot contain empty tokens")
		}
		if strings.ContainsAny(tok, " \t\n\r") {
			return fmt.Errorf("subject token %q contains whitespace", tok)
		}
		if strings.Contains(tok, ">") && tok != ">" {
			return fmt.Errorf("token %q contains invalid > wildcard usage", tok)
		}
		if strings.Contains(tok, "*") && tok != "*" {
			return fmt.Errorf("token %q contains invalid * wildcard usage", tok)
		}
		if tok == ">" && i != len(tokens)-1 {
			return fmt.Errorf("> wildcard is only allowed as the final token")
		}
	}
	return nil
}
