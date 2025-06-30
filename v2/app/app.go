package app

import (
	"github.com/circularing/encore/pkg/option"
	"github.com/circularing/encore/pkg/paths"
	"github.com/circularing/encore/v2/app/apiframework"
	"github.com/circularing/encore/v2/internals/parsectx"
	"github.com/circularing/encore/v2/internals/perr"
	"github.com/circularing/encore/v2/internals/pkginfo"
	"github.com/circularing/encore/v2/parser"
	"github.com/circularing/encore/v2/parser/apis/api"
	"github.com/circularing/encore/v2/parser/apis/middleware"
	"github.com/circularing/encore/v2/parser/resource"
	"github.com/circularing/encore/v2/parser/resource/usage"
)

// Desc describes an Encore application.
type Desc struct {
	Errs *perr.List

	BuildInfo  parsectx.BuildInfo
	MainModule *pkginfo.Module
	Parse      *parser.Result
	Services   []*Service
	Gateways   []*Gateway

	// Framework describes API Framework-specific application-global data.
	Framework option.Option[*apiframework.AppDesc]

	// ResourceUsageOutsideServices describes resources that are used outside of a service.
	ResourceUsageOutsideServices map[resource.Resource][]usage.Usage
}

// MatchingMiddleware reports which middleware applies to the given RPC,
// and the order they apply in.
func (d *Desc) MatchingMiddleware(ep *api.Endpoint) []*middleware.Middleware {
	var matches []*middleware.Middleware

	// Ensure middleware ordering is preserved.

	// First add global middleware.
	d.Framework.ForAll(func(fw *apiframework.AppDesc) {
		for _, mw := range fw.GlobalMiddleware {
			if mw.Target.ContainsAny(ep.Tags) {
				matches = append(matches, mw)
			}
		}
	})

	// Then add service-specific middleware.
	if svc, ok := d.ServiceForPath(ep.File.Pkg.FSPath); ok {
		svc.Framework.ForAll(func(fw *apiframework.ServiceDesc) {
			for _, mw := range fw.Middleware {
				if mw.Target.ContainsAny(ep.Tags) {
					matches = append(matches, mw)
				}
			}
		})
	}

	return matches
}

// MatchingGlobalMiddleware reports which global middleware applies to the given RPC,
// and the order they apply in.
func (d *Desc) MatchingGlobalMiddleware(ep *api.Endpoint) []*middleware.Middleware {
	var matches []*middleware.Middleware
	d.Framework.ForAll(func(fw *apiframework.AppDesc) {
		for _, mw := range fw.GlobalMiddleware {
			if mw.Target.ContainsAny(ep.Tags) {
				matches = append(matches, mw)
			}
		}
	})
	return matches
}

// ValidateAndDescribe validates the application and computes the
// application description.
func ValidateAndDescribe(pc *parsectx.Context, result *parser.Result) *Desc {
	defer pc.Trace("app.ValidateAndDescribe").Done()

	// First we want to discover the service layout
	services := discoverServices(pc, result)

	// We always have a default API gateway, for now.
	gateways := []*Gateway{{EncoreName: "api-gateway"}}

	// Now we can configure the API framework by combining the service information
	// with the parse results.
	framework := configureAPIFramework(pc, services, result)

	desc := &Desc{
		Errs:                         pc.Errs,
		BuildInfo:                    pc.Build,
		MainModule:                   result.MainModule(),
		Parse:                        result,
		Services:                     services,
		Gateways:                     gateways,
		Framework:                    framework,
		ResourceUsageOutsideServices: make(map[resource.Resource][]usage.Usage),
	}

	// Find each services infra binds and usage.
	desc.locateResourceBinds(result)
	desc.locateResourceUsage(result)

	// Run the application-level validations against the application description.
	desc.validate(pc, result)

	return desc
}

// ServiceForPath returns the service a given folder path belongs to, if any.
func (d *Desc) ServiceForPath(path paths.FS) (*Service, bool) {
	for _, svc := range d.Services {
		if path.HasPrefix(svc.FSRoot) {
			return svc, true
		}
	}
	return nil, false
}
