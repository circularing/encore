package apiframework

import (
	"github.com/circularing/encore/pkg/option"
	"github.com/circularing/encore/v2/internals/pkginfo"
	"github.com/circularing/encore/v2/parser/apis/api"
	"github.com/circularing/encore/v2/parser/apis/authhandler"
	"github.com/circularing/encore/v2/parser/apis/middleware"
	"github.com/circularing/encore/v2/parser/apis/servicestruct"
)

// AppDesc describes an Encore Framework-based application.
type AppDesc struct {
	// GlobalMiddleware is the list of application-global middleware.
	GlobalMiddleware []*middleware.Middleware

	// AuthHandler defines the application's auth handler, if any.
	AuthHandler option.Option[*authhandler.AuthHandler]
}

// ServiceDesc describes an Encore Framework-based service.
//
// For code that deals with general services, use *service.Service instead of this type.
type ServiceDesc struct {
	// Middleware are the service-specific middleware
	Middleware []*middleware.Middleware

	// RootPkg is the root package of the service.
	RootPkg *pkginfo.Package

	// Endpoints are the endpoints defined in this service.
	Endpoints []*api.Endpoint

	// ServiceStruct defines the service's service struct, if any.
	ServiceStruct option.Option[*servicestruct.ServiceStruct]
}
