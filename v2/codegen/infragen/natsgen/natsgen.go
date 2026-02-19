package natsgen

import (
	"fmt"
	"sort"
	"strings"

	. "github.com/dave/jennifer/jen"

	"encr.dev/v2/codegen"
	"encr.dev/v2/internals/pkginfo"
	"encr.dev/v2/internals/schema"
	"encr.dev/v2/parser/apis/nats"
)

func Gen(gen *codegen.Generator, pkg *pkginfo.Package, subs []*nats.Subscription) {
	if len(subs) == 0 {
		return
	}

	sort.SliceStable(subs, func(i, j int) bool {
		if subs[i].Subject != subs[j].Subject {
			return subs[i].Subject < subs[j].Subject
		}
		return subs[i].Name < subs[j].Name
	})

	file := gen.File(pkg, "nats_pubsub")
	file.Add(Var().Id("encoreInternalNATSClient").Op("=").Qual("encr.dev/v2/parser/plugin/natspubsub", "NewClient").Call())

	topicsByKey := make(map[string]string)
	msgTypeBySubject := make(map[string]string)

	for _, sub := range subs {
		if sub == nil || sub.MessageType == nil {
			continue
		}

		msgType := sub.MessageType.Decl.Name
		if existing, ok := msgTypeBySubject[sub.Subject]; ok && existing != msgType {
			gen.Errs.Addf(sub.Decl.Pos(), "nats subject %q is used with incompatible message types", sub.Subject)
			continue
		}
		msgTypeBySubject[sub.Subject] = msgType
		typeArg := messageTypeArg(sub)

		streamName, streamSubjects := effectiveStreamDefaults(sub)
		maxInflight := effectiveMaxInflight(sub)
		topicKey := sub.Subject + "|" + streamName + "|" + strings.Join(streamSubjects, ",") + "|" + string(sub.NATS.Mode) +
			"|" + fmt.Sprint(sub.NATS.AckWait.Nanoseconds()) + "|" + fmt.Sprint(maxInflight) + "|" + sub.NATS.QueueGroup
		topicVar, ok := topicsByKey[topicKey]
		if !ok {
			topicVar = fmt.Sprintf("encoreInternalNATSTopic%d", len(topicsByKey)+1)
			topicsByKey[topicKey] = topicVar

			opts := []Code{
				Qual("encr.dev/v2/parser/plugin/natspubsub", "WithStreamName").Call(Lit(streamName)),
				Qual("encr.dev/v2/parser/plugin/natspubsub", "WithStreamSubjects").Call(stringsToCodes(streamSubjects)...),
				Qual("encr.dev/v2/parser/plugin/natspubsub", "WithSubscriptionOptions").Call(
					Qual("time", "Duration").Call(Lit(sub.NATS.AckWait.Nanoseconds())),
					Lit(maxInflight),
					Lit(sub.NATS.QueueGroup),
				),
			}
			if sub.NATS.Mode == nats.ModeAtMostOnce {
				opts = append(opts, Qual("encr.dev/v2/parser/plugin/natspubsub", "WithAtMostOnce").Call())
			} else {
				opts = append(opts, Qual("encr.dev/v2/parser/plugin/natspubsub", "WithAtLeastOnce").Call())
			}

			file.Add(
				Var().Id(topicVar).Op("=").Qual("encr.dev/v2/parser/plugin/natspubsub", "NewTopic").Types(
					gen.Util.Type(typeArg),
				).Call(append([]Code{
					Id("encoreInternalNATSClient"),
					Lit(sub.Subject),
				}, opts...)...),
			)
		}

		file.Add(
			Func().Id("init").Params().Block(
				If(
					Err().Op(":=").Id(topicVar).Dot("Subscribe").Call(
						Lit(uniqueName(sub.Name, sub.HandlerName)),
						Qual("encr.dev/v2/parser/plugin/natspubsub", "SubscriptionConfig").Types(
							gen.Util.Type(typeArg),
						).Values(Dict{
							Id("Handler"): Id(sub.HandlerName),
						}),
					),
					Err().Op("!=").Nil(),
				).Block(
					Panic(Err()),
				),
			),
		)
	}
}

func uniqueName(base, handler string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "subscription"
	}
	handler = strings.TrimSpace(handler)
	if handler == "" {
		return base
	}
	return base + "-" + strings.ToLower(handler)
}

func effectiveStreamDefaults(sub *nats.Subscription) (string, []string) {
	stream := strings.TrimSpace(sub.NATS.StreamName)
	subjects := append([]string(nil), sub.NATS.StreamSubjects...)
	if stream != "" && len(subjects) > 0 {
		return stream, subjects
	}

	subject := strings.TrimSpace(sub.Subject)
	if subject == "" {
		subject = "events.default"
	}
	if stream == "" {
		stream = "encore_nats_" + sanitizeIdent(strings.ReplaceAll(subject, ".", "_"))
	}
	if len(subjects) == 0 {
		subjects = []string{subject}
	}
	return stream, subjects
}

func subjectRoot(subject string) string {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return "events"
	}
	parts := strings.Split(subject, ".")
	if len(parts) == 0 || parts[0] == "" || parts[0] == "*" || parts[0] == ">" {
		return "events"
	}
	return parts[0]
}

func sanitizeIdent(in string) string {
	var b strings.Builder
	for _, r := range in {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if r >= 'A' && r <= 'Z' {
			b.WriteRune(r + ('a' - 'A'))
		} else {
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "events"
	}
	return out
}

func stringsToCodes(vals []string) []Code {
	out := make([]Code, 0, len(vals))
	for _, v := range vals {
		out = append(out, Lit(v))
	}
	return out
}

func messageTypeArg(sub *nats.Subscription) schema.Type {
	if sub == nil || sub.MessageType == nil {
		return nil
	}
	typ := sub.MessageType.ToType()
	if ptr, ok := typ.(schema.PointerType); ok {
		return ptr.Elem
	}
	return typ
}

func effectiveMaxInflight(sub *nats.Subscription) int {
	if sub == nil {
		return 1
	}
	if !sub.NATS.MaxInflightSet {
		return 1
	}
	if sub.NATS.MaxInflight <= 0 {
		return 1
	}
	return sub.NATS.MaxInflight
}

