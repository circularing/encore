package infragen

import (
	"encr.dev/pkg/fns"
	"encr.dev/pkg/option"
	"encr.dev/pkg/paths"
	"encr.dev/v2/app"
	"encr.dev/v2/codegen"
	"encr.dev/v2/codegen/infragen/cachegen"
	"encr.dev/v2/codegen/infragen/configgen"
	"encr.dev/v2/codegen/infragen/metricsgen"
	"encr.dev/v2/codegen/infragen/natsgen"
	"encr.dev/v2/codegen/infragen/pubsubgen"
	"encr.dev/v2/codegen/infragen/secretsgen"
	"encr.dev/v2/internals/pkginfo"
	"encr.dev/v2/parser/apis/nats"
	"encr.dev/v2/parser/infra/caches"
	"encr.dev/v2/parser/infra/config"
	"encr.dev/v2/parser/infra/metrics"
	"encr.dev/v2/parser/infra/pubsub"
	"encr.dev/v2/parser/infra/secrets"
	"encr.dev/v2/parser/resource"
)

func Process(gg *codegen.Generator, appDesc *app.Desc) {
	type groupKey struct {
		pkg      paths.Pkg
		resource string
	}

	groups := make(map[groupKey][]resource.Resource)
	pkgMap := make(map[paths.Pkg]*pkginfo.Package)
	for _, r := range appDesc.Parse.Resources() {
		// Group by package.
		var pkg *pkginfo.Package
		var resourceType string
		switch r := r.(type) {
		case *caches.Keyspace:
			pkg = r.File.Pkg
			resourceType = "cache-keyspace"
		case *metrics.Metric:
			pkg = r.File.Pkg
			resourceType = "metric"
		case *secrets.Secrets:
			pkg = r.File.Pkg
			resourceType = "secrets"
		case *pubsub.Subscription:
			pkg = r.File.Pkg
			resourceType = "pubsub-subscription"
		case *nats.Subscription:
			pkg = r.File.Pkg
			resourceType = "nats-subscription"
		case *config.Load:
			pkg = r.File.Pkg
			resourceType = "config-load"
		default:
			continue
		}

		key := groupKey{pkg: pkg.ImportPath, resource: resourceType}
		groups[key] = append(groups[key], r)
		pkgMap[pkg.ImportPath] = pkg
	}

	for key, resources := range groups {
		pkg := pkgMap[key.pkg]
		switch key.resource {
		case "cache-keyspace":
			cachegen.GenKeyspace(gg, pkg, fns.Map(resources, func(r resource.Resource) *caches.Keyspace {
				return r.(*caches.Keyspace)
			}))
		case "metric":
			metricsgen.Gen(gg, pkg, fns.Map(resources, func(r resource.Resource) *metrics.Metric {
				return r.(*metrics.Metric)
			}))
		case "pubsub-subscription":
			pubsubgen.Gen(gg, pkg, appDesc, fns.Map(resources, func(r resource.Resource) *pubsub.Subscription {
				return r.(*pubsub.Subscription)
			}))
		case "nats-subscription":
			natsgen.Gen(gg, pkg, fns.Map(resources, func(r resource.Resource) *nats.Subscription {
				return r.(*nats.Subscription)
			}))
		case "secrets":
			svc, _ := appDesc.ServiceForPath(pkg.FSPath)
			secretsgen.Gen(gg, option.AsOptional(svc), pkg, fns.Map(resources, func(r resource.Resource) *secrets.Secrets {
				return r.(*secrets.Secrets)
			}))
		case "config-load":
			svc, ok := appDesc.ServiceForPath(pkg.FSPath)
			if !ok {
				gg.Errs.Addf(resources[0].(*config.Load).AST.Pos(), "config loads must be declared in a service package")
				continue
			}

			configgen.Gen(gg, svc, pkg, fns.Map(resources, func(r resource.Resource) *config.Load {
				return r.(*config.Load)
			}))
		}
	}
}
