package apigen

import (
	"maps"

	"encore.dev/appruntime/exported/config"
	"github.com/circularing/encore/pkg/option"
	"github.com/circularing/encore/pkg/paths"
	"github.com/circularing/encore/v2/app"
	"github.com/circularing/encore/v2/codegen"
	"github.com/circularing/encore/v2/codegen/apigen/authhandlergen"
	"github.com/circularing/encore/v2/codegen/apigen/endpointgen"
	"github.com/circularing/encore/v2/codegen/apigen/maingen"
	"github.com/circularing/encore/v2/codegen/apigen/middlewaregen"
	"github.com/circularing/encore/v2/codegen/apigen/servicestructgen"
	"github.com/circularing/encore/v2/codegen/apigen/userfacinggen"
	"github.com/circularing/encore/v2/internals/pkginfo"
	"github.com/circularing/encore/v2/parser/apis/api"
	"github.com/circularing/encore/v2/parser/apis/authhandler"
	"github.com/circularing/encore/v2/parser/apis/middleware"
)

type Params struct {
	Gen           *codegen.Generator
	Desc          *app.Desc
	MainModule    *pkginfo.Module
	RuntimeModule *pkginfo.Module

	CompilerVersion string
	AppRevision     string
	AppUncommitted  bool

	Test option.Option[codegen.TestConfig]

	ExecScriptMainPkg option.Option[paths.Pkg]
}

func Process(p Params) *config.Static {
	gp := maingen.GenParams{
		Gen:               p.Gen,
		Desc:              p.Desc,
		MainModule:        p.MainModule,
		RuntimeModule:     p.RuntimeModule,
		Test:              p.Test,
		ExecScriptMainPkg: p.ExecScriptMainPkg,

		CompilerVersion: p.CompilerVersion,
		AppRevision:     p.AppRevision,
		AppUncommitted:  p.AppUncommitted,

		APIHandlers:    make(map[*api.Endpoint]*codegen.VarDecl),
		Middleware:     make(map[*middleware.Middleware]*codegen.VarDecl),
		ServiceStructs: make(map[*app.Service]*codegen.VarDecl),

		// Set below
		AuthHandler: option.None[*codegen.VarDecl](),
	}

	if fw, ok := p.Desc.Framework.Get(); ok {

		svcStructBySvc := make(map[string]*codegen.VarDecl)

		for _, svc := range p.Desc.Services {
			var svcStruct option.Option[*codegen.VarDecl]

			var svcMiddleware map[*middleware.Middleware]*codegen.VarDecl
			if svcDesc, ok := svc.Framework.Get(); ok {
				if ss, ok := svcDesc.ServiceStruct.Get(); ok {
					decl := servicestructgen.Gen(p.Gen, svc, ss)
					gp.ServiceStructs[svc] = decl
					svcStruct = option.Some(decl)
					svcStructBySvc[svc.Name] = decl
				}

				svcMiddleware = middlewaregen.Gen(p.Gen, svcDesc.Middleware, svcStruct)
				maps.Copy(gp.Middleware, svcMiddleware)
			}

			eps := endpointgen.Gen(p.Gen, p.Desc, svc, svcStruct, svcMiddleware)
			maps.Copy(gp.APIHandlers, eps)

			// Generate user-facing code with the implementation in place.
			userfacinggen.Gen(p.Gen, svc, svcStruct)
		}

		gp.AuthHandler = option.Map(fw.AuthHandler, func(ah *authhandler.AuthHandler) *codegen.VarDecl {
			var svcStruct option.Option[*codegen.VarDecl]
			if svc, ok := p.Desc.ServiceForPath(ah.Decl.File.FSPath); ok {
				svcStruct = option.AsOptional(svcStructBySvc[svc.Name])
			}
			return authhandlergen.Gen(p.Gen, p.Desc, ah, svcStruct)
		})

		mws := middlewaregen.Gen(p.Gen, fw.GlobalMiddleware, option.None[*codegen.VarDecl]())
		maps.Copy(gp.Middleware, mws)
	}

	return maingen.Gen(gp)
}
